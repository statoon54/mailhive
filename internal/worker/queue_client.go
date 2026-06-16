package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
)

// QueueClient implémente port.QueueClient avec Asynq.
type QueueClient struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// NewQueueClient crée un nouveau client de file d'attente.
func NewQueueClient(client *asynq.Client, inspector *asynq.Inspector) *QueueClient {
	return &QueueClient{client: client, inspector: inspector}
}

// DeleteTask supprime une tâche de la file d'attente.
func (q *QueueClient) DeleteTask(queue, taskID string) error {
	return q.inspector.DeleteTask(queue, taskID)
}

// EnqueueMailSend met en file d'attente une tâche d'envoi de mail.
func (q *QueueClient) EnqueueMailSend(
	ctx context.Context,
	mailID, tenantID uuid.UUID,
	priority domain.MailPriority,
	scheduledAt *time.Time,
) (string, error) {
	task, err := NewMailSendTask(mailID, tenantID, priority, scheduledAt)
	if err != nil {
		return "", fmt.Errorf(i18n.T(i18n.FR, "worker.err.create_task"), err)
	}

	info, err := q.client.Enqueue(task)
	if err != nil {
		return "", fmt.Errorf(i18n.T(i18n.FR, "worker.err.enqueue"), err)
	}

	return info.ID, nil
}
