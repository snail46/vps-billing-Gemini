package identity_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	domainAudit "vps-billing/internal/domain/audit"
	domainIdentity "vps-billing/internal/domain/identity"
	auditService "vps-billing/internal/service/audit"
	serviceIdentity "vps-billing/internal/service/identity"
	"vps-billing/internal/service/session"
)

// In-memory mock repositories for testing
type mockUserRepo struct {
	mu    sync.RWMutex
	users map[string]*domainIdentity.User
	byID  map[uuid.UUID]*domainIdentity.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*domainIdentity.User),
		byID:  make(map[uuid.UUID]*domainIdentity.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, u *domainIdentity.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domainIdentity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, domainIdentity.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*domainIdentity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, domainIdentity.ErrUserNotFound
}

func (m *mockUserRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domainIdentity.UserStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[id]; ok {
		u.Status = status
		return nil
	}
	return domainIdentity.ErrUserNotFound
}

func (m *mockUserRepo) UpdateLastLogin(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[id]; ok {
		now := time.Now().UTC()
		u.LastLoginAt = &now
		return nil
	}
	return domainIdentity.ErrUserNotFound
}

func (m *mockUserRepo) ListUsers(_ context.Context, limit int) ([]*domainIdentity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*domainIdentity.User, 0, len(m.byID))
	for _, u := range m.byID {
		res = append(res, u)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

type mockAdminRepo struct {
	mu     sync.RWMutex
	admins map[string]*domainIdentity.Admin
	byID   map[uuid.UUID]*domainIdentity.Admin
	roles  map[uuid.UUID][]string
	perms  map[uuid.UUID][]string
}

func newMockAdminRepo() *mockAdminRepo {
	return &mockAdminRepo{
		admins: make(map[string]*domainIdentity.Admin),
		byID:   make(map[uuid.UUID]*domainIdentity.Admin),
		roles:  make(map[uuid.UUID][]string),
		perms:  make(map[uuid.UUID][]string),
	}
}

func (m *mockAdminRepo) Create(_ context.Context, a *domainIdentity.Admin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.admins[a.Email] = a
	m.byID[a.ID] = a
	return nil
}

func (m *mockAdminRepo) GetByID(_ context.Context, id uuid.UUID) (*domainIdentity.Admin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if a, ok := m.byID[id]; ok {
		return a, nil
	}
	return nil, domainIdentity.ErrAdminNotFound
}

func (m *mockAdminRepo) GetByEmail(_ context.Context, email string) (*domainIdentity.Admin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if a, ok := m.admins[email]; ok {
		return a, nil
	}
	return nil, domainIdentity.ErrAdminNotFound
}

func (m *mockAdminRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domainIdentity.AdminStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.byID[id]; ok {
		a.Status = status
		return nil
	}
	return domainIdentity.ErrAdminNotFound
}

func (m *mockAdminRepo) UpdateLastLogin(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.byID[id]; ok {
		now := time.Now().UTC()
		a.LastLoginAt = &now
		return nil
	}
	return domainIdentity.ErrAdminNotFound
}

func (m *mockAdminRepo) GetRoles(_ context.Context, adminID uuid.UUID) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.roles[adminID], nil
}

func (m *mockAdminRepo) GetPermissions(_ context.Context, adminID uuid.UUID) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.perms[adminID], nil
}

func (m *mockAdminRepo) AssignRole(_ context.Context, adminID uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockAdminRepo) ListAdmins(_ context.Context) ([]*domainIdentity.Admin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*domainIdentity.Admin, 0, len(m.byID))
	for _, a := range m.byID {
		res = append(res, a)
	}
	return res, nil
}

func (m *mockAdminRepo) UpdateTwoFactor(_ context.Context, id uuid.UUID, enabled bool, secret *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.byID[id]; ok {
		a.TwoFactorEnabled = enabled
		a.TwoFactorSecret = secret
		return nil
	}
	return domainIdentity.ErrAdminNotFound
}

type mockAuditRepo struct {
	mu     sync.RWMutex
	events []domainAudit.AuditEvent
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{
		events: make([]domainAudit.AuditEvent, 0),
	}
}

func (m *mockAuditRepo) Create(_ context.Context, e *domainAudit.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, *e)
	return nil
}

func (m *mockAuditRepo) List(_ context.Context, limit, offset int32) ([]domainAudit.AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	start := int(offset)
	if start >= len(m.events) {
		return []domainAudit.AuditEvent{}, nil
	}
	end := start + int(limit)
	if end > len(m.events) {
		end = len(m.events)
	}
	return m.events[start:end], nil
}

type mockRBACRepo struct {
	mu    sync.RWMutex
	roles map[string]*domainIdentity.Role
}

func newMockRBACRepo() *mockRBACRepo {
	return &mockRBACRepo{
		roles: make(map[string]*domainIdentity.Role),
	}
}

func (m *mockRBACRepo) SeedInitialRolesAndPermissions(_ context.Context) error {
	return nil
}

func (m *mockRBACRepo) GetRoleByKey(_ context.Context, key string) (*domainIdentity.Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.roles[key]; ok {
		return r, nil
	}
	return &domainIdentity.Role{ID: uuid.New(), Key: key, NameKey: "role." + key}, nil
}

func TestUserRegistrationAndDuplicate(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	auditRepo := newMockAuditRepo()
	auditSvc := auditService.NewService(auditRepo)
	sessMgr := session.NewManager(session.NewMemorySessionStore(), false)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)

	// 1. Success registration
	user, err := userSvc.Register(ctx, "test@example.com", "Password123!", "zh-CN", "Asia/Shanghai", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected registration success, got: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
	if user.Status != domainIdentity.UserStatusActive {
		t.Errorf("expected status active, got %s", user.Status)
	}

	// 2. Duplicate registration fails
	_, err = userSvc.Register(ctx, "test@example.com", "AnotherPassword123!", "en-US", "UTC", "127.0.0.1", "test-agent")
	if !errors.Is(err, domainIdentity.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got: %v", err)
	}

	// 3. Audit Event recorded
	events, err := auditSvc.List(ctx, 10, 0)
	if err != nil || len(events) == 0 {
		t.Fatalf("expected audit events to be recorded")
	}
	if events[0].Action != "user.registered" {
		t.Errorf("expected audit action user.registered, got %s", events[0].Action)
	}
}

func TestUserLoginStatuses(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	auditRepo := newMockAuditRepo()
	auditSvc := auditService.NewService(auditRepo)
	sessMgr := session.NewManager(session.NewMemorySessionStore(), false)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)

	_, err := userSvc.Register(ctx, "user@example.com", "StrongPassword123!", "zh-CN", "UTC", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// 1. Valid login
	u, sess, err := userSvc.Login(ctx, "user@example.com", "StrongPassword123!", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected login success, got: %v", err)
	}
	if sess.Token == "" || sess.CSRFToken == "" {
		t.Errorf("expected session token and csrf token")
	}

	// 2. Invalid password
	_, _, err = userSvc.Login(ctx, "user@example.com", "WrongPassword123!", "127.0.0.1", "test-agent")
	if !errors.Is(err, domainIdentity.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got: %v", err)
	}

	// 3. Suspended user
	_ = userRepo.UpdateStatus(ctx, u.ID, domainIdentity.UserStatusSuspended)
	_, _, err = userSvc.Login(ctx, "user@example.com", "StrongPassword123!", "127.0.0.1", "test-agent")
	if !errors.Is(err, domainIdentity.ErrAccountSuspended) {
		t.Errorf("expected ErrAccountSuspended, got: %v", err)
	}

	// 4. Disabled user
	_ = userRepo.UpdateStatus(ctx, u.ID, domainIdentity.UserStatusDisabled)
	_, _, err = userSvc.Login(ctx, "user@example.com", "StrongPassword123!", "127.0.0.1", "test-agent")
	if !errors.Is(err, domainIdentity.ErrAccountDisabled) {
		t.Errorf("expected ErrAccountDisabled, got: %v", err)
	}
}

func TestAdminLoginAndRBAC(t *testing.T) {
	ctx := context.Background()
	adminRepo := newMockAdminRepo()
	rbacRepo := newMockRBACRepo()
	auditRepo := newMockAuditRepo()
	auditSvc := auditService.NewService(auditRepo)
	sessMgr := session.NewManager(session.NewMemorySessionStore(), false)
	adminSvc := serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

	// Create super admin
	admin, err := adminSvc.CreateInitialAdmin(ctx, "super@admin.com", "AdminPass123!", "Super Admin", domainIdentity.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("failed to create initial admin: %v", err)
	}
	adminRepo.roles[admin.ID] = []string{domainIdentity.RoleSuperAdmin}

	// Login super admin
	_, sess, err := adminSvc.Login(ctx, "super@admin.com", "AdminPass123!", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected admin login success, got: %v", err)
	}
	if sess.ActorType != "admin" {
		t.Errorf("expected session ActorType admin, got %s", sess.ActorType)
	}

	// Check Super Admin permission
	hasPerm, err := adminSvc.HasPermission(ctx, admin.ID, "any.arbitrary.permission")
	if err != nil || !hasPerm {
		t.Errorf("expected super_admin to have all permissions, got %v, err %v", hasPerm, err)
	}

	// Create operations admin
	opsAdmin, err := adminSvc.CreateInitialAdmin(ctx, "ops@admin.com", "OpsPass123!", "Ops Admin", domainIdentity.RoleOperations)
	if err != nil {
		t.Fatalf("failed to create ops admin: %v", err)
	}
	adminRepo.roles[opsAdmin.ID] = []string{domainIdentity.RoleOperations}
	adminRepo.perms[opsAdmin.ID] = []string{"nodes.read", "instances.restart"}

	// Ops admin has instances.restart
	hasRestart, err := adminSvc.HasPermission(ctx, opsAdmin.ID, "instances.restart")
	if err != nil || !hasRestart {
		t.Errorf("expected ops admin to have instances.restart")
	}

	// Ops admin lacks payments.refund
	hasRefund, err := adminSvc.HasPermission(ctx, opsAdmin.ID, "payments.refund")
	if err != nil || hasRefund {
		t.Errorf("expected ops admin to NOT have payments.refund, got %v", hasRefund)
	}
}

func TestAdminTwoFactorAuthentication(t *testing.T) {
	ctx := context.Background()
	adminRepo := newMockAdminRepo()
	rbacRepo := newMockRBACRepo()
	auditRepo := newMockAuditRepo()
	auditSvc := auditService.NewService(auditRepo)
	sessMgr := session.NewManager(session.NewMemorySessionStore(), false)
	adminSvc := serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

	admin, err := adminSvc.CreateInitialAdmin(ctx, "2fa@admin.com", "AdminPass123!", "2FA Admin", domainIdentity.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// 1. Setup 2FA
	secret, uri, err := adminSvc.SetupTwoFactor(ctx, admin.ID)
	if err != nil {
		t.Fatalf("failed to setup 2FA: %v", err)
	}
	if secret == "" || uri == "" {
		t.Fatalf("expected non-empty secret and uri")
	}

	// 2. Enable with invalid code should fail
	err = adminSvc.EnableTwoFactor(ctx, admin.ID, secret, "000000")
	if err != domainIdentity.ErrInvalidTwoFactorCode {
		t.Fatalf("expected ErrInvalidTwoFactorCode, got %v", err)
	}

	// 3. Enable with valid code
	validCode, err := domainIdentity.GenerateTOTPCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to generate totp: %v", err)
	}
	err = adminSvc.EnableTwoFactor(ctx, admin.ID, secret, validCode)
	if err != nil {
		t.Fatalf("failed to enable 2FA: %v", err)
	}

	// 4. Standard Login without 2FA must return ErrTwoFactorRequired
	_, _, err = adminSvc.Login(ctx, "2fa@admin.com", "AdminPass123!", "127.0.0.1", "test-agent")
	if err != domainIdentity.ErrTwoFactorRequired {
		t.Fatalf("expected ErrTwoFactorRequired, got %v", err)
	}

	// 5. LoginWith2FA with bad code must fail
	_, _, err = adminSvc.LoginWith2FA(ctx, "2fa@admin.com", "AdminPass123!", "999999", "127.0.0.1", "test-agent")
	if err != domainIdentity.ErrInvalidTwoFactorCode {
		t.Fatalf("expected ErrInvalidTwoFactorCode, got %v", err)
	}

	// 6. LoginWith2FA with valid code must succeed
	currentCode, _ := domainIdentity.GenerateTOTPCode(secret, time.Now().UTC())
	loggedAdmin, sess, err := adminSvc.LoginWith2FA(ctx, "2fa@admin.com", "AdminPass123!", currentCode, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected login success with valid 2FA, got %v", err)
	}
	if loggedAdmin == nil || sess == nil {
		t.Fatalf("expected non-nil admin and session")
	}

	// 7. Disable 2FA
	err = adminSvc.DisableTwoFactor(ctx, admin.ID, currentCode)
	if err != nil {
		t.Fatalf("failed to disable 2FA: %v", err)
	}

	// 8. Standard Login now succeeds without 2FA
	_, _, err = adminSvc.Login(ctx, "2fa@admin.com", "AdminPass123!", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected login success after disabling 2FA, got %v", err)
	}
}
