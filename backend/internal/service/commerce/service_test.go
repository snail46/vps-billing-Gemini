package commerce

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

func TestFakePaymentGateway_Signature(t *testing.T) {
	secret := "test_secret_key_123"
	gw := NewFakePaymentGateway(secret)

	payload := []byte(`{"payment_no":"PAY-001","amount_minor":1000}`)
	sig := gw.GenerateSignature(payload)
	if sig == "" {
		t.Fatalf("expected non-empty signature")
	}

	if !gw.VerifySignature(payload, sig) {
		t.Fatalf("signature verification should have passed")
	}

	// Tampered body should fail
	tampered := []byte(`{"payment_no":"PAY-001","amount_minor":2000}`)
	if gw.VerifySignature(tampered, sig) {
		t.Fatalf("tampered payload should fail verification")
	}

	// Wrong signature should fail
	if gw.VerifySignature(payload, "invalid_sig") {
		t.Fatalf("invalid signature should fail verification")
	}
}

func TestFakePaymentGateway_ParseWebhook(t *testing.T) {
	secret := "test_secret_key_123"
	gw := NewFakePaymentGateway(secret)

	paymentNo := "PAY-20260923-TEST"
	orderID := uuid.New()
	body, sig, err := gw.GenerateSimulatedWebhook(paymentNo, orderID, 2500, "USD")
	if err != nil {
		t.Fatalf("failed to generate simulated webhook: %v", err)
	}

	parsed, err := gw.ParseWebhook(body, sig)
	if err != nil {
		t.Fatalf("failed to parse webhook: %v", err)
	}

	if parsed.PaymentNo != paymentNo {
		t.Errorf("expected %s, got %s", paymentNo, parsed.PaymentNo)
	}
	if parsed.AmountMinor != 2500 {
		t.Errorf("expected 2500, got %d", parsed.AmountMinor)
	}
	if parsed.Currency != "USD" {
		t.Errorf("expected USD, got %s", parsed.Currency)
	}
	if parsed.Status != "succeeded" {
		t.Errorf("expected succeeded, got %s", parsed.Status)
	}
}

func TestOrderService_Validation(t *testing.T) {
	orderSvc := NewOrderService(nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// Empty items should fail
	_, err := orderSvc.CreateOrder(ctx, userID, nil)
	if err != domainCommerce.ErrEmptyOrderItems {
		t.Errorf("expected ErrEmptyOrderItems, got %v", err)
	}

	// Invalid quantity <= 0 should fail
	_, err = orderSvc.CreateOrder(ctx, userID, []CreateOrderItemInput{
		{PlanID: uuid.New(), Quantity: 0},
	})
	if err == nil {
		t.Errorf("expected error on quantity 0, got nil")
	}
}
