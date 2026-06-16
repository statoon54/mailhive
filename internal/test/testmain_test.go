package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/config"
	"github.com/statoon54/mailhive/migrations"
)

// sharedPool est le pool partagé par tous les tests du package.
var sharedPool *pgxpool.Pool

// TestMain démarre un seul conteneur PostgreSQL pour tous les tests du package.
func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("mailhive_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("impossible de démarrer le conteneur PostgreSQL : %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		log.Fatalf("impossible de récupérer l'hôte : %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		log.Fatalf("impossible de récupérer le port : %v", err)
	}

	dbCfg := config.DBConfig{
		Host:     host,
		Port:     port.Int(),
		User:     "test",
		Password: "test",
		Name:     "mailhive_test",
		SSLMode:  "disable",
	}

	// Appliquer les migrations
	if err := postgres.RunMigrations(dbCfg, migrations.FS); err != nil {
		log.Fatalf("erreur de migration : %v", err)
	}

	// Créer le pool partagé
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name, dbCfg.SSLMode,
	)
	sharedPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("erreur de connexion au pool : %v", err)
	}

	// Exécuter les tests
	code := m.Run()

	// Nettoyage
	sharedPool.Close()
	if err := pgContainer.Terminate(ctx); err != nil {
		log.Printf("erreur de terminaison du conteneur : %v", err)
	}

	os.Exit(code)
}

// PGPool retourne le pool partagé et nettoie les tables avant chaque test.
func PGPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if sharedPool == nil {
		t.Fatal("sharedPool est nil — TestMain n'a pas initialisé le conteneur PostgreSQL")
	}
	CleanDB(t, sharedPool)
	return sharedPool
}

// CleanDB vide toutes les tables dans le bon ordre (respect des FKs).
func CleanDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	tables := []string{
		"mail_recipients",
		"mails",
		"smtp_configs",
		"mail_templates",
		"audit_logs",
		"tenants",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Fatalf("erreur de nettoyage de la table %s : %v", table, err)
		}
	}

	// Réinitialiser le branding
	_, _ = pool.Exec(ctx, `UPDATE app_branding SET app_title = 'MailHive', app_subtitle = 'Gestion des mails', timezone = 'Europe/Paris', logo_data = '', logo_content_type = '', updated_at = NOW() WHERE id = 1`)
}
