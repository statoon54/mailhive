package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/i18n"
)

// PartitionHandler gère la création automatique des partitions mensuelles pour audit_logs.
type PartitionHandler struct {
	pool *pgxpool.Pool
}

// NewPartitionHandler crée un nouveau PartitionHandler.
func NewPartitionHandler(pool *pgxpool.Pool) *PartitionHandler {
	return &PartitionHandler{pool: pool}
}

// HandlePartitionMaintenance est le handler Asynq pour la tâche de maintenance des partitions.
func (h *PartitionHandler) HandlePartitionMaintenance(_ context.Context, _ *asynq.Task) error {
	return h.EnsurePartitions(context.Background())
}

// EnsurePartitions crée les partitions pour le mois courant et les 3 mois suivants.
// Créer les partitions en avance évite que des lignes tombent dans la partition default.
func (h *PartitionHandler) EnsurePartitions(ctx context.Context) error {
	now := time.Now().UTC()

	for i := range 4 {
		t := now.AddDate(0, i, 0)
		name := fmt.Sprintf("audit_logs_%d_%02d", t.Year(), t.Month())
		from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)

		_, err := h.pool.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_logs FOR VALUES FROM ('%s') TO ('%s')`,
			name,
			from.Format("2006-01-02"),
			to.Format("2006-01-02"),
		))
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				slog.Warn("table audit_logs absente, partitionnement ignoré")
				return nil
			}
			if strings.Contains(err.Error(), "would be violated") {
				slog.Info("migration de la partition par défaut", "partition", name)
				if err := h.migrateFromDefault(ctx, name, from.Format("2006-01-02"), to.Format("2006-01-02")); err != nil {
					return fmt.Errorf(i18n.T(i18n.FR, "worker.err.partition_migrate"), name, err)
				}
				slog.Info("partition prête (migrée)", "partition", name)
				continue
			}
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.partition_create"), name, err)
		}
		slog.Info("partition prête", "partition", name)
	}

	return nil
}

// migrateFromDefault détache la partition default, crée la nouvelle partition,
// déplace les lignes concernées, puis rattache la partition default.
func (h *PartitionHandler) migrateFromDefault(ctx context.Context, name, from, to string) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	steps := []string{
		// Détacher la partition default
		`ALTER TABLE audit_logs DETACH PARTITION audit_logs_default`,
		// Créer la nouvelle partition
		fmt.Sprintf(
			`CREATE TABLE %s PARTITION OF audit_logs FOR VALUES FROM ('%s') TO ('%s')`,
			name,
			from,
			to,
		),
		// Déplacer les lignes de default vers la nouvelle partition
		fmt.Sprintf(
			`WITH moved AS (DELETE FROM audit_logs_default WHERE created_at >= '%s' AND created_at < '%s' RETURNING *) INSERT INTO %s SELECT * FROM moved`,
			from,
			to,
			name,
		),
		// Rattacher la partition default
		`ALTER TABLE audit_logs ATTACH PARTITION audit_logs_default DEFAULT`,
	}

	for _, q := range steps {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("%s : %w", q[:min(60, len(q))], err)
		}
	}

	return tx.Commit(ctx)
}
