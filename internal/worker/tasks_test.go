package worker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
)

func TestNewMailSendTask_PayloadJSON(t *testing.T) {
	mailID := uuid.New()
	tenantID := uuid.New()

	task, err := NewMailSendTask(mailID, tenantID, domain.MailPriorityDefault, nil)
	require.NoError(t, err)
	assert.Equal(t, TypeMailSend, task.Type())

	var payload MailSendPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &payload))
	assert.Equal(t, mailID, payload.MailID)
	assert.Equal(t, tenantID, payload.TenantID)
}

func TestNewMailSendTask_PriorityQueue(t *testing.T) {
	tests := []struct {
		priority domain.MailPriority
		queue    string
	}{
		{domain.MailPriorityCritical, "critical"},
		{domain.MailPriorityDefault, "default"},
		{domain.MailPriorityLow, "low"},
	}
	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			task, err := NewMailSendTask(uuid.New(), uuid.New(), tt.priority, nil)
			require.NoError(t, err)
			assert.NotNil(t, task)
		})
	}
}

func TestNewMailSendTask_Scheduled(t *testing.T) {
	future := time.Now().Add(time.Hour)
	task, err := NewMailSendTask(uuid.New(), uuid.New(), domain.MailPriorityDefault, &future)
	require.NoError(t, err)
	assert.NotNil(t, task)
}

func TestNewMailSendTask_NoSchedule(t *testing.T) {
	task, err := NewMailSendTask(uuid.New(), uuid.New(), domain.MailPriorityDefault, nil)
	require.NoError(t, err)
	assert.NotNil(t, task)
}
