package test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"

	"github.com/statoon54/mailhive/internal/domain"
)

// SkipIfNoPostgres vérifie que la variable TEST_DATABASE_URL est définie.
func SkipIfNoPostgres(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL non défini, test d'intégration ignoré")
	}
}

// SkipIfNoRedis vérifie que la variable TEST_REDIS_ADDR est définie.
func SkipIfNoRedis(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_REDIS_ADDR") == "" {
		t.Skip("TEST_REDIS_ADDR non défini, test d'intégration ignoré")
	}
}

// RedisClient crée un client Redis de test.
func RedisClient(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 15})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis non disponible : %v", err)
	}
	t.Cleanup(func() {
		_ = client.FlushDB(context.Background()).Err()
		_ = client.Close()
	})
	return client
}

// JWTSecret est la clé secrète utilisée dans les tests.
const JWTSecret = "test-jwt-secret-key-for-integration-tests"

// GenerateTestToken génère un JWT de test pour un tenant.
func GenerateTestToken(tenantID uuid.UUID, slug, role string) string {
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"slug":      slug,
		"role":      role,
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		log.Fatalf("erreur de génération du token de test : %v", err)
	}
	return signed
}

// DoRequest envoie une requête HTTP au serveur Echo et retourne le recorder.
func DoRequest(e *echo.Echo, method, path string, body any, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// DecodeResponse décode la réponse JSON.
func DecodeResponse(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
		t.Fatalf("erreur de décodage de la réponse : %v (body: %s)", err, rec.Body.String())
	}
}

// AssertStatus vérifie le code HTTP attendu.
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Errorf("code HTTP attendu %d, obtenu %d (body: %s)", expected, rec.Code, rec.Body.String())
	}
}

// TestTenant crée un tenant de test.
func TestTenant(id uuid.UUID) *domain.Tenant {
	return &domain.Tenant{
		ID:       id,
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		APIKey:   "test-api-key",
		IsActive: true,
		Settings: domain.TenantSettings{
			RateLimit:        10,
			RateBurst:        10,
			MaxDestinataires: 100,
			DefaultPriority:  domain.MailPriorityDefault,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestTemplate crée un template de test.
func TestTemplate(id, tenantID uuid.UUID) *domain.Template {
	return &domain.Template{
		ID:          id,
		TenantID:    tenantID,
		Name:        "Test Template",
		Slug:        "test-template",
		SubjectTmpl: "Hello {{.name}}",
		TextBody:    "Hello {{.name}}, welcome!",
		HTMLBody:    "<html><body><p>Hello {{.name}}</p></body></html>",
		Variables:   map[string]string{"name": "Nom du destinataire"},
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SpamRequest retourne un body de requête avec des données de test.
func SpamRequest(data map[string]string) map[string]any {
	return map[string]any{"data": data}
}

// StatusFromBody extrait le code status et le message de la réponse.
func StatusFromBody(rec *httptest.ResponseRecorder) (int, string) {
	return rec.Code, rec.Body.String()
}

// MailWithTags crée un mail de test avec des tags.
func MailWithTags(tenantID uuid.UUID, tags []string) *domain.Mail {
	id, _ := uuid.NewV7()
	score := float32(0)
	return &domain.Mail{
		ID:        id,
		TenantID:  tenantID,
		FromEmail: "test@example.com",
		Subject:   "Test mail",
		Status:    domain.MailStatusPending,
		Priority:  domain.MailPriorityDefault,
		SpamScore: &score,
		Tags:      tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestMail insère un mail via l'interface HTTP de test.
func CreateTestMail(e *echo.Echo, token string, body any) *httptest.ResponseRecorder {
	return DoRequest(e, http.MethodPost, "/api/v1/mails", body, token)
}
