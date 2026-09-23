package commerce

import (
	"context"
	"errors"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type WalletService struct {
	walletRepo  domainCommerce.WalletRepository
	invoiceRepo domainCommerce.InvoiceRepository
}

func NewWalletService(
	walletRepo domainCommerce.WalletRepository,
	invoiceRepo domainCommerce.InvoiceRepository,
) *WalletService {
	return &WalletService{
		walletRepo:  walletRepo,
		invoiceRepo: invoiceRepo,
	}
}

func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, amountMinor int64, currency string) (*domainCommerce.Wallet, error) {
	if amountMinor <= 0 {
		return nil, errors.New("deposit amount must be greater than zero")
	}
	if currency == "" {
		currency = "USD"
	}
	return s.walletRepo.DepositWalletTx(ctx, userID, amountMinor, currency, "Wallet deposit via online payment")
}

func (s *WalletService) GetUserWallet(ctx context.Context, userID uuid.UUID, currency string) (*domainCommerce.Wallet, error) {

	if currency == "" {
		currency = "USD"
	}
	return s.walletRepo.GetOrCreateWallet(ctx, userID, currency)
}

func (s *WalletService) ListUserLedgerEntries(ctx context.Context, userID uuid.UUID) ([]*domainCommerce.LedgerEntry, error) {
	return s.walletRepo.ListUserLedgerEntries(ctx, userID)
}

func (s *WalletService) ListLedgerTransactions(ctx context.Context, limit, offset int) ([]*domainCommerce.LedgerTransaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.walletRepo.ListLedgerTransactions(ctx, limit, offset)
}

func (s *WalletService) GetInvoiceByID(ctx context.Context, userID uuid.UUID, invoiceID uuid.UUID) (*domainCommerce.Invoice, error) {
	inv, err := s.invoiceRepo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.UserID != userID {
		return nil, domainCommerce.ErrInvoiceNotFound
	}
	return inv, nil
}

func (s *WalletService) ListUserInvoices(ctx context.Context, userID uuid.UUID) ([]*domainCommerce.Invoice, error) {
	return s.invoiceRepo.ListInvoicesByUserID(ctx, userID)
}

func (s *WalletService) ListAllInvoices(ctx context.Context) ([]*domainCommerce.Invoice, error) {
	return s.invoiceRepo.ListAllInvoices(ctx)
}
