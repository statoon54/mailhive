package worker

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/statoon54/mailhive/internal/domain"
)

// Constantes des types de tâches.
const (
	TypeMailSend             = "mail:send"
	TypeMailArchive          = "mail:archive"
	TypeAttachmentGC         = "attachment:gc"
	TypePartitionMaintenance = "partition:maintenance"
)

// MailSendPayload contient les données nécessaires pour envoyer un mail.
type MailSendPayload struct {
	MailID   uuid.UUID `json:"mail_id"`
	TenantID uuid.UUID `json:"tenant_id"`
}

// NewMailSendTask crée une nouvelle tâche d'envoi de mail.
func NewMailSendTask(
	mailID, tenantID uuid.UUID,
	priority domain.MailPriority,
	scheduledAt *time.Time,
) (*asynq.Task, error) {
	payload, err := json.Marshal(MailSendPayload{
		MailID:   mailID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, err
	}
	opts := []asynq.Option{
		asynq.MaxRetry(5),
		asynq.Queue(string(priority)),
	}
	if scheduledAt != nil && scheduledAt.After(time.Now()) {
		opts = append(opts, asynq.ProcessAt(*scheduledAt))
	}
	return asynq.NewTask(TypeMailSend, payload, opts...), nil
}
