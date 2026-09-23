package commerce

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type CreateOrderItemInput struct {
	PlanID   uuid.UUID `json:"plan_id"`
	Quantity int       `json:"quantity"`
}

type OrderService struct {
	orderRepo   domainCommerce.OrderRepository
	productRepo domainCommerce.ProductRepository
}

func NewOrderService(
	orderRepo domainCommerce.OrderRepository,
	productRepo domainCommerce.ProductRepository,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, inputs []CreateOrderItemInput) (*domainCommerce.Order, error) {
	if len(inputs) == 0 {
		return nil, domainCommerce.ErrEmptyOrderItems
	}

	orderID := uuid.New()
	orderNo := fmt.Sprintf("ORD-%s-%s", time.Now().UTC().Format("20060102"), uuid.New().String()[:8])

	var (
		subtotalMinor int64
		currency      string
		orderItems    = make([]domainCommerce.OrderItem, 0, len(inputs))
	)

	for _, input := range inputs {
		if input.Quantity <= 0 {
			return nil, fmt.Errorf("invalid item quantity: %d", input.Quantity)
		}

		plan, err := s.productRepo.GetPlanByID(ctx, input.PlanID)
		if err != nil {
			return nil, fmt.Errorf("failed to get plan %s: %w", input.PlanID, err)
		}
		if plan.Status != "active" {
			return nil, domainCommerce.ErrPlanInactive
		}

		product, err := s.productRepo.GetProductByID(ctx, plan.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to get product %s: %w", plan.ProductID, err)
		}
		if product.Status != "active" {
			return nil, domainCommerce.ErrProductNotFound
		}

		if currency == "" {
			currency = plan.Currency
		} else if currency != plan.Currency {
			return nil, fmt.Errorf("%w: mixed currencies in single order", domainCommerce.ErrInvalidCurrency)
		}

		itemTotalMinor := plan.PriceMinor * int64(input.Quantity)
		subtotalMinor += itemTotalMinor

		prodSnapshot, _ := json.Marshal(product)
		planSnapshot, _ := json.Marshal(plan)

		orderItems = append(orderItems, domainCommerce.OrderItem{
			ID:              uuid.New(),
			OrderID:         orderID,
			ProductID:       product.ID,
			PlanID:          plan.ID,
			Quantity:        input.Quantity,
			UnitPriceMinor:  plan.PriceMinor,
			TotalMinor:      itemTotalMinor,
			ProductSnapshot: prodSnapshot,
			PlanSnapshot:    planSnapshot,
		})
	}

	order := &domainCommerce.Order{
		ID:            orderID,
		OrderNo:       orderNo,
		UserID:        userID,
		Status:        domainCommerce.OrderStatusPending,
		SubtotalMinor: subtotalMinor,
		DiscountMinor: 0,
		TotalMinor:    subtotalMinor,
		Currency:      currency,
	}

	// Create pending invoice alongside order
	invoiceID := uuid.New()
	invoiceNo := fmt.Sprintf("INV-%s-%s", time.Now().UTC().Format("20060102"), uuid.New().String()[:8])
	dueAt := time.Now().UTC().Add(24 * time.Hour)

	invoice := &domainCommerce.Invoice{
		ID:          invoiceID,
		InvoiceNo:   invoiceNo,
		UserID:      userID,
		OrderID:     &orderID,
		Status:      domainCommerce.InvoiceStatusPending,
		AmountMinor: subtotalMinor,
		Currency:    currency,
		DueAt:       &dueAt,
	}

	invDesc, _ := json.Marshal(map[string]string{
		"en-US": fmt.Sprintf("Order %s", orderNo),
		"zh-CN": fmt.Sprintf("订单 %s", orderNo),
	})
	invoiceItem := &domainCommerce.InvoiceItem{
		ID:              uuid.New(),
		InvoiceID:       invoiceID,
		DescriptionI18n: invDesc,
		Quantity:        1,
		UnitAmountMinor: subtotalMinor,
		TotalMinor:      subtotalMinor,
	}

	created, err := s.orderRepo.CreateOrderWithItems(ctx, order, orderItems, invoice, invoiceItem)
	if err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	return created, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*domainCommerce.Order, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domainCommerce.ErrOrderNotFound
	}
	return order, nil
}

func (s *OrderService) GetOrderByOrderNo(ctx context.Context, userID uuid.UUID, orderNo string) (*domainCommerce.Order, error) {
	order, err := s.orderRepo.GetOrderByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domainCommerce.ErrOrderNotFound
	}
	return order, nil
}

func (s *OrderService) ListUserOrders(ctx context.Context, userID uuid.UUID) ([]*domainCommerce.Order, error) {
	return s.orderRepo.ListOrdersByUserID(ctx, userID)
}

func (s *OrderService) ListAllOrders(ctx context.Context) ([]*domainCommerce.Order, error) {
	return s.orderRepo.ListAllOrders(ctx)
}

func (s *OrderService) CancelOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*domainCommerce.Order, error) {
	order, err := s.GetOrderByID(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != domainCommerce.OrderStatusPending {
		return nil, domainCommerce.ErrInvalidOrderStatus
	}

	return s.orderRepo.UpdateOrderStatus(ctx, orderID, domainCommerce.OrderStatusCancelled)
}
