package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMTPPermanentError_Error(t *testing.T) {
	cause := fmt.Errorf("auth failed")
	permErr := NewSMTPPermanentError(cause)
	assert.Contains(t, permErr.Error(), "erreur SMTP permanente")
	assert.Contains(t, permErr.Error(), "auth failed")
}

func TestSMTPPermanentError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("auth failed")
	permErr := NewSMTPPermanentError(cause)
	assert.Equal(t, cause, permErr.Unwrap())
	assert.True(t, errors.Is(permErr, cause))
}

func TestSMTPPermanentError_New(t *testing.T) {
	err := NewSMTPPermanentError(fmt.Errorf("test"))
	require.NotNil(t, err)
	var permErr *SMTPPermanentError
	assert.True(t, errors.As(err, &permErr))
}

func TestDomainErrors_AreDistinct(t *testing.T) {
	errs := []error{
		ErrNotFound, ErrTemplateNotFound, ErrSMTPConfigNotFound,
		ErrMailNotFound, ErrTenantNotFound, ErrConflict,
		ErrUnauthorized, ErrForbidden, ErrValidation,
		ErrInvalidAPIKey, ErrTenantInactive, ErrSMTPConfigNotSet,
		ErrMailNotPending, ErrMailNotFailed, ErrRateLimited,
		ErrCircuitOpen, ErrSMTPTemporary, ErrSpamBlocked,
		ErrLinkCheckRateLimit,
	}

	for i, a := range errs {
		for j, b := range errs {
			if i != j {
				assert.False(t, errors.Is(a, b), "expected %v != %v", a, b)
			}
		}
	}
}
