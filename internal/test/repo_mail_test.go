package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
)

func insertMailWithTenant(t *testing.T, pool *pgxpool.Pool) (*domain.Tenant, *domain.Mail) {
	t.Helper()
	tenant := insertTenant(t, pool)
	mailRepo := postgres.NewMailRepository(pool)

	mailID, _ := uuid.NewV7()
	score := float32(1.5)
	mail := &domain.Mail{
		ID:        mailID,
		TenantID:  tenant.ID,
		FromEmail: "test@example.com",
		Subject:   "Test mail",
		TextBody:  "Hello",
		Status:    domain.MailStatusPending,
		Priority:  domain.MailPriorityDefault,
		SpamScore: &score,
		Tags:      []string{"test-tag"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, mailRepo.Create(context.Background(), mail))
	return tenant, mail
}

func TestMailRepo_CreateAndGetByID(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()

	tenant, mail := insertMailWithTenant(t, pool)

	got, err := repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	assert.Equal(t, mail.Subject, got.Subject)
	assert.Equal(t, mail.Status, got.Status)
	require.NotNil(t, got.SpamScore)
	assert.InDelta(t, 1.5, float64(*got.SpamScore), 0.01)
	assert.Contains(t, got.Tags, "test-tag")
}

func TestMailRepo_GetByID_NotFound(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrMailNotFound)
}

func TestMailRepo_CreateBatch(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	mails := make([]*domain.Mail, 5)
	for i := range mails {
		id, _ := uuid.NewV7()
		mails[i] = &domain.Mail{
			ID:        id,
			TenantID:  tenant.ID,
			FromEmail: "batch@example.com",
			Subject:   "Batch mail",
			Status:    domain.MailStatusPending,
			Priority:  domain.MailPriorityDefault,
			Tags:      []string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	require.NoError(t, repo.CreateBatch(ctx, mails))

	for _, m := range mails {
		got, err := repo.GetByID(ctx, tenant.ID, m.ID)
		require.NoError(t, err)
		assert.Equal(t, "Batch mail", got.Subject)
	}
}

func TestMailRepo_CreateWithRecipients_Success(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	mailID, _ := uuid.NewV7()
	mail := &domain.Mail{
		ID: mailID, TenantID: tenant.ID, FromEmail: "tx@example.com",
		Subject: "Tx mail", Status: domain.MailStatusPending,
		Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	rid, _ := uuid.NewV7()
	recipients := []domain.MailRecipient{
		{ID: rid, MailID: mailID, Type: domain.RecipientTo, Email: "ok@example.com"},
	}

	require.NoError(t, repo.CreateWithRecipients(ctx, mail, recipients, nil))

	got, err := repo.GetByID(ctx, tenant.ID, mailID)
	require.NoError(t, err)
	assert.Equal(t, "Tx mail", got.Subject)

	recs, err := repo.GetRecipients(ctx, mailID)
	require.NoError(t, err)
	assert.Len(t, recs, 1)
}

// TestMailRepo_CreateWithRecipients_RollsBack vérifie l'atomicité : si l'insert
// d'un destinataire échoue (type hors enum recipient_type), le mail n'est pas
// persisté (transaction annulée, pas d'orphelin).
func TestMailRepo_CreateWithRecipients_RollsBack(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	mailID, _ := uuid.NewV7()
	mail := &domain.Mail{
		ID: mailID, TenantID: tenant.ID, FromEmail: "tx@example.com",
		Subject: "Tx mail", Status: domain.MailStatusPending,
		Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	rid, _ := uuid.NewV7()
	bad := []domain.MailRecipient{
		{ID: rid, MailID: mailID, Type: domain.RecipientType("invalide"), Email: "x@example.com"},
	}

	err := repo.CreateWithRecipients(ctx, mail, bad, nil)
	require.Error(t, err)

	_, err = repo.GetByID(ctx, tenant.ID, mailID)
	assert.ErrorIs(t, err, domain.ErrMailNotFound)
}

// TestMailRepo_CreateBatchWithRecipients_RollsBack vérifie l'atomicité du mode
// individuel : un destinataire invalide annule l'insertion de tous les mails.
func TestMailRepo_CreateBatchWithRecipients_RollsBack(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	id1, _ := uuid.NewV7()
	id2, _ := uuid.NewV7()
	mails := []*domain.Mail{
		{ID: id1, TenantID: tenant.ID, FromEmail: "a@example.com", Subject: "M1", Status: domain.MailStatusPending, Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: id2, TenantID: tenant.ID, FromEmail: "b@example.com", Subject: "M2", Status: domain.MailStatusPending, Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	rid, _ := uuid.NewV7()
	bad := []domain.MailRecipient{
		{ID: rid, MailID: id1, Type: domain.RecipientType("invalide"), Email: "x@example.com"},
	}

	err := repo.CreateBatchWithRecipients(ctx, mails, bad, nil)
	require.Error(t, err)

	for _, id := range []uuid.UUID{id1, id2} {
		_, err := repo.GetByID(ctx, tenant.ID, id)
		assert.ErrorIs(t, err, domain.ErrMailNotFound)
	}
}

func TestMailRepo_List(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, _ := insertMailWithTenant(t, pool)

	list, err := repo.List(ctx, tenant.ID, domain.MailListFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Items), 1)
}

func TestMailRepo_List_StatusFilter(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, _ := insertMailWithTenant(t, pool)

	status := domain.MailStatusPending
	list, err := repo.List(ctx, tenant.ID, domain.MailListFilter{
		Status: &status,
		Page:   1,
		Limit:  20,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Items), 1)

	sent := domain.MailStatusSent
	list, err = repo.List(ctx, tenant.ID, domain.MailListFilter{
		Status: &sent,
		Page:   1,
		Limit:  20,
	})
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestMailRepo_List_TagsFilter(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, _ := insertMailWithTenant(t, pool)

	// AND mode (default)
	list, err := repo.List(ctx, tenant.ID, domain.MailListFilter{
		Tags:  []string{"test-tag"},
		Page:  1,
		Limit: 20,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Items), 1)

	// OR mode
	list, err = repo.List(ctx, tenant.ID, domain.MailListFilter{
		Tags:    []string{"nonexistent"},
		TagMode: "or",
		Page:    1,
		Limit:   20,
	})
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestMailRepo_List_QueryFilter(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, _ := insertMailWithTenant(t, pool)

	list, err := repo.List(ctx, tenant.ID, domain.MailListFilter{
		Query: "Test mail",
		Page:  1,
		Limit: 20,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Items), 1)
}

// TestMailRepo_List_TotalMatchesFilter garantit que le comptage et le listage
// appliquent exactement le même filtre (clause WHERE partagée).
func TestMailRepo_List_TotalMatchesFilter(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, _ := insertMailWithTenant(t, pool) // sujet "Test mail"

	// Second mail au sujet distinct.
	id2, _ := uuid.NewV7()
	require.NoError(t, repo.Create(ctx, &domain.Mail{
		ID: id2, TenantID: tenant.ID, FromEmail: "x@example.com",
		Subject: "Bulletin mensuel", Status: domain.MailStatusPending,
		Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	list, err := repo.List(ctx, tenant.ID, domain.MailListFilter{
		Query: "Bulletin",
		Page:  1,
		Limit: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total, "Total doit refléter le même filtre que la liste")
	require.Len(t, list.Items, 1)
	assert.Equal(t, "Bulletin mensuel", list.Items[0].Subject)
}

func TestMailRepo_UpdateStatus(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail := insertMailWithTenant(t, pool)

	require.NoError(t, repo.UpdateStatus(ctx, mail.ID, domain.MailStatusSent, ""))

	got, err := repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailStatusSent, got.Status)
}

func TestMailRepo_SetQueued(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail := insertMailWithTenant(t, pool)

	require.NoError(t, repo.SetQueued(ctx, mail.ID, "task-123"))

	got, err := repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailStatusQueued, got.Status)
	assert.Equal(t, "task-123", got.TaskID)
}

func TestMailRepo_SetQueuedBatch(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	mails := make([]*domain.Mail, 3)
	mailTaskIDs := make(map[uuid.UUID]string)
	for i := range mails {
		id, _ := uuid.NewV7()
		mails[i] = &domain.Mail{
			ID: id, TenantID: tenant.ID, FromEmail: "test@test.com",
			Subject: "Batch", Status: domain.MailStatusPending,
			Priority: domain.MailPriorityDefault, Tags: []string{},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		mailTaskIDs[id] = "task-" + id.String()[:8]
	}
	require.NoError(t, repo.CreateBatch(ctx, mails))
	require.NoError(t, repo.SetQueuedBatch(ctx, mailTaskIDs))

	for _, m := range mails {
		got, err := repo.GetByID(ctx, tenant.ID, m.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.MailStatusQueued, got.Status)
		assert.NotEmpty(t, got.TaskID)
	}
}

func TestMailRepo_MarkSending(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail := insertMailWithTenant(t, pool)

	// Une seule requête doit passer le statut à "sending" ET incrémenter attempts.
	require.NoError(t, repo.MarkSending(ctx, mail.ID))
	require.NoError(t, repo.MarkSending(ctx, mail.ID))

	got, err := repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailStatusSending, got.Status)
	assert.Equal(t, 2, got.Attempts)
}

func TestMailRepo_MarkSent(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail := insertMailWithTenant(t, pool)

	// Une seule requête doit passer le statut à "sent" ET renseigner sent_at.
	require.NoError(t, repo.MarkSent(ctx, mail.ID))

	got, err := repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailStatusSent, got.Status)
	require.NotNil(t, got.SentAt)
}

func TestMailRepo_CreateAndGetRecipients(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	_, mail := insertMailWithTenant(t, pool)

	recID1, _ := uuid.NewV7()
	recID2, _ := uuid.NewV7()
	recipients := []domain.MailRecipient{
		{ID: recID1, MailID: mail.ID, Type: domain.RecipientTo, Email: "to@example.com", Name: "To"},
		{ID: recID2, MailID: mail.ID, Type: domain.RecipientCC, Email: "cc@example.com"},
	}

	require.NoError(t, repo.CreateRecipients(ctx, recipients))

	got, err := repo.GetRecipients(ctx, mail.ID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestMailRepo_GetRecipientsByMailIDs(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail1 := insertMailWithTenant(t, pool)

	// Second mail dans le même tenant.
	mail2ID, _ := uuid.NewV7()
	mail2 := &domain.Mail{
		ID: mail2ID, TenantID: tenant.ID, FromEmail: "test@example.com",
		Subject: "Mail 2", TextBody: "Hi", Status: domain.MailStatusPending,
		Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, mail2))

	rec1, _ := uuid.NewV7()
	rec2a, _ := uuid.NewV7()
	rec2b, _ := uuid.NewV7()
	require.NoError(t, repo.CreateRecipients(ctx, []domain.MailRecipient{
		{ID: rec1, MailID: mail1.ID, Type: domain.RecipientTo, Email: "a@example.com"},
		{ID: rec2a, MailID: mail2.ID, Type: domain.RecipientTo, Email: "b@example.com"},
		{ID: rec2b, MailID: mail2.ID, Type: domain.RecipientCC, Email: "c@example.com"},
	}))

	// Inclure un ID inexistant : il doit simplement être absent de la map.
	byID, err := repo.GetRecipientsByMailIDs(ctx, []uuid.UUID{mail1.ID, mail2.ID, uuid.New()})
	require.NoError(t, err)

	assert.Len(t, byID, 2)
	assert.Len(t, byID[mail1.ID], 1)
	assert.Len(t, byID[mail2.ID], 2)
	for _, r := range byID[mail2.ID] {
		assert.Equal(t, mail2.ID, r.MailID)
	}

	// Slice vide → map vide, pas d'erreur.
	empty, err := repo.GetRecipientsByMailIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestMailRepo_AddTags(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail := insertMailWithTenant(t, pool)

	require.NoError(t, repo.AddTags(ctx, mail.ID, []string{"bounced", "retry-1"}))

	got, err := repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	assert.Contains(t, got.Tags, "test-tag")
	assert.Contains(t, got.Tags, "bounced")
	assert.Contains(t, got.Tags, "retry-1")

	// Dedup
	require.NoError(t, repo.AddTags(ctx, mail.ID, []string{"bounced"}))
	got, err = repo.GetByID(ctx, tenant.ID, mail.ID)
	require.NoError(t, err)
	count := 0
	for _, tag := range got.Tags {
		if tag == "bounced" {
			count++
		}
	}
	assert.Equal(t, 1, count, "tags should be deduplicated")
}

func TestMailRepo_Stats(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewMailRepository(pool)
	ctx := context.Background()
	tenant, mail := insertMailWithTenant(t, pool)

	// Set one to sent
	require.NoError(t, repo.UpdateStatus(ctx, mail.ID, domain.MailStatusSent, ""))

	stats, err := repo.Stats(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Total)
	assert.Equal(t, int64(1), stats.Sent)
}
