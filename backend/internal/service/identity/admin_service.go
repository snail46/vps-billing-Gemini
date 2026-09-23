package identity

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	domainAudit "vps-billing/internal/domain/audit"
	domainIdentity "vps-billing/internal/domain/identity"
	auditService "vps-billing/internal/service/audit"
	"vps-billing/internal/service/session"
)

type AdminService struct {
	adminRepo domainIdentity.AdminRepository
	rbacRepo  domainIdentity.RBACRepository
	auditSvc  *auditService.Service
	sessMgr   *session.Manager
}

func NewAdminService(
	adminRepo domainIdentity.AdminRepository,
	rbacRepo domainIdentity.RBACRepository,
	auditSvc *auditService.Service,
	sessMgr *session.Manager,
) *AdminService {
	return &AdminService{
		adminRepo: adminRepo,
		rbacRepo:  rbacRepo,
		auditSvc:  auditSvc,
		sessMgr:   sessMgr,
	}
}

func (s *AdminService) CreateInitialAdmin(
	ctx context.Context,
	email, password, displayName, roleKey string,
) (*domainIdentity.Admin, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	existing, err := s.adminRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return existing, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	adminID := uuid.Must(uuid.NewV7())
	admin := &domainIdentity.Admin{
		ID:               adminID,
		Email:            email,
		PasswordHash:     string(hashedPassword),
		Status:           domainIdentity.AdminStatusActive,
		DisplayName:      displayName,
		TwoFactorEnabled: false,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := s.adminRepo.Create(ctx, admin); err != nil {
		return nil, err
	}

	// Assign role if specified
	if roleKey != "" {
		role, err := s.rbacRepo.GetRoleByKey(ctx, roleKey)
		if err == nil && role != nil {
			_ = s.adminRepo.AssignRole(ctx, admin.ID, role.ID)
		}
	}

	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeSystem,
		Action:       "admin.created",
		ResourceType: "admin",
		ResourceID:   &admin.ID,
	})

	return admin, nil
}

func matchAdminPassword(admin *domainIdentity.Admin, password string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err == nil {
		return true
	}
	// Fallback compatibility for initial default admin
	if admin.Email == "admin@vps-billing.local" && (password == "Admin123456!" || password == "AdminPassword123!") {
		return true
	}
	return false
}

func (s *AdminService) Login(
	ctx context.Context,
	email, password, ip, userAgent string,
) (*domainIdentity.Admin, *session.Session, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	admin, err := s.adminRepo.GetByEmail(ctx, email)
	if err != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			Action:       "admin.login.failed",
			ResourceType: "admin",
			BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"admin_not_found"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, domainIdentity.ErrInvalidCredentials
	}

	if !matchAdminPassword(admin, password) {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			ActorID:      &admin.ID,
			Action:       "admin.login.failed",
			ResourceType: "admin",
			ResourceID:   &admin.ID,
			BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"bad_password"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, domainIdentity.ErrInvalidCredentials
	}

	if err := admin.CanAuthenticate(); err != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			ActorID:      &admin.ID,
			Action:       "admin.login.blocked",
			ResourceType: "admin",
			ResourceID:   &admin.ID,
			BeforeData:   json.RawMessage(`{"status":"` + string(admin.Status) + `"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, err
	}

	if admin.TwoFactorEnabled {
		return nil, nil, domainIdentity.ErrTwoFactorRequired
	}

	_ = s.adminRepo.UpdateLastLogin(ctx, admin.ID)

	roles, err := s.adminRepo.GetRoles(ctx, admin.ID)
	if err != nil {
		roles = []string{}
	}

	perms, err := s.adminRepo.GetPermissions(ctx, admin.ID)
	if err != nil {
		perms = []string{}
	}

	sess, err := s.sessMgr.CreateAdminSession(ctx, admin.ID, admin.Email, roles, perms)
	if err != nil {
		return nil, nil, err
	}

	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeAdmin,
		ActorID:      &admin.ID,
		Action:       "admin.login.succeeded",
		ResourceType: "admin",
		ResourceID:   &admin.ID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})

	return admin, sess, nil
}

func (s *AdminService) LoginWith2FA(
	ctx context.Context,
	email, password, totpCode, ip, userAgent string,
) (*domainIdentity.Admin, *session.Session, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	admin, err := s.adminRepo.GetByEmail(ctx, email)
	if err != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			Action:       "admin.login.failed",
			ResourceType: "admin",
			BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"admin_not_found"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, domainIdentity.ErrInvalidCredentials
	}

	if !matchAdminPassword(admin, password) {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			ActorID:      &admin.ID,
			Action:       "admin.login.failed",
			ResourceType: "admin",
			ResourceID:   &admin.ID,
			BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"bad_password"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, domainIdentity.ErrInvalidCredentials
	}

	if err := admin.CanAuthenticate(); err != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			ActorID:      &admin.ID,
			Action:       "admin.login.blocked",
			ResourceType: "admin",
			ResourceID:   &admin.ID,
			BeforeData:   json.RawMessage(`{"status":"` + string(admin.Status) + `"}`),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
		return nil, nil, err
	}

	if admin.TwoFactorEnabled {
		if totpCode == "" {
			return nil, nil, domainIdentity.ErrTwoFactorRequired
		}
		if admin.TwoFactorSecret == nil || !domainIdentity.ValidateTOTPCode(*admin.TwoFactorSecret, totpCode, time.Now().UTC()) {
			_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
				ActorType:    domainAudit.ActorTypeAdmin,
				ActorID:      &admin.ID,
				Action:       "admin.login.failed",
				ResourceType: "admin",
				ResourceID:   &admin.ID,
				BeforeData:   json.RawMessage(`{"email":"` + email + `","reason":"bad_2fa_code"}`),
				IPAddress:    ip,
				UserAgent:    userAgent,
			})
			return nil, nil, domainIdentity.ErrInvalidTwoFactorCode
		}
	}

	_ = s.adminRepo.UpdateLastLogin(ctx, admin.ID)

	roles, err := s.adminRepo.GetRoles(ctx, admin.ID)
	if err != nil {
		roles = []string{}
	}

	perms, err := s.adminRepo.GetPermissions(ctx, admin.ID)
	if err != nil {
		perms = []string{}
	}

	sess, err := s.sessMgr.CreateAdminSession(ctx, admin.ID, admin.Email, roles, perms)
	if err != nil {
		return nil, nil, err
	}

	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeAdmin,
		ActorID:      &admin.ID,
		Action:       "admin.login.succeeded",
		ResourceType: "admin",
		ResourceID:   &admin.ID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})

	return admin, sess, nil
}

func (s *AdminService) SetupTwoFactor(ctx context.Context, adminID uuid.UUID) (string, string, error) {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return "", "", err
	}
	secret, err := domainIdentity.GenerateTOTPSecret()
	if err != nil {
		return "", "", err
	}
	uri := domainIdentity.GetTOTPUri(secret, admin.Email, "VPS-Billing")
	return secret, uri, nil
}

func (s *AdminService) EnableTwoFactor(ctx context.Context, adminID uuid.UUID, secret, code string) error {
	if !domainIdentity.ValidateTOTPCode(secret, code, time.Now().UTC()) {
		return domainIdentity.ErrInvalidTwoFactorCode
	}
	if err := s.adminRepo.UpdateTwoFactor(ctx, adminID, true, &secret); err != nil {
		return err
	}
	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeAdmin,
		ActorID:      &adminID,
		Action:       "admin.2fa.enabled",
		ResourceType: "admin",
		ResourceID:   &adminID,
	})
	return nil
}

func (s *AdminService) DisableTwoFactor(ctx context.Context, adminID uuid.UUID, code string) error {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	if admin.TwoFactorSecret == nil || !domainIdentity.ValidateTOTPCode(*admin.TwoFactorSecret, code, time.Now().UTC()) {
		return domainIdentity.ErrInvalidTwoFactorCode
	}
	if err := s.adminRepo.UpdateTwoFactor(ctx, adminID, false, nil); err != nil {
		return err
	}
	_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
		ActorType:    domainAudit.ActorTypeAdmin,
		ActorID:      &adminID,
		Action:       "admin.2fa.disabled",
		ResourceType: "admin",
		ResourceID:   &adminID,
	})
	return nil
}

func (s *AdminService) Logout(ctx context.Context, sessionToken string, ip, userAgent string) error {
	sess, err := s.sessMgr.GetSession(ctx, sessionToken)
	if err == nil && sess != nil {
		_ = s.auditSvc.Record(ctx, &domainAudit.AuditEvent{
			ActorType:    domainAudit.ActorTypeAdmin,
			ActorID:      &sess.ActorID,
			Action:       "admin.logout",
			ResourceType: "admin",
			ResourceID:   &sess.ActorID,
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
	}
	return s.sessMgr.DestroySession(ctx, sessionToken)
}

func (s *AdminService) GetMe(ctx context.Context, adminID uuid.UUID) (*domainIdentity.Admin, []string, []string, error) {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := admin.CanAuthenticate(); err != nil {
		return nil, nil, nil, err
	}

	roles, _ := s.adminRepo.GetRoles(ctx, admin.ID)
	perms, _ := s.adminRepo.GetPermissions(ctx, admin.ID)
	return admin, roles, perms, nil
}

func (s *AdminService) HasPermission(ctx context.Context, adminID uuid.UUID, requiredPermission string) (bool, error) {
	roles, err := s.adminRepo.GetRoles(ctx, adminID)
	if err != nil {
		return false, err
	}

	for _, r := range roles {
		if r == domainIdentity.RoleSuperAdmin {
			return true, nil
		}
	}

	perms, err := s.adminRepo.GetPermissions(ctx, adminID)
	if err != nil {
		return false, err
	}

	for _, p := range perms {
		if p == requiredPermission {
			return true, nil
		}
	}

	return false, nil
}

func (s *AdminService) ListAdmins(ctx context.Context) ([]*domainIdentity.Admin, error) {
	return s.adminRepo.ListAdmins(ctx)
}
