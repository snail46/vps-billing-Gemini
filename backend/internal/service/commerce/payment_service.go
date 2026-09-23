package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type OrderFulfiller interface {
	FulfillPaidOrder(ctx context.Context, orderID uuid.UUID) error
}

type PaymentService struct {
	paymentRepo domainCommerce.PaymentRepository
	orderRepo   domainCommerce.OrderRepository
	fakeGateway *FakePaymentGateway
	fulfiller   OrderFulfiller
}

func NewPaymentService(
	paymentRepo domainCommerce.PaymentRepository,
	orderRepo domainCommerce.OrderRepository,
	fakeGateway *FakePaymentGateway,
) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		orderRepo:   orderRepo,
		fakeGateway: fakeGateway,
	}
}

func (s *PaymentService) SetFulfiller(f OrderFulfiller) {
	s.fulfiller = f
}

type InitiatePaymentResult struct {
	Payment      *domainCommerce.Payment `json:"payment"`
	CheckoutURL  string                  `json:"checkout_url,omitempty"`
	RequiresPost bool                    `json:"requires_post"`
}

func (s *PaymentService) InitiatePayment(ctx context.Context, orderID uuid.UUID, gateway string) (*InitiatePaymentResult, error) {
	if gateway != "fake" {
		return nil, fmt.Errorf("unsupported payment gateway: %s", gateway)
	}

	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != domainCommerce.OrderStatusPending {
		return nil, domainCommerce.ErrInvalidOrderStatus
	}

	idempotencyKey := fmt.Sprintf("order_%s_gw_%s", order.ID.String(), gateway)
	existing, err := s.paymentRepo.GetPaymentByIdempotencyKey(ctx, idempotencyKey)
	if err == nil && existing != nil {
		if existing.Status == domainCommerce.PaymentStatusSucceeded {
			return nil, domainCommerce.ErrPaymentAlreadyProcessed
		}
		return &InitiatePaymentResult{
			Payment:     existing,
			CheckoutURL: fmt.Sprintf("/api/v1/payments/fake/simulate?payment_no=%s", existing.PaymentNo),
		}, nil
	}

	paymentNo := fmt.Sprintf("PAY-%s-%s", time.Now().UTC().Format("20060102"), uuid.New().String()[:8])
	p := &domainCommerce.Payment{
		ID:             uuid.New(),
		PaymentNo:      paymentNo,
		OrderID:        order.ID,
		Gateway:        gateway,
		Status:         domainCommerce.PaymentStatusPending,
		AmountMinor:    order.TotalMinor,
		Currency:       order.Currency,
		IdempotencyKey: idempotencyKey,
		GatewayPayload: json.RawMessage("{}"),
	}

	created, err := s.paymentRepo.CreatePayment(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return &InitiatePaymentResult{
		Payment:     created,
		CheckoutURL: fmt.Sprintf("/api/v1/payments/fake/simulate?payment_no=%s", created.PaymentNo),
	}, nil
}

type WebhookProcessResult struct {
	AlreadyProcessed bool   `json:"already_processed"`
	PaymentNo        string `json:"payment_no"`
	GatewayPaymentID string `json:"gateway_payment_id"`
}

func (s *PaymentService) ProcessWebhook(ctx context.Context, gateway string, body []byte, signature string) (*WebhookProcessResult, error) {
	if gateway != "fake" {
		return nil, fmt.Errorf("unsupported gateway: %s", gateway)
	}

	payload, err := s.fakeGateway.ParseWebhook(body, signature)
	if err != nil {
		return nil, err
	}

	payment, err := s.paymentRepo.GetPaymentByPaymentNo(ctx, payload.PaymentNo)
	if err != nil {
		return nil, fmt.Errorf("payment not found for payment_no %s: %w", payload.PaymentNo, err)
	}

	// Validate amounts and currency
	if payment.AmountMinor != payload.AmountMinor {
		return nil, fmt.Errorf("%w: expected %d, got %d", domainCommerce.ErrInvalidPaymentAmount, payment.AmountMinor, payload.AmountMinor)
	}
	if payment.Currency != payload.Currency {
		return nil, fmt.Errorf("%w: expected %s, got %s", domainCommerce.ErrInvalidCurrency, payment.Currency, payload.Currency)
	}

	// If gateway reports failure
	if payload.Status != "succeeded" {
		return nil, errors.New("gateway indicated payment was not successful")
	}

	order, err := s.orderRepo.GetOrderByID(ctx, payment.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to find associated order: %w", err)
	}

	// Execute atomic transaction with deduplication & row locking
	alreadyProcessed, err := s.paymentRepo.ProcessPaymentSuccessTx(ctx, domainCommerce.PaymentSuccessParams{
		PaymentID:        payment.ID,
		GatewayPaymentID: payload.GatewayPaymentID,
		GatewayPayload:   body,
		PaymentNo:        payment.PaymentNo,
		OrderID:          payment.OrderID,
		AmountMinor:      payment.AmountMinor,
		Currency:         payment.Currency,
		UserID:           order.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process payment transaction: %w", err)
	}

	if !alreadyProcessed && s.fulfiller != nil {
		go func(ordID uuid.UUID) {
			_ = s.fulfiller.FulfillPaidOrder(context.Background(), ordID)
		}(payment.OrderID)
	}

	return &WebhookProcessResult{
		AlreadyProcessed: alreadyProcessed,
		PaymentNo:        payment.PaymentNo,
		GatewayPaymentID: payload.GatewayPaymentID,
	}, nil
}

// GenerateFakeSimulatedWebhook is a helper to simulate payment for frontend and integration tests
func (s *PaymentService) GenerateFakeSimulatedWebhook(ctx context.Context, paymentNo string) ([]byte, string, error) {
	payment, err := s.paymentRepo.GetPaymentByPaymentNo(ctx, paymentNo)
	if err != nil {
		return nil, "", err
	}
	return s.fakeGateway.GenerateSimulatedWebhook(payment.PaymentNo, payment.OrderID, payment.AmountMinor, payment.Currency)
}
