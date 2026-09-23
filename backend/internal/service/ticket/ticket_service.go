package ticket

import (
	"context"
	"errors"
	"strings"

	domainTicket "vps-billing/internal/domain/ticket"
	"vps-billing/internal/repository"

	"github.com/google/uuid"
)

type Service struct {
	repo *repository.TicketRepository
}

func NewService(repo *repository.TicketRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTicket(ctx context.Context, userID uuid.UUID, subject string, priority domainTicket.TicketPriority, message string) (*domainTicket.Ticket, error) {
	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)
	if subject == "" || message == "" {
		return nil, errors.New("subject and message are required")
	}

	t := &domainTicket.Ticket{
		UserID:   userID,
		Subject:  subject,
		Priority: priority,
		Status:   domainTicket.StatusOpen,
	}

	if err := s.repo.Create(ctx, t, message); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) ListUserTickets(ctx context.Context, userID uuid.UUID) ([]*domainTicket.Ticket, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) ListAllTickets(ctx context.Context, limit, offset int) ([]*domainTicket.Ticket, error) {
	return s.repo.ListAll(ctx, limit, offset)
}

func (s *Service) GetTicket(ctx context.Context, id uuid.UUID) (*domainTicket.Ticket, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UserReply(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, message string) (*domainTicket.Message, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, errors.New("message cannot be empty")
	}
	t, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t.UserID != userID {
		return nil, errors.New("unauthorized to reply to this ticket")
	}
	return s.repo.AddMessage(ctx, ticketID, "user", userID, message)
}

func (s *Service) AdminReply(ctx context.Context, ticketID uuid.UUID, adminID uuid.UUID, message string) (*domainTicket.Message, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, errors.New("message cannot be empty")
	}
	return s.repo.AddMessage(ctx, ticketID, "admin", adminID, message)
}

func (s *Service) UpdateStatus(ctx context.Context, ticketID uuid.UUID, status domainTicket.TicketStatus) error {
	return s.repo.UpdateStatus(ctx, ticketID, status)
}
