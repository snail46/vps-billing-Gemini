package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type PostgresCommerceRepository struct {
	pool    *pgxpool.Pool
	queries *Queries
}

func NewPostgresCommerceRepository(pool *pgxpool.Pool, queries *Queries) *PostgresCommerceRepository {
	return &PostgresCommerceRepository{
		pool:    pool,
		queries: queries,
	}
}

// Ensure interface implementations
var (
	_ domainCommerce.ProductRepository = (*PostgresCommerceRepository)(nil)
	_ domainCommerce.OrderRepository   = (*PostgresCommerceRepository)(nil)
	_ domainCommerce.PaymentRepository = (*PostgresCommerceRepository)(nil)
	_ domainCommerce.InvoiceRepository = (*PostgresCommerceRepository)(nil)
	_ domainCommerce.WalletRepository  = (*PostgresCommerceRepository)(nil)
)

// --- Product & Plan Methods ---

func toDomainProduct(p Products) *domainCommerce.Product {
	return &domainCommerce.Product{
		ID:              fromPgUUID(p.ID),
		Slug:            p.Slug,
		NameI18n:        p.NameI18n,
		DescriptionI18n: p.DescriptionI18n,
		Status:          p.Status,
		SortOrder:       int(p.SortOrder),
		CreatedAt:       p.CreatedAt.Time,
		UpdatedAt:       p.UpdatedAt.Time,
	}
}

func toDomainPlan(p Plans) *domainCommerce.Plan {
	return &domainCommerce.Plan{
		ID:             fromPgUUID(p.ID),
		ProductID:      fromPgUUID(p.ProductID),
		NodeGroupID:    fromPgUUIDPtr(p.NodeGroupID),
		Slug:           p.Slug,
		NameI18n:       p.NameI18n,
		Status:         p.Status,
		CPUCores:       fromPgNumeric(p.CpuCores),
		MemoryMB:       int(p.MemoryMb),
		DiskGB:         int(p.DiskGb),
		TrafficGB:      fromPgInt8(p.TrafficGb),
		BandwidthMbps:  fromPgInt4(p.BandwidthMbps),
		IPv4Count:      int(p.Ipv4Count),
		IPv6Count:      int(p.Ipv6Count),
		NatPortCount:   int(p.NatPortCount),
		Virtualization: p.Virtualization,
		BillingCycle:   p.BillingCycle,
		PriceMinor:     p.PriceMinor,
		Currency:       p.Currency,
		StockMode:      p.StockMode,
		CreatedAt:      p.CreatedAt.Time,
		UpdatedAt:      p.UpdatedAt.Time,
	}
}

func (r *PostgresCommerceRepository) CreateProduct(ctx context.Context, p *domainCommerce.Product) (*domainCommerce.Product, error) {
	row, err := r.queries.CreateProduct(ctx, CreateProductParams{
		ID:              toPgUUID(p.ID),
		Slug:            p.Slug,
		NameI18n:        p.NameI18n,
		DescriptionI18n: p.DescriptionI18n,
		Status:          p.Status,
		SortOrder:       int32(p.SortOrder),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}
	return toDomainProduct(row), nil
}

func (r *PostgresCommerceRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*domainCommerce.Product, error) {
	row, err := r.queries.GetProductByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrProductNotFound
		}
		return nil, err
	}
	prod := toDomainProduct(row)
	plans, err := r.ListPlansByProductID(ctx, prod.ID)
	if err == nil {
		for _, pl := range plans {
			prod.Plans = append(prod.Plans, *pl)
		}
	}
	return prod, nil
}

func (r *PostgresCommerceRepository) GetProductBySlug(ctx context.Context, slug string) (*domainCommerce.Product, error) {
	row, err := r.queries.GetProductBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrProductNotFound
		}
		return nil, err
	}
	prod := toDomainProduct(row)
	plans, err := r.ListPlansByProductID(ctx, prod.ID)
	if err == nil {
		for _, pl := range plans {
			prod.Plans = append(prod.Plans, *pl)
		}
	}
	return prod, nil
}

func (r *PostgresCommerceRepository) ListActiveProductsWithPlans(ctx context.Context) ([]*domainCommerce.Product, error) {
	rows, err := r.queries.ListActiveProducts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Product, 0, len(rows))
	for _, row := range rows {
		prod := toDomainProduct(row)
		plans, err := r.ListActivePlansByProductID(ctx, prod.ID)
		if err == nil {
			for _, pl := range plans {
				prod.Plans = append(prod.Plans, *pl)
			}
		}
		result = append(result, prod)
	}
	return result, nil
}

func (r *PostgresCommerceRepository) ListAllProducts(ctx context.Context) ([]*domainCommerce.Product, error) {
	rows, err := r.queries.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Product, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainProduct(row))
	}
	return result, nil
}

func (r *PostgresCommerceRepository) UpdateProduct(ctx context.Context, p *domainCommerce.Product) (*domainCommerce.Product, error) {
	row, err := r.queries.UpdateProduct(ctx, UpdateProductParams{
		ID:              toPgUUID(p.ID),
		NameI18n:        p.NameI18n,
		DescriptionI18n: p.DescriptionI18n,
		Status:          p.Status,
		SortOrder:       int32(p.SortOrder),
	})
	if err != nil {
		return nil, err
	}
	return toDomainProduct(row), nil
}

func (r *PostgresCommerceRepository) CreatePlan(ctx context.Context, p *domainCommerce.Plan) (*domainCommerce.Plan, error) {
	row, err := r.queries.CreatePlan(ctx, CreatePlanParams{
		ID:             toPgUUID(p.ID),
		ProductID:      toPgUUID(p.ProductID),
		NodeGroupID:    toPgUUIDPtr(p.NodeGroupID),
		Slug:           p.Slug,
		NameI18n:       p.NameI18n,
		Status:         p.Status,
		CpuCores:       toPgNumeric(p.CPUCores),
		MemoryMb:       int32(p.MemoryMB),
		DiskGb:         int32(p.DiskGB),
		TrafficGb:      toPgInt8(p.TrafficGB),
		BandwidthMbps:  toPgInt4(p.BandwidthMbps),
		Ipv4Count:      int32(p.IPv4Count),
		Ipv6Count:      int32(p.IPv6Count),
		NatPortCount:   int32(p.NatPortCount),
		Virtualization: p.Virtualization,
		BillingCycle:   p.BillingCycle,
		PriceMinor:     p.PriceMinor,
		Currency:       p.Currency,
		StockMode:      p.StockMode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	return toDomainPlan(row), nil
}

func (r *PostgresCommerceRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (*domainCommerce.Plan, error) {
	row, err := r.queries.GetPlanByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrPlanNotFound
		}
		return nil, err
	}
	return toDomainPlan(row), nil
}

func (r *PostgresCommerceRepository) GetPlanByProductAndSlug(ctx context.Context, productID uuid.UUID, slug string) (*domainCommerce.Plan, error) {
	row, err := r.queries.GetPlanByProductAndSlug(ctx, GetPlanByProductAndSlugParams{
		ProductID: toPgUUID(productID),
		Slug:      slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrPlanNotFound
		}
		return nil, err
	}
	return toDomainPlan(row), nil
}

func (r *PostgresCommerceRepository) ListPlansByProductID(ctx context.Context, productID uuid.UUID) ([]*domainCommerce.Plan, error) {
	rows, err := r.queries.ListPlansByProductID(ctx, toPgUUID(productID))
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Plan, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainPlan(row))
	}
	return result, nil
}

func (r *PostgresCommerceRepository) ListActivePlansByProductID(ctx context.Context, productID uuid.UUID) ([]*domainCommerce.Plan, error) {
	rows, err := r.queries.ListActivePlansByProductID(ctx, toPgUUID(productID))
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Plan, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainPlan(row))
	}
	return result, nil
}

func (r *PostgresCommerceRepository) UpdatePlan(ctx context.Context, p *domainCommerce.Plan) (*domainCommerce.Plan, error) {
	row, err := r.queries.UpdatePlan(ctx, UpdatePlanParams{
		ID:            toPgUUID(p.ID),
		NameI18n:      p.NameI18n,
		Status:        p.Status,
		CpuCores:      toPgNumeric(p.CPUCores),
		MemoryMb:      int32(p.MemoryMB),
		DiskGb:        int32(p.DiskGB),
		TrafficGb:     toPgInt8(p.TrafficGB),
		BandwidthMbps: toPgInt4(p.BandwidthMbps),
		Ipv4Count:     int32(p.IPv4Count),
		Ipv6Count:     int32(p.IPv6Count),
		NatPortCount:  int32(p.NatPortCount),
		PriceMinor:    p.PriceMinor,
		Currency:      p.Currency,
		StockMode:     p.StockMode,
	})
	if err != nil {
		return nil, err
	}
	return toDomainPlan(row), nil
}

// --- Order Methods ---

func toDomainOrder(o Orders) *domainCommerce.Order {
	return &domainCommerce.Order{
		ID:            fromPgUUID(o.ID),
		OrderNo:       o.OrderNo,
		UserID:        fromPgUUID(o.UserID),
		Status:        domainCommerce.OrderStatus(o.Status),
		SubtotalMinor: o.SubtotalMinor,
		DiscountMinor: o.DiscountMinor,
		TotalMinor:    o.TotalMinor,
		Currency:      o.Currency,
		PaidAt:        fromPgTimestamptz(o.PaidAt),
		CreatedAt:     o.CreatedAt.Time,
		UpdatedAt:     o.UpdatedAt.Time,
	}
}

func toDomainOrderItem(item OrderItems) domainCommerce.OrderItem {
	return domainCommerce.OrderItem{
		ID:              fromPgUUID(item.ID),
		OrderID:         fromPgUUID(item.OrderID),
		ProductID:       fromPgUUID(item.ProductID),
		PlanID:          fromPgUUID(item.PlanID),
		Quantity:        int(item.Quantity),
		UnitPriceMinor:  item.UnitPriceMinor,
		TotalMinor:      item.TotalMinor,
		ProductSnapshot: item.ProductSnapshot,
		PlanSnapshot:    item.PlanSnapshot,
		CreatedAt:       item.CreatedAt.Time,
	}
}

func (r *PostgresCommerceRepository) CreateOrderWithItems(
	ctx context.Context,
	order *domainCommerce.Order,
	items []domainCommerce.OrderItem,
	invoice *domainCommerce.Invoice,
	invoiceItem *domainCommerce.InvoiceItem,
) (*domainCommerce.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Create order
	ordRow, err := qtx.CreateOrder(ctx, CreateOrderParams{
		ID:            toPgUUID(order.ID),
		OrderNo:       order.OrderNo,
		UserID:        toPgUUID(order.UserID),
		Status:        string(order.Status),
		SubtotalMinor: order.SubtotalMinor,
		DiscountMinor: order.DiscountMinor,
		TotalMinor:    order.TotalMinor,
		Currency:      order.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	createdOrder := toDomainOrder(ordRow)

	// 2. Create order items
	for _, item := range items {
		itemRow, err := qtx.CreateOrderItem(ctx, CreateOrderItemParams{
			ID:              toPgUUID(item.ID),
			OrderID:         toPgUUID(order.ID),
			ProductID:       toPgUUID(item.ProductID),
			PlanID:          toPgUUID(item.PlanID),
			Quantity:        int32(item.Quantity),
			UnitPriceMinor:  item.UnitPriceMinor,
			TotalMinor:      item.TotalMinor,
			ProductSnapshot: item.ProductSnapshot,
			PlanSnapshot:    item.PlanSnapshot,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
		createdOrder.Items = append(createdOrder.Items, toDomainOrderItem(itemRow))
	}

	// 3. Create initial pending invoice if provided
	if invoice != nil {
		_, err = qtx.CreateInvoice(ctx, CreateInvoiceParams{
			ID:             toPgUUID(invoice.ID),
			InvoiceNo:      invoice.InvoiceNo,
			UserID:         toPgUUID(invoice.UserID),
			SubscriptionID: toPgUUIDPtr(invoice.SubscriptionID),
			OrderID:        toPgUUIDPtr(invoice.OrderID),
			Status:         string(invoice.Status),
			AmountMinor:    invoice.AmountMinor,
			Currency:       invoice.Currency,
			DueAt:          toPgTimestamptz(invoice.DueAt),
			PaidAt:         toPgTimestamptz(invoice.PaidAt),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to insert invoice: %w", err)
		}

		if invoiceItem != nil {
			_, err = qtx.CreateInvoiceItem(ctx, CreateInvoiceItemParams{
				ID:              toPgUUID(invoiceItem.ID),
				InvoiceID:       toPgUUID(invoice.ID),
				DescriptionI18n: invoiceItem.DescriptionI18n,
				Quantity:        int32(invoiceItem.Quantity),
				UnitAmountMinor: invoiceItem.UnitAmountMinor,
				TotalMinor:      invoiceItem.TotalMinor,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to insert invoice item: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit order tx: %w", err)
	}

	return createdOrder, nil
}

func (r *PostgresCommerceRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*domainCommerce.Order, error) {
	row, err := r.queries.GetOrderByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrOrderNotFound
		}
		return nil, err
	}
	ord := toDomainOrder(row)
	items, err := r.queries.ListOrderItemsByOrderID(ctx, row.ID)
	if err == nil {
		for _, item := range items {
			ord.Items = append(ord.Items, toDomainOrderItem(item))
		}
	}
	return ord, nil
}

func (r *PostgresCommerceRepository) GetOrderByOrderNo(ctx context.Context, orderNo string) (*domainCommerce.Order, error) {
	row, err := r.queries.GetOrderByOrderNo(ctx, orderNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrOrderNotFound
		}
		return nil, err
	}
	ord := toDomainOrder(row)
	items, err := r.queries.ListOrderItemsByOrderID(ctx, row.ID)
	if err == nil {
		for _, item := range items {
			ord.Items = append(ord.Items, toDomainOrderItem(item))
		}
	}
	return ord, nil
}

func (r *PostgresCommerceRepository) ListOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*domainCommerce.Order, error) {
	rows, err := r.queries.ListOrdersByUserID(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Order, 0, len(rows))
	for _, row := range rows {
		ord := toDomainOrder(row)
		items, _ := r.queries.ListOrderItemsByOrderID(ctx, row.ID)
		for _, item := range items {
			ord.Items = append(ord.Items, toDomainOrderItem(item))
		}
		result = append(result, ord)
	}
	return result, nil
}

func (r *PostgresCommerceRepository) ListAllOrders(ctx context.Context) ([]*domainCommerce.Order, error) {
	rows, err := r.queries.ListAllOrders(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Order, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainOrder(row))
	}
	return result, nil
}

func (r *PostgresCommerceRepository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status domainCommerce.OrderStatus) (*domainCommerce.Order, error) {
	row, err := r.queries.UpdateOrderStatus(ctx, UpdateOrderStatusParams{
		ID:     toPgUUID(id),
		Status: string(status),
	})
	if err != nil {
		return nil, err
	}
	return toDomainOrder(row), nil
}

// --- Payment Methods ---

func toDomainPayment(p Payments) *domainCommerce.Payment {
	return &domainCommerce.Payment{
		ID:               fromPgUUID(p.ID),
		PaymentNo:        p.PaymentNo,
		OrderID:          fromPgUUID(p.OrderID),
		Gateway:          p.Gateway,
		GatewayPaymentID: fromPgText(p.GatewayPaymentID),
		Status:           domainCommerce.PaymentStatus(p.Status),
		AmountMinor:      p.AmountMinor,
		Currency:         p.Currency,
		IdempotencyKey:   p.IdempotencyKey,
		GatewayPayload:   p.GatewayPayload,
		PaidAt:           fromPgTimestamptz(p.PaidAt),
		CreatedAt:        p.CreatedAt.Time,
		UpdatedAt:        p.UpdatedAt.Time,
	}
}

func (r *PostgresCommerceRepository) CreatePayment(ctx context.Context, p *domainCommerce.Payment) (*domainCommerce.Payment, error) {
	var gPayID pgtype.Text
	if p.GatewayPaymentID != nil {
		gPayID = toPgText(*p.GatewayPaymentID)
	}
	row, err := r.queries.CreatePayment(ctx, CreatePaymentParams{
		ID:               toPgUUID(p.ID),
		PaymentNo:        p.PaymentNo,
		OrderID:          toPgUUID(p.OrderID),
		Gateway:          p.Gateway,
		GatewayPaymentID: gPayID,
		Status:           string(p.Status),
		AmountMinor:      p.AmountMinor,
		Currency:         p.Currency,
		IdempotencyKey:   p.IdempotencyKey,
		GatewayPayload:   p.GatewayPayload,
		PaidAt:           toPgTimestamptz(p.PaidAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}
	return toDomainPayment(row), nil
}

func (r *PostgresCommerceRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*domainCommerce.Payment, error) {
	row, err := r.queries.GetPaymentByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrPaymentNotFound
		}
		return nil, err
	}
	return toDomainPayment(row), nil
}

func (r *PostgresCommerceRepository) GetPaymentByPaymentNo(ctx context.Context, paymentNo string) (*domainCommerce.Payment, error) {
	row, err := r.queries.GetPaymentByPaymentNo(ctx, paymentNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrPaymentNotFound
		}
		return nil, err
	}
	return toDomainPayment(row), nil
}

func (r *PostgresCommerceRepository) GetPaymentByIdempotencyKey(ctx context.Context, key string) (*domainCommerce.Payment, error) {
	row, err := r.queries.GetPaymentByIdempotencyKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrPaymentNotFound
		}
		return nil, err
	}
	return toDomainPayment(row), nil
}

func (r *PostgresCommerceRepository) GetPaymentByGatewayAndExternalID(ctx context.Context, gateway, externalID string) (*domainCommerce.Payment, error) {
	row, err := r.queries.GetPaymentByGatewayAndExternalID(ctx, GetPaymentByGatewayAndExternalIDParams{
		Gateway:          gateway,
		GatewayPaymentID: toPgText(externalID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrPaymentNotFound
		}
		return nil, err
	}
	return toDomainPayment(row), nil
}

func (r *PostgresCommerceRepository) ListPaymentsByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domainCommerce.Payment, error) {
	rows, err := r.queries.ListPaymentsByOrderID(ctx, toPgUUID(orderID))
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Payment, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainPayment(row))
	}
	return result, nil
}

func (r *PostgresCommerceRepository) ListAllPayments(ctx context.Context) ([]*domainCommerce.Payment, error) {
	rows, err := r.queries.ListAllPayments(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Payment, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainPayment(row))
	}
	return result, nil
}

// ProcessPaymentSuccessTx executes Payment + Order + Invoice + Ledger + Outbox in one atomic transaction with idempotency protection.
// Returns (alreadyProcessed bool, err error).
func (r *PostgresCommerceRepository) ProcessPaymentSuccessTx(ctx context.Context, params domainCommerce.PaymentSuccessParams) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Lock payment row FOR UPDATE by ID to serialize concurrent callbacks
	paymentRow, err := qtx.GetPaymentForUpdate(ctx, toPgUUID(params.PaymentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, domainCommerce.ErrPaymentNotFound
		}
		return false, fmt.Errorf("failed to lock payment: %w", err)
	}

	// If payment is already succeeded, return immediately with alreadyProcessed = true (100% idempotent)
	if paymentRow.Status == string(domainCommerce.PaymentStatusSucceeded) {
		_ = tx.Commit(ctx)
		return true, nil
	}

	// 2. Update payment status to 'succeeded'
	_, err = qtx.UpdatePaymentStatus(ctx, UpdatePaymentStatusParams{
		ID:               toPgUUID(params.PaymentID),
		Status:           string(domainCommerce.PaymentStatusSucceeded),
		GatewayPaymentID: toPgText(params.GatewayPaymentID),
		GatewayPayload:   params.GatewayPayload,
	})
	if err != nil {
		return false, fmt.Errorf("failed to update payment: %w", err)
	}

	// 3. Update order status to 'paid'
	_, err = qtx.UpdateOrderStatus(ctx, UpdateOrderStatusParams{
		ID:     toPgUUID(params.OrderID),
		Status: string(domainCommerce.OrderStatusPaid),
	})
	if err != nil {
		return false, fmt.Errorf("failed to update order: %w", err)
	}

	// 4. Update invoice to 'paid' if exists
	inv, err := qtx.GetInvoiceByOrderID(ctx, toPgUUID(params.OrderID))
	if err == nil {
		_, _ = qtx.UpdateInvoiceStatus(ctx, UpdateInvoiceStatusParams{
			ID:     inv.ID,
			Status: string(domainCommerce.InvoiceStatusPaid),
		})
	}

	// 5. Create balanced Double-Entry Ledger Transaction & Entries
	// Invariants:
	// - Sum(Debits) == Sum(Credits)
	// - Immutable: no updates to ledger rows
	ledgerTxID := uuid.New()
	refType := "payment"
	refDesc := fmt.Sprintf("Payment %s for Order %s", params.PaymentNo, params.OrderID)
	_, err = qtx.CreateLedgerTransaction(ctx, CreateLedgerTransactionParams{
		ID:            toPgUUID(ledgerTxID),
		Type:          "order_payment",
		ReferenceType: toPgText(refType),
		ReferenceID:   toPgUUID(params.PaymentID),
		Description:   toPgText(refDesc),
	})
	if err != nil {
		return false, fmt.Errorf("failed to create ledger transaction: %w", err)
	}

	// Entry 1 (Debit): Gateway account receives money
	_, err = qtx.CreateLedgerEntry(ctx, CreateLedgerEntryParams{
		ID:            toPgUUID(uuid.New()),
		TransactionID: toPgUUID(ledgerTxID),
		AccountType:   string(domainCommerce.AccountPaymentGateway),
		AccountID:     toPgUUID(domainCommerce.SystemAccountPaymentGateway),
		Direction:     string(domainCommerce.DirectionDebit),
		AmountMinor:   params.AmountMinor,
		Currency:      params.Currency,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create debit ledger entry: %w", err)
	}

	// Entry 2 (Credit): Platform revenue earns money
	_, err = qtx.CreateLedgerEntry(ctx, CreateLedgerEntryParams{
		ID:            toPgUUID(uuid.New()),
		TransactionID: toPgUUID(ledgerTxID),
		AccountType:   string(domainCommerce.AccountPlatformRevenue),
		AccountID:     toPgUUID(domainCommerce.SystemAccountPlatformRevenue),
		Direction:     string(domainCommerce.DirectionCredit),
		AmountMinor:   params.AmountMinor,
		Currency:      params.Currency,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create credit ledger entry: %w", err)
	}

	// 6. Write Transactional Outbox Event for downstream workers/workflows
	outboxPayload, _ := json.Marshal(map[string]any{
		"payment_id":         params.PaymentID.String(),
		"payment_no":         params.PaymentNo,
		"order_id":           params.OrderID.String(),
		"user_id":            params.UserID.String(),
		"amount_minor":       params.AmountMinor,
		"currency":           params.Currency,
		"gateway_payment_id": params.GatewayPaymentID,
		"paid_at":            time.Now().UTC().Format(time.RFC3339),
	})
	_, err = qtx.CreateOutboxEvent(ctx, CreateOutboxEventParams{
		ID:            toPgUUID(uuid.New()),
		EventType:     "payment.succeeded.v1",
		AggregateType: "payment",
		AggregateID:   toPgUUID(params.PaymentID),
		Payload:       outboxPayload,
		Status:        "pending",
		Attempts:      0,
		NextAttemptAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return false, nil
}

// --- Invoice Methods ---

func toDomainInvoice(i Invoices) *domainCommerce.Invoice {
	return &domainCommerce.Invoice{
		ID:             fromPgUUID(i.ID),
		InvoiceNo:      i.InvoiceNo,
		UserID:         fromPgUUID(i.UserID),
		SubscriptionID: fromPgUUIDPtr(i.SubscriptionID),
		OrderID:        fromPgUUIDPtr(i.OrderID),
		Status:         domainCommerce.InvoiceStatus(i.Status),
		AmountMinor:    i.AmountMinor,
		Currency:       i.Currency,
		DueAt:          fromPgTimestamptz(i.DueAt),
		PaidAt:         fromPgTimestamptz(i.PaidAt),
		CreatedAt:      i.CreatedAt.Time,
		UpdatedAt:      i.UpdatedAt.Time,
	}
}

func (r *PostgresCommerceRepository) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*domainCommerce.Invoice, error) {
	row, err := r.queries.GetInvoiceByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrInvoiceNotFound
		}
		return nil, err
	}
	inv := toDomainInvoice(row)
	items, err := r.queries.ListInvoiceItemsByInvoiceID(ctx, row.ID)
	if err == nil {
		for _, item := range items {
			inv.Items = append(inv.Items, domainCommerce.InvoiceItem{
				ID:              fromPgUUID(item.ID),
				InvoiceID:       fromPgUUID(item.InvoiceID),
				DescriptionI18n: item.DescriptionI18n,
				Quantity:        int(item.Quantity),
				UnitAmountMinor: item.UnitAmountMinor,
				TotalMinor:      item.TotalMinor,
				CreatedAt:       item.CreatedAt.Time,
			})
		}
	}
	return inv, nil
}

func (r *PostgresCommerceRepository) GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*domainCommerce.Invoice, error) {
	row, err := r.queries.GetInvoiceByOrderID(ctx, toPgUUID(orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainCommerce.ErrInvoiceNotFound
		}
		return nil, err
	}
	inv := toDomainInvoice(row)
	items, err := r.queries.ListInvoiceItemsByInvoiceID(ctx, row.ID)
	if err == nil {
		for _, item := range items {
			inv.Items = append(inv.Items, domainCommerce.InvoiceItem{
				ID:              fromPgUUID(item.ID),
				InvoiceID:       fromPgUUID(item.InvoiceID),
				DescriptionI18n: item.DescriptionI18n,
				Quantity:        int(item.Quantity),
				UnitAmountMinor: item.UnitAmountMinor,
				TotalMinor:      item.TotalMinor,
				CreatedAt:       item.CreatedAt.Time,
			})
		}
	}
	return inv, nil
}

func (r *PostgresCommerceRepository) ListInvoicesByUserID(ctx context.Context, userID uuid.UUID) ([]*domainCommerce.Invoice, error) {
	rows, err := r.queries.ListInvoicesByUserID(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Invoice, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainInvoice(row))
	}
	return result, nil
}

func (r *PostgresCommerceRepository) ListAllInvoices(ctx context.Context) ([]*domainCommerce.Invoice, error) {
	rows, err := r.queries.ListAllInvoices(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.Invoice, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainInvoice(row))
	}
	return result, nil
}

// --- Wallet & Ledger Methods ---

func toDomainWallet(w Wallets) *domainCommerce.Wallet {
	return &domainCommerce.Wallet{
		ID:                    fromPgUUID(w.ID),
		UserID:                fromPgUUID(w.UserID),
		Currency:              w.Currency,
		AvailableBalanceMinor: w.AvailableBalanceMinor,
		CreatedAt:             w.CreatedAt.Time,
		UpdatedAt:             w.UpdatedAt.Time,
	}
}

func (r *PostgresCommerceRepository) GetOrCreateWallet(ctx context.Context, userID uuid.UUID, currency string) (*domainCommerce.Wallet, error) {
	row, err := r.queries.GetWalletByUserID(ctx, GetWalletByUserIDParams{
		UserID:   toPgUUID(userID),
		Currency: currency,
	})
	if err == nil {
		return toDomainWallet(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create wallet
	newWalletID := uuid.New()
	created, err := r.queries.CreateWallet(ctx, CreateWalletParams{
		ID:                    toPgUUID(newWalletID),
		UserID:                toPgUUID(userID),
		Currency:              currency,
		AvailableBalanceMinor: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}
	return toDomainWallet(created), nil
}

func (r *PostgresCommerceRepository) DepositWalletTx(ctx context.Context, userID uuid.UUID, amountMinor int64, currency, description string) (*domainCommerce.Wallet, error) {
	if currency == "" {
		currency = "USD"
	}
	if description == "" {
		description = "Wallet deposit"
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Get or create wallet with row lock
	walletRow, err := qtx.GetWalletByUserIDForUpdate(ctx, GetWalletByUserIDForUpdateParams{
		UserID:   toPgUUID(userID),
		Currency: currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			newWalletID := uuid.New()
			walletRow, err = qtx.CreateWallet(ctx, CreateWalletParams{
				ID:                    toPgUUID(newWalletID),
				UserID:                toPgUUID(userID),
				Currency:              currency,
				AvailableBalanceMinor: 0,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create wallet for deposit: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to lock wallet: %w", err)
		}
	}

	// 2. Create balanced Double-Entry Ledger Transaction & Entries
	ledgerTxID := uuid.New()
	refType := "wallet_deposit"
	_, err = qtx.CreateLedgerTransaction(ctx, CreateLedgerTransactionParams{
		ID:            toPgUUID(ledgerTxID),
		Type:          "wallet_deposit",
		ReferenceType: toPgText(refType),
		ReferenceID:   walletRow.ID,
		Description:   toPgText(description),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ledger transaction: %w", err)
	}

	// Debit: Gateway Account
	_, err = qtx.CreateLedgerEntry(ctx, CreateLedgerEntryParams{
		ID:            toPgUUID(uuid.New()),
		TransactionID: toPgUUID(ledgerTxID),
		AccountType:   string(domainCommerce.AccountPaymentGateway),
		AccountID:     toPgUUID(domainCommerce.SystemAccountPaymentGateway),
		Direction:     string(domainCommerce.DirectionDebit),
		AmountMinor:   amountMinor,
		Currency:      currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create debit ledger entry: %w", err)
	}

	// Credit: User Wallet Account
	_, err = qtx.CreateLedgerEntry(ctx, CreateLedgerEntryParams{
		ID:            toPgUUID(uuid.New()),
		TransactionID: toPgUUID(ledgerTxID),
		AccountType:   string(domainCommerce.AccountUserWallet),
		AccountID:     toPgUUID(userID),
		Direction:     string(domainCommerce.DirectionCredit),
		AmountMinor:   amountMinor,
		Currency:      currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credit ledger entry: %w", err)
	}

	// 3. Update wallet balance
	newBalance := walletRow.AvailableBalanceMinor + amountMinor
	updatedWallet, err := qtx.UpdateWalletBalance(ctx, UpdateWalletBalanceParams{
		UserID:                walletRow.UserID,
		Currency:              walletRow.Currency,
		AvailableBalanceMinor: newBalance,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// 4. Create outbox event
	outboxPayload, _ := json.Marshal(map[string]any{
		"user_id":      userID.String(),
		"amount_minor": amountMinor,
		"currency":     currency,
		"new_balance":  newBalance,
		"deposited_at": time.Now().UTC().Format(time.RFC3339),
	})
	_, _ = qtx.CreateOutboxEvent(ctx, CreateOutboxEventParams{
		ID:            toPgUUID(uuid.New()),
		EventType:     "wallet.deposited.v1",
		AggregateType: "wallet",
		AggregateID:   walletRow.ID,
		Payload:       outboxPayload,
		Status:        "pending",
		Attempts:      0,
		NextAttemptAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit deposit tx: %w", err)
	}

	return toDomainWallet(updatedWallet), nil
}

func (r *PostgresCommerceRepository) ListLedgerTransactions(ctx context.Context, limit, offset int) ([]*domainCommerce.LedgerTransaction, error) {
	rows, err := r.queries.ListRecentLedgerTransactions(ctx, ListRecentLedgerTransactionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.LedgerTransaction, 0, len(rows))
	for _, row := range rows {
		txObj := &domainCommerce.LedgerTransaction{
			ID:            fromPgUUID(row.ID),
			Type:          row.Type,
			ReferenceType: fromPgText(row.ReferenceType),
			ReferenceID:   fromPgUUIDPtr(row.ReferenceID),
			Description:   fromPgText(row.Description),
			CreatedAt:     row.CreatedAt.Time,
		}
		entries, _ := r.queries.ListLedgerEntriesByTransactionID(ctx, row.ID)
		for _, ent := range entries {
			txObj.Entries = append(txObj.Entries, domainCommerce.LedgerEntry{
				ID:            fromPgUUID(ent.ID),
				TransactionID: fromPgUUID(ent.TransactionID),
				AccountType:   domainCommerce.AccountType(ent.AccountType),
				AccountID:     fromPgUUID(ent.AccountID),
				Direction:     domainCommerce.LedgerDirection(ent.Direction),
				AmountMinor:   ent.AmountMinor,
				Currency:      ent.Currency,
				CreatedAt:     ent.CreatedAt.Time,
			})
		}
		result = append(result, txObj)
	}
	return result, nil
}

func (r *PostgresCommerceRepository) ListUserLedgerEntries(ctx context.Context, userID uuid.UUID) ([]*domainCommerce.LedgerEntry, error) {
	rows, err := r.queries.ListLedgerEntriesByAccount(ctx, ListLedgerEntriesByAccountParams{
		AccountType: string(domainCommerce.AccountUserWallet),
		AccountID:   toPgUUID(userID),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domainCommerce.LedgerEntry, 0, len(rows))
	for _, ent := range rows {
		result = append(result, &domainCommerce.LedgerEntry{
			ID:            fromPgUUID(ent.ID),
			TransactionID: fromPgUUID(ent.TransactionID),
			AccountType:   domainCommerce.AccountType(ent.AccountType),
			AccountID:     fromPgUUID(ent.AccountID),
			Direction:     domainCommerce.LedgerDirection(ent.Direction),
			AmountMinor:   ent.AmountMinor,
			Currency:      ent.Currency,
			CreatedAt:     ent.CreatedAt.Time,
		})
	}
	return result, nil
}
