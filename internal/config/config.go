package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contient toute la configuration de l'application, chargée depuis les variables d'environnement.
type Config struct {
	Admin      AdminConfig
	Encryption EncryptionConfig
	SMTPMode   string `env:"SMTP_MODE" envDefault:"real"`
	API        APIConfig
	JWT        JWTConfig
	Log        LogConfig
	Redis      RedisConfig
	Mailpit    MailpitConfig
	LLM        LLMConfig
	Blob       BlobConfig
	DB         DBConfig
	Worker     WorkerConfig
	RateLimit  RateLimitConfig
}

// BlobConfig contient la configuration du stockage des pièces jointes.
// Backend "postgres" (défaut) : contenu dans la table attachment_blobs.
// Backend "s3" : object store compatible S3 (SeaweedFS, MinIO, S3, R2).
type BlobConfig struct {
	Backend     string `env:"BLOB_BACKEND"       envDefault:"postgres"` // postgres | s3
	S3Endpoint  string `env:"BLOB_S3_ENDPOINT"   envDefault:""`         // ex: localhost:8333 (SeaweedFS) ou s3.amazonaws.com
	S3Bucket    string `env:"BLOB_S3_BUCKET"     envDefault:"mailhive-attachments"`
	S3AccessKey string `env:"BLOB_S3_ACCESS_KEY" envDefault:""`
	S3SecretKey string `env:"BLOB_S3_SECRET_KEY" envDefault:""`
	S3Region    string `env:"BLOB_S3_REGION"     envDefault:"us-east-1"`
	S3UseSSL    bool   `env:"BLOB_S3_USE_SSL"    envDefault:"false"`
}

// LogConfig contient la configuration du logger structuré (slog).
type LogConfig struct {
	Level  string `env:"LOG_LEVEL"  envDefault:"info"`
	Format string `env:"LOG_FORMAT" envDefault:"text"`
}

// LLMConfig contient la configuration du fournisseur LLM pour la génération de contenu.
type LLMConfig struct {
	Provider string `env:"LLM_PROVIDER" envDefault:"ollama"`
	BaseURL  string `env:"LLM_BASE_URL" envDefault:"http://localhost:11434"`
	Model    string `env:"LLM_MODEL"    envDefault:"llama3"`
	APIKey   string `env:"LLM_API_KEY"  envDefault:""`
}

// APIConfig contient la configuration du serveur HTTP.
type APIConfig struct {
	Host string `env:"API_HOST" envDefault:"0.0.0.0"`
	Port int    `env:"API_PORT" envDefault:"8080"`
}

// DBConfig contient la configuration de PostgreSQL.
type DBConfig struct {
	HealthCheckPeriod time.Duration `env:"DB_HEALTH_CHECK_PERIOD" envDefault:"30s"`
	MaxConnLifetime   time.Duration `env:"DB_MAX_CONN_LIFETIME"   envDefault:"30m"`
	Host              string        `env:"DB_HOST"                envDefault:"localhost"`
	Name              string        `env:"DB_NAME"                envDefault:"mailhive"`
	Password          string        `env:"DB_PASSWORD"            envDefault:"mailhive_secret"`
	SSLMode           string        `env:"DB_SSL_MODE"            envDefault:"disable"`
	User              string        `env:"DB_USER"                envDefault:"mailhive"`
	// MaxConns doit être ≥ WORKER_CONCURRENCY : en mode worker (et a fortiori en mode
	// serve où l'API partage le pool), chaque goroutine peut frapper une phase DB
	// simultanément. Un pool plus petit que la concurrence plafonne le débit. Voir PoolWarning.
	MaxConns int `env:"DB_MAX_CONNS"           envDefault:"50"`
	MinConns int `env:"DB_MIN_CONNS"           envDefault:"5"`
	Port     int `env:"DB_PORT"                envDefault:"5432"`
}

// RedisConfig contient la configuration de Redis.
type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR"     envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB"       envDefault:"0"`
}

// JWTConfig contient la configuration JWT.
type JWTConfig struct {
	Expiration time.Duration `env:"JWT_EXPIRATION"      envDefault:"24h"`
	Secret     string        `env:"JWT_SECRET,required"`
}

// AdminConfig contient la clé API admin.
type AdminConfig struct {
	APIKey string `env:"ADMIN_API_KEY,required"`
}

// EncryptionConfig contient la clé de chiffrement des mots de passe SMTP.
type EncryptionConfig struct {
	Key string `env:"ENCRYPTION_KEY,required"`
}

// WorkerConfig contient la configuration du worker Asynq.
type WorkerConfig struct {
	Concurrency   int `env:"WORKER_CONCURRENCY"    envDefault:"50"`
	QueueCritical int `env:"WORKER_QUEUE_CRITICAL" envDefault:"6"`
	QueueDefault  int `env:"WORKER_QUEUE_DEFAULT"  envDefault:"3"`
	QueueLow      int `env:"WORKER_QUEUE_LOW"      envDefault:"1"`
}

// RateLimitConfig contient la configuration par défaut du rate limiting.
type RateLimitConfig struct {
	DefaultRate  float64 `env:"DEFAULT_RATE_LIMIT" envDefault:"100"`
	DefaultBurst int     `env:"DEFAULT_RATE_BURST" envDefault:"200"`
}

// MailpitConfig contient la configuration du serveur SMTP de test Mailpit.
type MailpitConfig struct {
	FromEmail string `env:"MAILPIT_FROM"      envDefault:"noreply@mailhive.dev"`
	FromName  string `env:"MAILPIT_FROM_NAME" envDefault:"MailHive"`
	Host      string `env:"MAILPIT_HOST"      envDefault:"localhost"`
	Port      int    `env:"MAILPIT_PORT"      envDefault:"1025"`
}

// Load charge la configuration depuis les variables d'environnement.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// PoolWarning retourne un avertissement si le pool de connexions PostgreSQL est
// trop petit pour la concurrence du worker, sinon une chaîne vide.
//
// pgxpool acquiert/libère une connexion par requête : un goroutine worker ne tient
// pas de connexion pendant l'envoi SMTP. Mais si DB_MAX_CONNS < WORKER_CONCURRENCY,
// les phases DB simultanées (mises à jour de statut, chargement tenant/SMTP)
// entrent en contention et plafonnent le débit.
func (c *Config) PoolWarning() string {
	if c.DB.MaxConns >= c.Worker.Concurrency {
		return ""
	}
	return fmt.Sprintf(
		"pool DB sous-dimensionné : DB_MAX_CONNS=%d < WORKER_CONCURRENCY=%d ; "+
			"les phases DB du worker vont entrer en contention. "+
			"Recommandation : DB_MAX_CONNS ≥ %d.",
		c.DB.MaxConns, c.Worker.Concurrency, c.Worker.Concurrency,
	)
}
