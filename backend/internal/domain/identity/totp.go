package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var b32Encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateTOTPSecret generates a secure random 160-bit (20-byte) base32-encoded secret.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	return b32Encoding.EncodeToString(buf), nil
}

// GenerateTOTPCode produces a 6-digit TOTP code for the given secret at time t.
func GenerateTOTPCode(secret string, t time.Time) (string, error) {
	cleanSecret := strings.ToUpper(strings.TrimSpace(secret))
	key, err := b32Encoding.DecodeString(cleanSecret)
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(cleanSecret)
		if err != nil {
			return "", fmt.Errorf("invalid base32 secret: %w", err)
		}
	}

	counter := uint64(t.Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	binaryCode := (int(hash[offset])&0x7f)<<24 |
		(int(hash[offset+1])&0xff)<<16 |
		(int(hash[offset+2])&0xff)<<8 |
		(int(hash[offset+3]) & 0xff)

	otp := binaryCode % 1000000
	return fmt.Sprintf("%06d", otp), nil
}

// ValidateTOTPCode validates code against secret allowing 1 step (30s) clock skew backward and forward.
func ValidateTOTPCode(secret string, code string, t time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}

	steps := []time.Time{
		t.Add(-30 * time.Second),
		t,
		t.Add(30 * time.Second),
	}

	for _, stepTime := range steps {
		generated, err := GenerateTOTPCode(secret, stepTime)
		if err == nil && hmac.Equal([]byte(generated), []byte(code)) {
			return true
		}
	}

	return false
}

// GetTOTPUri builds a standard otpauth URI for authenticator apps.
func GetTOTPUri(secret, accountName, issuer string) string {
	cleanSecret := strings.ToUpper(strings.TrimSpace(secret))
	encodedIssuer := url.QueryEscape(issuer)
	encodedAccount := url.QueryEscape(accountName)
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		encodedIssuer, encodedAccount, cleanSecret, encodedIssuer)
}
