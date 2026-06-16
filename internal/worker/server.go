package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hibiken/asynq"

	"github.com/statoon54/mailhive/internal/config"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// NewServer crée un nouveau serveur Asynq.
func NewServer(
	redisAddr string,
	cfg config.WorkerConfig,
	mailRepo port.MailRepository,
) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues: map[string]int{
				"critical": cfg.QueueCritical,
				"default":  cfg.QueueDefault,
				"low":      cfg.QueueLow,
			},
			RetryDelayFunc: retryDelay,
			// IsFailure détermine si une erreur compte comme un échec (consomme un retry).
			// Les erreurs de rate limit et circuit breaker sont transitoires et ne doivent
			// pas épuiser les retries du mail.
			IsFailure: func(err error) bool {
				if errors.Is(err, domain.ErrRateLimited) {
					return false
				}
				if errors.Is(err, domain.ErrCircuitOpen) {
					return false
				}
				return true
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					slog.Error("erreur de tâche", "task_type", task.Type(), "err", err)

					// Sur le dernier retry d'un mail, marquer en échec en BDD
					if task.Type() == TypeMailSend {
						retried, _ := asynq.GetRetryCount(ctx)
						maxRetry, _ := asynq.GetMaxRetry(ctx)
						if retried >= maxRetry-1 {
							var payload MailSendPayload
							if jsonErr := json.Unmarshal(task.Payload(), &payload); jsonErr == nil {
								if dbErr := mailRepo.UpdateStatus(
									ctx, payload.MailID,
									domain.MailStatusFailed,
									err.Error(),
								); dbErr != nil {
									slog.Error("échec de mise à jour du statut en échec",
										"mail_id", payload.MailID, "err", dbErr)
								}
							}
						}
					}
				},
			),
		},
	)
}

// retryDelay calcule le délai avant le prochain retry en fonction du type d'erreur.
func retryDelay(n int, err error, task *asynq.Task) time.Duration {
	// Circuit breaker ouvert → attendre plus longtemps (60s fixe)
	if errors.Is(err, domain.ErrCircuitOpen) {
		return 60 * time.Second
	}

	// Rate limit → attendre peu avec jitter pour éviter le thundering herd.
	// Délai aléatoire entre 1s et 5s pour étaler les retentatives.
	if errors.Is(err, domain.ErrRateLimited) {
		return time.Duration(1000+rand.IntN(4000)) * time.Millisecond
	}

	// Erreur SMTP temporaire → backoff exponentiel : 15s, 60s, 240s...
	delay := min(time.Duration(math.Pow(4, float64(n)))*15*time.Second, 10*time.Minute)
	return delay
}

// NewMux crée un nouveau multiplexeur de tâches Asynq.
func NewMux(
	mailHandler *MailHandler,
	partitionHandler *PartitionHandler,
	archiveHandler *ArchiveHandler,
	gcHandler *GCHandler,
) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeMailSend, mailHandler.HandleMailSend)
	mux.HandleFunc(TypePartitionMaintenance, partitionHandler.HandlePartitionMaintenance)
	mux.HandleFunc(TypeMailArchive, archiveHandler.HandleMailArchive)
	mux.HandleFunc(TypeAttachmentGC, gcHandler.HandleAttachmentGC)
	return mux
}

// NewAsynqClient crée un nouveau client Asynq.
func NewAsynqClient(redisAddr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
}

// NewScheduler crée un nouveau scheduler Asynq pour les tâches cron.
func NewScheduler(redisAddr string) *asynq.Scheduler {
	return asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddr}, nil,
	)
}
