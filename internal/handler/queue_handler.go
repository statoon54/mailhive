package handler

import (
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/i18n"
)

// QueueHandler gère les endpoints de monitoring des queues Asynq.
type QueueHandler struct {
	inspector *asynq.Inspector
}

// NewQueueHandler crée un nouveau handler de queues.
func NewQueueHandler(inspector *asynq.Inspector) *QueueHandler {
	return &QueueHandler{inspector: inspector}
}

// queueInfoResponse représente les informations d'une queue Asynq.
type queueInfoResponse struct {
	Name      string `json:"name"`
	Active    int    `json:"active"`
	Pending   int    `json:"pending"`
	Scheduled int    `json:"scheduled"`
	Retry     int    `json:"retry"`
	Archived  int    `json:"archived"`
	Completed int    `json:"completed"`
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
	LatencyMs int64  `json:"latency_ms"`
	Paused    bool   `json:"paused"`
}

// List retourne les informations de toutes les queues Asynq.
func (h *QueueHandler) List(c *echo.Context) error {
	queues, err := h.inspector.Queues()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: i18n.T(lang(c), "err.queue_fetch"),
		})
	}

	result := make([]queueInfoResponse, 0, len(queues))
	for _, name := range queues {
		info, err := h.inspector.GetQueueInfo(name)
		if err != nil {
			continue
		}
		result = append(result, queueInfoResponse{
			Name:      info.Queue,
			Active:    info.Active,
			Pending:   info.Pending,
			Scheduled: info.Scheduled,
			Retry:     info.Retry,
			Archived:  info.Archived,
			Completed: info.Completed,
			Processed: info.Processed,
			Failed:    info.Failed,
			LatencyMs: info.Latency.Milliseconds(),
			Paused:    info.Paused,
		})
	}

	return ok(c, result)
}
