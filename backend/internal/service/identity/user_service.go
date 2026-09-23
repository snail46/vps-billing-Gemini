package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	domainAudit "vps-billing/internal/domain/audit"
	domainIdentity "vps-billing/internal/domain/identity"
	auditService "vps-billing/internal/service/audit"
	"vps-billing/internal/service/session"
)

type UserService struct {
	userRepo domainIdentity.UserRepository
	auditSvc *auditService.Service
	sessMgr  *session.Manager
}

func NewUserService(
	userRepo domainIdentity.UserRepository,
	auditSvc *auditService.Service,
	sessMgr *session.Manager,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		auditSvc: auditSvc,
		sessMgr:  sessMgr,
	}
}

func (s *UserService) Register(
	ctx context.Context,
	email, password, locale, timezone, ip, userAgent string,
) (*domainIdentity.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, errors.New("invalid email address")
	}

	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters long")
	}

	if locale == "" {
		locale = "zh-CN"
	}
	if timezone == "" {
		timezone = "UTC"
	}

	// Check if email already exists
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, domainIdentity.ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userID := uuid.Must(uuid.NewV7())
	user := &domainIdentity.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Status:       domainIdentity.UserStatusActive,
		Locale:       locale,
		Timezone:     timezone,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Record Audit Event
	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeUser,
		ActorID:      &user.ID,
		Action:       "user.registered",
		ResourceType: "user",
		ResourceID:   &user.ID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})

	return user, nil
}

func (s *UserService) Login(
	ctx context.Context,
	email, password, ip, userAgent string,
) (*domainIdentity.User, *session.Session, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Log failed attempt
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeUser,
			Action:       "user.login.failed",
			ResourceType: "user",
			BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"user_not_found"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, domainIdentity.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeUser,
			ActorID:      &user.ID,
			Action:       "user.login.failed",
			ResourceType: "user",
			ResourceID:   &user.ID,
			BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"bad_password"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, domainIdentity.ErrInvalidCredentials
	}

	if err := user.CanAuthenticate(); err != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeUser,
			ActorID:      &user.ID,
			Action:       "user.login.blocked",
			ResourceType: "user",
			ResourceID:   &user.ID,
			BeforeData:   json.RawMessage(`{"status":"` + string(user.Status) + `"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, err
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	sess, err := s.sessMgr.CreateUserSession(ctx, user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeUser,
		ActorID:      &user.ID,
		Action:       "user.login.succeeded",
		ResourceType: "user",
		ResourceID:   &user.ID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})

	return user, sess, nil
}

func (s *UserService) Logout(ctx context.Context, sessionToken string, ip, userAgent string) error {
	sess, err := s.sessMgr.GetSession(ctx, sessionToken)
	if err == nil && sess != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeUser,
			ActorID:      &sess.ActorID,
			Action:       "user.logout",
			ResourceType: "user",
			ResourceID:   &sess.ActorID,
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
	}
	return s.sessMgr.DestroySession(ctx, sessionToken)
}

func (s *UserService) GetMe(ctx context.Context, userID uuid.UUID) (*domainIdentity.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := user.CanAuthenticate(); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context, limit int) ([]*domainIdentity.User, error) {
	return s.userRepo.ListUsers(ctx, limit)
}
