package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestMailHandler_Create_Grouped(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	svc := &mocks.MockMailService{
		Mails: []*domain.Mail{{ID: mailID, Status: domain.MailStatusQueued}},
	}
	h := NewMailHandler(svc)

	body := map[string]any{
		"to":        []map[string]string{{"email": "a@b.com"}},
		"subject":   "Test",
		"text_body": "Hello",
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/mails", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestMailHandler_Create_Individuel(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockMailService{
		Mails: []*domain.Mail{
			{ID: uuid.New(), Status: domain.MailStatusQueued},
			{ID: uuid.New(), Status: domain.MailStatusQueued},
		},
	}
	h := NewMailHandler(svc)

	body := map[string]any{
		"to":         []map[string]string{{"email": "a@b.com"}, {"email": "c@d.com"}},
		"subject":    "Test",
		"individuel": true,
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/mails", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Contains(t, rec.Body.String(), "total")
}

func TestMailHandler_Create_ValidationError(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockMailService{}
	h := NewMailHandler(svc)

	body := map[string]any{
		"subject": "Test",
		// Missing 'to' field
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/mails", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMailHandler_Create_InvalidCC(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockMailService{}
	h := NewMailHandler(svc)

	body := map[string]any{
		"to":        []map[string]string{{"email": "valide@example.com"}},
		"cc":        []map[string]string{{"email": "pas-une-adresse"}},
		"subject":   "Test",
		"text_body": "Hello",
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/mails", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMailHandler_List_Default(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockMailService{
		List_: &domain.PaginatedList[domain.Mail]{
			Items: []domain.Mail{}, Total: 0, Page: 1, Limit: 20,
		},
	}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/mails", nil)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.List(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMailHandler_GetByID(t *testing.T) {
	tenantID := uuid.New()
	mailID := uuid.New()
	svc := &mocks.MockMailService{
		Mail: &domain.Mail{ID: mailID, Subject: "Test"},
	}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/mails/"+mailID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": mailID.String()})

	err := h.GetByID(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMailHandler_DownloadAttachment(t *testing.T) {
	tenantID := uuid.New()
	mailID := uuid.New()
	attID := uuid.New()
	svc := &mocks.MockMailService{
		AttachRef:  &domain.AttachmentRef{Filename: "document.pdf", ContentType: "application/pdf", Size: 3},
		AttachData: []byte("abc"),
	}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/mails/"+mailID.String()+"/attachments/"+attID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": mailID.String(), "attachmentId": attID.String()})

	err := h.DownloadAttachment(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), `filename="document.pdf"`)
	assert.Equal(t, "abc", rec.Body.String())
}

func TestMailHandler_DownloadAttachment_NotFound(t *testing.T) {
	tenantID := uuid.New()
	mailID := uuid.New()
	attID := uuid.New()
	svc := &mocks.MockMailService{Err: domain.ErrAttachmentNotFound}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/mails/"+mailID.String()+"/attachments/"+attID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": mailID.String(), "attachmentId": attID.String()})

	err := h.DownloadAttachment(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMailHandler_Cancel(t *testing.T) {
	tenantID := uuid.New()
	mailID := uuid.New()
	svc := &mocks.MockMailService{}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/mails/"+mailID.String()+"/cancel", nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": mailID.String()})

	err := h.Cancel(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMailHandler_Cancel_NotPending(t *testing.T) {
	tenantID := uuid.New()
	mailID := uuid.New()
	svc := &mocks.MockMailService{Err: domain.ErrMailNotPending}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/mails/"+mailID.String()+"/cancel", nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": mailID.String()})

	err := h.Cancel(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestMailHandler_Retry(t *testing.T) {
	tenantID := uuid.New()
	mailID := uuid.New()
	svc := &mocks.MockMailService{}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/mails/"+mailID.String()+"/retry", nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": mailID.String()})

	err := h.Retry(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMailHandler_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockMailService{}
	h := NewMailHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/mails", nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	// Override with invalid JSON body
	c.Request().Body = http.NoBody

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
