// cmd/mailhive — Point d'entrée unique de MailHive.
//
// Usage :
//
//	mailhive              # Lance API + Worker (défaut)
//	mailhive serve        # Lance API + Worker
//	mailhive api          # Lance uniquement l'API
//	mailhive worker       # Lance uniquement le Worker
//	mailhive migrate      # Exécute les migrations
//	mailhive migrate-down # Rollback dernière migration
//	mailhive version      # Affiche la version
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	goredis "github.com/redis/go-redis/v9"

	openapi "github.com/statoon54/mailhive/api"
	"github.com/statoon54/mailhive/internal/adapter/blobstore"
	"github.com/statoon54/mailhive/internal/adapter/mailer"
	"github.com/statoon54/mailhive/internal/adapter/postgres"
	adaptRedis "github.com/statoon54/mailhive/internal/adapter/redis"
	"github.com/statoon54/mailhive/internal/config"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/frontend"
	"github.com/statoon54/mailhive/internal/handler"
	"github.com/statoon54/mailhive/internal/logging"
	mw "github.com/statoon54/mailhive/internal/middleware"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/worker"
	"github.com/statoon54/mailhive/migrations"
)

// version est la version du binaire, injectée au build via
// -ldflags "-X main.version=...". Vaut "dev" pour une compilation locale brute.
var version = "dev"

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	// La version ne nécessite aucune configuration : on la traite avant Load().
	if cmd == "version" {
		fmt.Printf("mailhive %s (%s %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("erreur de configuration", "err", err)
		os.Exit(1)
	}

	logging.Setup(cfg.Log.Level, cfg.Log.Format)

	switch cmd {
	case "migrate":
		runMigrate(cfg, false)
	case "migrate-down":
		runMigrate(cfg, true)
	case "serve":
		runAll(cfg)
	case "api":
		runAPI(cfg)
	case "worker":
		runWorker(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Commande inconnue : %s\nUsage : mailhive [serve|api|worker|migrate|migrate-down|version]\n", cmd)
		os.Exit(1)
	}
}

// fatal journalise une erreur de manière structurée puis termine le processus.
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// buildAttachmentService construit le service de pièces jointes selon le backend
// configuré (postgres par défaut, ou s3 compatible SeaweedFS/MinIO). Si le
// backend s3 est demandé mais injoignable, on échoue au démarrage (fatal)
// plutôt que de stocker silencieusement dans le mauvais backend.
func buildAttachmentService(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, component string) *service.AttachmentService {
	repo := postgres.NewAttachmentRepository(pool)

	if cfg.Blob.Backend == domain.AttachmentStorageS3 {
		store, err := blobstore.NewS3Store(ctx, cfg.Blob)
		if err != nil {
			fatal("erreur d'initialisation du stockage S3 des pièces jointes", "component", component, "err", err)
		}
		slog.Info("stockage des pièces jointes : S3", "component", component, "endpoint", cfg.Blob.S3Endpoint, "bucket", cfg.Blob.S3Bucket)
		return service.NewAttachmentService(repo, store, domain.AttachmentStorageS3)
	}

	slog.Info("stockage des pièces jointes : PostgreSQL", "component", component)
	return service.NewAttachmentService(repo, blobstore.NewPostgresStore(pool), domain.AttachmentStoragePostgres)
}

// ---------- Migrations ----------

func runMigrate(cfg *config.Config, down bool) {
	if down {
		slog.Info("rollback de la dernière migration")
		if err := postgres.RollbackMigrations(cfg.DB, migrations.FS); err != nil {
			fatal("erreur de rollback", "err", err)
		}
		slog.Info("rollback effectué")
	} else {
		slog.Info("exécution des migrations")
		if err := postgres.RunMigrations(cfg.DB, migrations.FS); err != nil {
			fatal("erreur de migration", "err", err)
		}
		slog.Info("migrations effectuées")
	}
}

// ---------- Serve (API + Worker) ----------

func runAll(cfg *config.Config) {
	slog.Info("démarrage en mode complet (API + Worker)")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Infra commune
	pool, redisClient := connectInfra(ctx, cfg)
	defer pool.Close()
	defer func() { _ = redisClient.Close() }()

	// Migrations auto
	slog.Info("exécution des migrations")
	if err := postgres.RunMigrations(cfg.DB, migrations.FS); err != nil {
		fatal("erreur de migration", "err", err)
	}

	// Seed des données par défaut (tenant admin ; + config Mailpit en mode dev)
	if cfg.SMTPMode != "simulation" {
		seedDefaults(ctx, pool, cfg)
	}

	var wg sync.WaitGroup

	// Lancer l'API
	wg.Add(1)
	go func() {
		defer wg.Done()
		startAPI(ctx, cfg, pool, redisClient)
	}()

	// Lancer le Worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		startWorker(ctx, cfg, pool, redisClient)
	}()

	wg.Wait()
	slog.Info("arrêt complet")
}

// ---------- API seule ----------

func runAPI(cfg *config.Config) {
	slog.Info("démarrage en mode API")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, redisClient := connectInfra(ctx, cfg)
	defer pool.Close()
	defer func() { _ = redisClient.Close() }()

	slog.Info("exécution des migrations")
	if err := postgres.RunMigrations(cfg.DB, migrations.FS); err != nil {
		fatal("erreur de migration", "err", err)
	}

	if cfg.SMTPMode != "simulation" {
		seedDefaults(ctx, pool, cfg)
	}

	startAPI(ctx, cfg, pool, redisClient)
}

// ---------- Worker seul ----------

func runWorker(cfg *config.Config) {
	slog.Info("démarrage en mode Worker")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, redisClient := connectInfra(ctx, cfg)
	defer pool.Close()
	defer func() { _ = redisClient.Close() }()

	startWorker(ctx, cfg, pool, redisClient)
}

// ---------- Infrastructure ----------

func connectInfra(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, *goredis.Client) {
	pool, err := postgres.NewPool(ctx, cfg.DB)
	if err != nil {
		fatal("erreur de connexion PostgreSQL", "err", err)
	}

	redisClient, err := adaptRedis.NewClient(ctx, cfg.Redis)
	if err != nil {
		fatal("erreur de connexion Redis", "err", err)
	}

	return pool, redisClient
}

// ---------- Démarrage API ----------

func startAPI(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, redisClient *goredis.Client) {
	// Repositories
	tenantRepo := postgres.NewTenantRepository(pool)
	smtpRepo := postgres.NewSMTPConfigRepository(pool)
	mailRepo := postgres.NewMailRepository(pool)
	templateRepo := postgres.NewTemplateRepository(pool)
	brandingRepo := postgres.NewBrandingRepository(pool)
	auditLogRepo := postgres.NewAuditLogRepository(pool)

	// Sender
	var sender port.MailSender
	if cfg.SMTPMode == "simulation" {
		sender = mailer.NewLogSender()
		slog.Info("mode SMTP configuré", "component", "api", "mode", "simulation")
	} else {
		sender = mailer.NewSender()
		slog.Info("mode SMTP configuré", "component", "api", "mode", "réel")
	}

	// Asynq client + inspector
	asynqClient := worker.NewAsynqClient(cfg.Redis.Addr)
	defer func() { _ = asynqClient.Close() }()
	asynqInspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: cfg.Redis.Addr})
	defer func() { _ = asynqInspector.Close() }()
	queueClient := worker.NewQueueClient(asynqClient, asynqInspector)

	// Services
	authService := service.NewAuthService(tenantRepo, cfg.JWT, cfg.Admin.APIKey)
	tenantService := service.NewTenantService(tenantRepo)
	templateService := service.NewTemplateService(templateRepo)
	smtpService, err := service.NewSMTPConfigService(smtpRepo, sender, cfg.Encryption.Key)
	if err != nil {
		fatal("erreur service SMTP", "component", "api", "err", err)
	}
	analysisService := service.NewAnalysisService(templateRepo)
	attachmentService := buildAttachmentService(ctx, cfg, pool, "api")
	mailService := service.NewMailService(mailRepo, smtpRepo, templateRepo, tenantRepo, queueClient, analysisService, attachmentService)
	brandingService := service.NewBrandingService(brandingRepo)
	auditLogService := service.NewAuditLogService(auditLogRepo)
	llmService := service.NewLLMService(cfg.LLM)

	// Charger la timezone
	if b, err := brandingService.Get(ctx); err == nil {
		service.ApplyTimezone(b.Timezone)
	}

	// Handlers
	healthHandler := handler.NewHealthHandler(pool, redisClient)
	queueHandler := handler.NewQueueHandler(asynqInspector)
	authHandler := handler.NewAuthHandler(authService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	mailHandler := handler.NewMailHandler(mailService)
	templateHandler := handler.NewTemplateHandler(templateService)
	smtpConfigHandler := handler.NewSMTPConfigHandler(smtpService)
	brandingHandler := handler.NewBrandingHandler(brandingService)
	auditLogHandler := handler.NewAuditLogHandler(auditLogService)
	llmHandler := handler.NewLLMHandler(llmService)
	analysisHandler := handler.NewAnalysisHandler(analysisService, redisClient)

	// Echo
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.CORS("*"))
	e.Use(mw.RequestID())

	// Frontend embarqué (SPA)
	if err := frontend.RegisterRoutes(e); err != nil {
		fatal("erreur d'enregistrement du frontend", "component", "api", "err", err)
	}

	// Swagger
	e.GET("/swagger/openapi.yaml", func(c *echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/yaml")
		return c.Blob(http.StatusOK, "application/yaml", openapi.OpenAPISpec)
	})
	swaggerHTML := `<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8">
  <title>MailHive — Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger/openapi.yaml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`
	e.GET("/swagger", func(c *echo.Context) error {
		return c.HTML(http.StatusOK, swaggerHTML)
	})
	e.GET("/swagger/", func(c *echo.Context) error {
		return c.HTML(http.StatusOK, swaggerHTML)
	})

	// Asynqmon
	mon := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitoring",
		RedisConnOpt: asynq.RedisClientOpt{Addr: cfg.Redis.Addr},
	})
	e.Any("/monitoring/*", echo.WrapHandler(mon))
	e.Any("/monitoring", echo.WrapHandler(mon))

	// Routes API v1
	api := e.Group("/api/v1")
	api.POST("/auth/token", authHandler.GenerateToken)
	api.POST("/auth/refresh", authHandler.RefreshToken)
	api.GET("/health", healthHandler.Health)
	api.GET("/branding", brandingHandler.Get)
	api.GET("/branding/logo", brandingHandler.GetLogo)

	protected := api.Group(
		"",
		mw.JWTMiddleware(cfg.JWT.Secret),
		mw.TenantContext(tenantRepo),
		mw.AuditMiddleware(auditLogService),
	)

	admin := protected.Group("/admin", mw.AdminOnly())
	admin.POST("/tenants", tenantHandler.Create)
	admin.GET("/tenants", tenantHandler.List)
	admin.GET("/tenants/:id", tenantHandler.GetByID)
	admin.PUT("/tenants/:id", tenantHandler.Update)
	admin.DELETE("/tenants/:id", tenantHandler.Delete)
	admin.POST("/tenants/:id/regenerate-key", tenantHandler.RegenerateAPIKey)
	admin.GET("/stats/by-tenant", mailHandler.StatsByTenant)
	admin.GET("/queues", queueHandler.List)
	admin.PUT("/branding", brandingHandler.Update)
	admin.POST("/branding/logo", brandingHandler.UploadLogo)
	admin.GET("/audit-logs", auditLogHandler.List)

	protected.GET("/tenant/me", tenantHandler.Me)
	protected.GET("/queues", queueHandler.List)
	protected.GET("/audit-logs", auditLogHandler.ListByTenant)

	protected.POST("/mails", mailHandler.Create)
	protected.GET("/mails", mailHandler.List)
	protected.GET("/mails/stats", mailHandler.Stats)
	protected.GET("/mails/:id", mailHandler.GetByID)
	protected.GET("/mails/:id/attachments/:attachmentId", mailHandler.DownloadAttachment)
	protected.POST("/mails/:id/cancel", mailHandler.Cancel)
	protected.POST("/mails/:id/retry", mailHandler.Retry)

	protected.POST("/templates", templateHandler.Create)
	protected.GET("/templates", templateHandler.List)
	protected.GET("/templates/:id", templateHandler.GetByID)
	protected.PUT("/templates/:id", templateHandler.Update)
	protected.DELETE("/templates/:id", templateHandler.Delete)
	protected.POST("/templates/:id/preview", templateHandler.Preview)
	protected.POST("/templates/:id/spam-check", analysisHandler.SpamCheck)
	protected.POST("/templates/:id/html-check", analysisHandler.HTMLCheck)
	protected.POST("/templates/:id/link-check", analysisHandler.LinkCheck)

	protected.POST("/smtp-configs", smtpConfigHandler.Create)
	protected.GET("/smtp-configs", smtpConfigHandler.List)
	protected.GET("/smtp-configs/:id", smtpConfigHandler.GetByID)
	protected.PUT("/smtp-configs/:id", smtpConfigHandler.Update)
	protected.DELETE("/smtp-configs/:id", smtpConfigHandler.Delete)
	protected.POST("/smtp-configs/:id/test", smtpConfigHandler.Test)

	protected.POST("/ai/generate", llmHandler.Generate)
	protected.GET("/ai/status", llmHandler.Status)

	// Démarrage serveur HTTP
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("serveur API démarré", "component", "api", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatal("erreur serveur", "component", "api", "err", err)
		}
	}()

	<-ctx.Done()

	slog.Info("arrêt gracieux", "component", "api")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("erreur d'arrêt", "component", "api", "err", err)
	}
	// Le serveur HTTP est arrêté : plus aucune requête ne produit d'audit.
	// On draine les écritures d'audit en attente.
	auditLogService.Close()
	slog.Info("serveur API arrêté", "component", "api")
}

// ---------- Démarrage Worker ----------

func startWorker(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, redisClient *goredis.Client) {
	if warning := cfg.PoolWarning(); warning != "" {
		slog.Warn(warning, "component", "worker")
	}

	rateLimiter := adaptRedis.NewRateLimiter(redisClient)

	tenantRepo := postgres.NewTenantRepository(pool)
	smtpRepo := postgres.NewSMTPConfigRepository(pool)
	mailRepo := postgres.NewMailRepository(pool)
	templateRepo := postgres.NewTemplateRepository(pool)

	var sender port.MailSender
	if cfg.SMTPMode == "simulation" {
		sender = mailer.NewLogSender()
		slog.Info("mode SMTP configuré", "component", "worker", "mode", "simulation")
	} else {
		sender = mailer.NewSender()
		slog.Info("mode SMTP configuré", "component", "worker", "mode", "réel")
	}
	smtpService, err := service.NewSMTPConfigService(smtpRepo, sender, cfg.Encryption.Key)
	if err != nil {
		fatal("erreur service SMTP", "component", "worker", "err", err)
	}

	cbRegistry := worker.NewCircuitBreakerRegistry(worker.DefaultCircuitBreakerConfig())

	attachmentService := buildAttachmentService(ctx, cfg, pool, "worker")

	mailHandler := worker.NewMailHandler(
		mailRepo, smtpRepo, tenantRepo, templateRepo,
		sender, smtpService, rateLimiter, attachmentService, cbRegistry,
	)
	partitionHandler := worker.NewPartitionHandler(pool)
	archiveHandler := worker.NewArchiveHandler(pool)
	gcHandler := worker.NewGCHandler(attachmentService)

	if err := partitionHandler.EnsurePartitions(ctx); err != nil {
		slog.Warn("partitions initiales non créées", "component", "worker", "err", err)
	}

	srv := worker.NewServer(cfg.Redis.Addr, cfg.Worker, mailRepo)
	mux := worker.NewMux(mailHandler, partitionHandler, archiveHandler, gcHandler)

	scheduler := worker.NewScheduler(cfg.Redis.Addr)
	_, err = scheduler.Register("@daily", asynq.NewTask(worker.TypePartitionMaintenance, nil))
	if err != nil {
		fatal("erreur cron partition", "component", "worker", "err", err)
	}
	_, err = scheduler.Register("@daily", asynq.NewTask(worker.TypeMailArchive, nil))
	if err != nil {
		fatal("erreur cron archivage", "component", "worker", "err", err)
	}
	_, err = scheduler.Register("@hourly", asynq.NewTask(worker.TypeAttachmentGC, nil))
	if err != nil {
		fatal("erreur cron GC pièces jointes", "component", "worker", "err", err)
	}
	go func() {
		if err := scheduler.Run(); err != nil {
			slog.Error("erreur scheduler", "component", "worker", "err", err)
		}
	}()

	// Arrêt gracieux sur signal
	go func() {
		<-ctx.Done()
		slog.Info("arrêt gracieux", "component", "worker")
		scheduler.Shutdown()
		srv.Shutdown()
	}()

	slog.Info("worker démarré", "component", "worker", "concurrency", cfg.Worker.Concurrency)
	if err := srv.Run(mux); err != nil {
		slog.Error("serveur worker arrêté", "component", "worker", "err", err)
	}
	slog.Info("worker arrêté", "component", "worker")
}

// ---------- Seed des données par défaut ----------

func seedDefaults(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) {
	tenantRepo := postgres.NewTenantRepository(pool)
	smtpRepo := postgres.NewSMTPConfigRepository(pool)

	tenant, err := tenantRepo.GetBySlug(ctx, "admin")
	if err != nil {
		now := time.Now()
		tenant = &domain.Tenant{
			ID:       uuid.New(),
			Name:     "Administration",
			Slug:     "admin",
			APIKey:   cfg.Admin.APIKey,
			IsActive: true,
			Settings: domain.TenantSettings{
				RateLimit:        100,
				RateBurst:        200,
				MaxDestinataires: 100,
				DefaultPriority:  domain.MailPriorityCritical,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tenantRepo.Create(ctx, tenant); err != nil {
			t, err2 := tenantRepo.GetBySlug(ctx, "admin")
			if err2 != nil {
				slog.Error("erreur création tenant admin", "err", err)
				return
			}
			tenant = t
		}
		slog.Info("tenant admin créé")
	}

	// La config SMTP Mailpit par défaut n'a de sens qu'avec le conteneur Mailpit
	// du profil dev (SMTP_MODE=mailpit). En mode "real" (prod), l'administrateur
	// configure son propre SMTP : on évite de seeder une config par défaut qui
	// pointerait vers un hôte "mailpit" inexistant.
	if cfg.SMTPMode != "mailpit" {
		return
	}

	configs, err := smtpRepo.List(ctx, tenant.ID)
	if err != nil {
		slog.Error("erreur vérification configs SMTP", "err", err)
		return
	}
	if len(configs) > 0 {
		return
	}

	now := time.Now()
	smtpCfg := &domain.SMTPConfig{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		Name:       "Mailpit (dev)",
		Host:       cfg.Mailpit.Host,
		Port:       cfg.Mailpit.Port,
		AuthMethod: domain.AuthNone,
		TLSPolicy:  domain.TLSNone,
		FromEmail:  cfg.Mailpit.FromEmail,
		FromName:   cfg.Mailpit.FromName,
		Charset:    domain.CharsetUTF8,
		Encoding:   domain.EncodingQP,
		IsDefault:  true,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := smtpRepo.Create(ctx, smtpCfg); err != nil {
		slog.Error("erreur création config SMTP Mailpit", "err", err)
		return
	}
	slog.Info("config SMTP Mailpit créée", "host", cfg.Mailpit.Host, "port", cfg.Mailpit.Port)
}
