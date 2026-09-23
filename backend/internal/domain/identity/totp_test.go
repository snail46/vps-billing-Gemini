package identity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vps-billing/internal/domain/identity"
)

func TestTOTPGenerationAndValidation(t *testing.T) {
	secret, err := identity.GenerateTOTPSecret()
	require.NoError(t, err)
	assert.NotEmpty(t, secret)

	now := time.Now().UTC()
	code, err := identity.GenerateTOTPCode(secret, now)
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// Valid at current time
	assert.True(t, identity.ValidateTOTPCode(secret, code, now))

	// Valid within clock skew (+15s)
	assert.True(t, identity.ValidateTOTPCode(secret, code, now.Add(15*time.Second)))

	// Valid within clock skew (-15s)
	assert.True(t, identity.ValidateTOTPCode(secret, code, now.Add(-15*time.Second)))

	// Invalid outside clock skew window (>60s)
	assert.False(t, identity.ValidateTOTPCode(secret, code, now.Add(90*time.Second)))
	assert.False(t, identity.ValidateTOTPCode(secret, code, now.Add(-90*time.Second)))

	// Invalid code
	assert.False(t, identity.ValidateTOTPCode(secret, "000000", now))

	// Invalid format
	assert.False(t, identity.ValidateTOTPCode(secret, "abc", now))

	// Test URI builder
	uri := identity.GetTOTPUri(secret, "admin@example.com", "VPS-Billing")
	assert.Contains(t, uri, "otpauth://totp/")
	assert.Contains(t, uri, "admin%40example.com")
	assert.Contains(t, uri, "secret="+secret)
}
