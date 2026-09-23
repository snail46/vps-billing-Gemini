package commerce

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Well-known system accounts for double-entry ledger
var (
	SystemAccountPaymentGateway  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	SystemAccountPlatformRevenue = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

type AccountType string

const (
	AccountPaymentGateway     AccountType = "payment_gateway"
	AccountPlatformRevenue    AccountType = "platform_revenue"
	AccountUserWallet         AccountType = "user_wallet"
	AccountAccountsReceivable AccountType = "accounts_receivable"
)

type LedgerDirection string

const (
	DirectionDebit  LedgerDirection = "debit"
	DirectionCredit LedgerDirection = "credit"
)

type Product struct {
	ID              uuid.UUID       `json:"id"`
	Slug            string          `json:"slug"`
	NameI18n        json.RawMessage `json:"name_i18n"`
	DescriptionI18n json.RawMessage `json:"description_i18n"`
	Status          string          `json:"status"`
	SortOrder       int             `json:"sort_order"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Plans           []Plan          `json:"plans,omitempty"`
}

type Plan struct {
	ID             uuid.UUID       `json:"id"`
	ProductID      uuid.UUID       `json:"product_id"`
	NodeGroupID    *uuid.UUID      `json:"node_group_id,omitempty"`
	Slug           string          `json:"slug"`
	NameI18n       json.RawMessage `json:"name_i18n"`
	Status         string          `json:"status"`
	CPUCores       float64         `json:"cpu_cores"`
	MemoryMB       int             `json:"memory_mb"`
	DiskGB         int             `json:"disk_gb"`
	TrafficGB      *int64          `json:"traffic_gb,omitempty"`
	BandwidthMbps  *int            `json:"bandwidth_mbps,omitempty"`
	IPv4Count      int             `json:"ipv4_count"`
	IPv6Count      int             `json:"ipv6_count"`
	NatPortCount   int             `json:"nat_port_count"`
	Virtualization string          `json:"virtualization"`
	BillingCycle   string          `json:"billing_cycle"`
	PriceMinor     int64           `json:"price_minor"`
	Currency       string          `json:"currency"`
	StockMode      string          `json:"stock_mode"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type OrderStatus string

const (
	OrderStatusPending       OrderStatus = "pending"
	OrderStatusPaid          OrderStatus = "paid"
	OrderStatusFulfilling    OrderStatus = "fulfilling"
	OrderStatusFulfilled     OrderStatus = "fulfilled"
	OrderStatusCancelled     OrderStatus = "cancelled"
	OrderStatusRefundPending OrderStatus = "refund_pending"
	OrderStatusRefunded      OrderStatus = "refunded"
)

type Order struct {
	ID            uuid.UUID   `json:"id"`
	OrderNo       string      `json:"order_no"`
	UserID        uuid.UUID   `json:"user_id"`
	Status        OrderStatus `json:"status"`
	SubtotalMinor int64       `json:"subtotal_minor"`
	DiscountMinor int64       `json:"discount_minor"`
	TotalMinor    int64       `json:"total_minor"`
	Currency      string      `json:"currency"`
	PaidAt        *time.Time  `json:"paid_at,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	Items         []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID              uuid.UUID       `json:"id"`
	OrderID         uuid.UUID       `json:"order_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	PlanID          uuid.UUID       `json:"plan_id"`
	Quantity        int             `json:"quantity"`
	UnitPriceMinor  int64           `json:"unit_price_minor"`
	TotalMinor      int64           `json:"total_minor"`
	ProductSnapshot json.RawMessage `json:"product_snapshot"`
	PlanSnapshot    json.RawMessage `json:"plan_snapshot"`
	CreatedAt       time.Time       `json:"created_at"`
}

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type Payment struct {
	ID               uuid.UUID       `json:"id"`
	PaymentNo        string          `json:"payment_no"`
	OrderID          uuid.UUID       `json:"order_id"`
	Gateway          string          `json:"gateway"`
	GatewayPaymentID *string         `json:"gateway_payment_id,omitempty"`
	Status           PaymentStatus   `json:"status"`
	AmountMinor      int64           `json:"amount_minor"`
	Currency         string          `json:"currency"`
	IdempotencyKey   string          `json:"idempotency_key"`
	GatewayPayload   json.RawMessage `json:"gateway_payload"`
	PaidAt           *time.Time      `json:"paid_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type InvoiceStatus string

const (
	InvoiceStatusPending   InvoiceStatus = "pending"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
	InvoiceStatusRefunded  InvoiceStatus = "refunded"
)

type Invoice struct {
	ID             uuid.UUID     `json:"id"`
	InvoiceNo      string        `json:"invoice_no"`
	UserID         uuid.UUID     `json:"user_id"`
	SubscriptionID *uuid.UUID    `json:"subscription_id,omitempty"`
	OrderID        *uuid.UUID    `json:"order_id,omitempty"`
	Status         InvoiceStatus `json:"status"`
	AmountMinor    int64         `json:"amount_minor"`
	Currency       string        `json:"currency"`
	DueAt          *time.Time    `json:"due_at,omitempty"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Items          []InvoiceItem `json:"items,omitempty"`
}

type InvoiceItem struct {
	ID              uuid.UUID       `json:"id"`
	InvoiceID       uuid.UUID       `json:"invoice_id"`
	DescriptionI18n json.RawMessage `json:"description_i18n"`
	Quantity        int             `json:"quantity"`
	UnitAmountMinor int64           `json:"unit_amount_minor"`
	TotalMinor      int64           `json:"total_minor"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Wallet struct {
	ID                    uuid.UUID `json:"id"`
	UserID                uuid.UUID `json:"user_id"`
	Currency              string    `json:"currency"`
	AvailableBalanceMinor int64     `json:"available_balance_minor"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type LedgerTransaction struct {
	ID            uuid.UUID     `json:"id"`
	Type          string        `json:"type"`
	ReferenceType *string       `json:"reference_type,omitempty"`
	ReferenceID   *uuid.UUID    `json:"reference_id,omitempty"`
	Description   *string       `json:"description,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	Entries       []LedgerEntry `json:"entries,omitempty"`
}

type LedgerEntry struct {
	ID            uuid.UUID       `json:"id"`
	TransactionID uuid.UUID       `json:"transaction_id"`
	AccountType   AccountType     `json:"account_type"`
	AccountID     uuid.UUID       `json:"account_id"`
	Direction     LedgerDirection `json:"direction"`
	AmountMinor   int64           `json:"amount_minor"`
	Currency      string          `json:"currency"`
	CreatedAt     time.Time       `json:"created_at"`
}

type OutboxEvent struct {
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	Attempts      int             `json:"attempts"`
	NextAttemptAt time.Time       `json:"next_attempt_at"`
	CreatedAt     time.Time       `json:"created_at"`
	PublishedAt   *time.Time      `json:"published_at,omitempty"`
}
