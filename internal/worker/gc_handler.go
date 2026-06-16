package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/statoon54/mailhive/internal/service"
)

// attachmentGCBatchSize borne le nombre de pièces jointes orphelines traitées par exécution.
const attachmentGCBatchSize = 1000

// attachmentGCGracePeriod est le délai de grâce avant de collecter une pièce
// jointe orpheline (sans lien mail_attachments). Il évite de supprimer un contenu
// qu'une campagne en cours de création est en train de référencer.
const attachmentGCGracePeriod = time.Hour

// GCHandler collecte les pièces jointes dédupliquées devenues orphelines
// (plus aucun mail ne les référence) : supprime le blob puis la métadonnée.
type GCHandler struct {
	attachments *service.AttachmentService
}

// NewGCHandler crée un nouveau handler de garbage collection des pièces jointes.
func NewGCHandler(attachments *service.AttachmentService) *GCHandler {
	return &GCHandler{attachments: attachments}
}

// HandleAttachmentGC est le handler Asynq pour la tâche de GC des pièces jointes.
func (h *GCHandler) HandleAttachmentGC(ctx context.Context, _ *asynq.Task) error {
	cutoff := time.Now().Add(-attachmentGCGracePeriod)
	deleted, err := h.attachments.CollectOrphans(ctx, cutoff, attachmentGCBatchSize)
	if err != nil {
		return err
	}
	if deleted > 0 {
		slog.Info("pièces jointes orphelines collectées", "deleted", deleted)
	}
	return nil
}
