package commerce

import (
	"context"

	"github.com/google/uuid"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, p *Product) (*Product, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	ListActiveProductsWithPlans(ctx context.Context) ([]*Product, error)
	ListAllProducts(ctx context.Context) ([]*Product, error)
	UpdateProduct(ctx context.Context, p *Product) (*Product, error)

	CreatePlan(ctx context.Context, plan *Plan) (*Plan, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*Plan, error)
	GetPlanByProductAndSlug(ctx context.Context, productID uuid.UUID, slug string) (*Plan, error)
	ListPlansByProductID(ctx context.Context, productID uuid.UUID) ([]*Plan, error)
	ListActivePlansByProductID(ctx context.Context, productID uuid.UUID) ([]*Plan, error)
	UpdatePlan(ctx context.Context, plan *Plan) (*Plan, error)
}

type OrderRepository interface {
	CreateOrderWithItems(ctx context.Context, order *Order, items []OrderItem, invoice *Invoice, invoiceItem *InvoiceItem) (*Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetOrderByOrderNo(ctx context.Context, orderNo string) (*Order, error)
	ListOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*Order, error)
	ListAllOrders(ctx context.Context) ([]*Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status OrderStatus) (*Order, error)
}

type PaymentSuccessParams struct {
	PaymentID        uuid.UUID
	GatewayPaymentID string
	GatewayPayload   []byte
	PaymentNo        string
	OrderID          uuid.UUID
	AmountMinor      int64
	Currency         string
	UserID           uuid.UUID
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, p *Payment) (*Payment, error)
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetPaymentByPaymentNo(ctx context.Context, paymentNo string) (*Payment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, key string) (*Payment, error)
	GetPaymentByGatewayAndExternalID(ctx context.Context, gateway, externalID string) (*Payment, error)
	ListPaymentsByOrderID(ctx context.Context, orderID uuid.UUID) ([]*Payment, error)
	ListAllPayments(ctx context.Context) ([]*Payment, error)

	// ProcessPaymentSuccessTx executes Payment + Order + Invoice + Ledger + Outbox in one atomic transaction with idempotency protection.
	// Returns (alreadyProcessed bool, err error).
	ProcessPaymentSuccessTx(ctx context.Context, params PaymentSuccessParams) (bool, error)
}

type InvoiceRepository interface {
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*Invoice, error)
	ListInvoicesByUserID(ctx context.Context, userID uuid.UUID) ([]*Invoice, error)
	ListAllInvoices(ctx context.Context) ([]*Invoice, error)
}

type WalletRepository interface {
	GetOrCreateWallet(ctx context.Context, userID uuid.UUID, currency string) (*Wallet, error)
	ListLedgerTransactions(ctx context.Context, limit, offset int) ([]*LedgerTransaction, error)
	ListUserLedgerEntries(ctx context.Context, userID uuid.UUID) ([]*LedgerEntry, error)
}
