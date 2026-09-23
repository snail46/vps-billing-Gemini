package commerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type FakeWebhookPayload struct {
	PaymentNo        string `json:"payment_no"`
	GatewayPaymentID string `json:"gateway_payment_id"`
	OrderID          string `json:"order_id"`
	AmountMinor      int64  `json:"amount_minor"`
	Currency         string `json:"currency"`
	Status           string `json:"status"` // "succeeded", "failed"
	Timestamp        int64  `json:"timestamp"`
}

type FakePaymentGateway struct {
	webhookSecret string
}

func NewFakePaymentGateway(secret string) *FakePaymentGateway {
	if secret == "" {
		secret = "fake_gateway_default_secret_key_2026"
	}
	return &FakePaymentGateway{webhookSecret: secret}
}

func (g *FakePaymentGateway) Name() string {
	return "fake"
}

// GenerateSignature generates HMAC-SHA256 signature for payload
func (g *FakePaymentGateway) GenerateSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(g.webhookSecret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies the HMAC signature header
func (g *FakePaymentGateway) VerifySignature(payload []byte, signature string) bool {
	expected := g.GenerateSignature(payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// GenerateSimulatedWebhook generates a valid signed webhook payload for testing/simulation
func (g *FakePaymentGateway) GenerateSimulatedWebhook(
	paymentNo string,
	orderID uuid.UUID,
	amountMinor int64,
	currency string,
) ([]byte, string, error) {
	payload := FakeWebhookPayload{
		PaymentNo:        paymentNo,
		GatewayPaymentID: fmt.Sprintf("fake_gw_tx_%s", uuid.New().String()[:12]),
		OrderID:          orderID.String(),
		AmountMinor:      amountMinor,
		Currency:         currency,
		Status:           "succeeded",
		Timestamp:        time.Now().Unix(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	sig := g.GenerateSignature(body)
	return body, sig, nil
}

// ParseWebhook parses and validates incoming webhook body
func (g *FakePaymentGateway) ParseWebhook(body []byte, signature string) (*FakeWebhookPayload, error) {
	if !g.VerifySignature(body, signature) {
		return nil, domainCommerce.ErrInvalidWebhookSignature
	}

	var payload FakeWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode webhook json: %w", err)
	}

	if payload.PaymentNo == "" {
		return nil, errors.New("missing payment_no in payload")
	}

	return &payload, nil
}
