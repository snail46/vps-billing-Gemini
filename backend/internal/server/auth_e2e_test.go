package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"vps-billing/internal/config"
	domainAudit "vps-billing/internal/domain/audit"
	domainIdentity "vps-billing/internal/domain/identity"
	"vps-billing/internal/server"
	serviceAudit "vps-billing/internal/service/audit"
	serviceIdentity "vps-billing/internal/service/identity"
	"vps-billing/internal/service/session"
)

// In-memory test mock repos
type testUserRepo struct {
	mu    sync.RWMutex
	users map[string]*domainIdentity.User
	byID  map[uuid.UUID]*domainIdentity.User
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{
		users: make(map[string]*domainIdentity.User),
		byID:  make(map[uuid.UUID]*domainIdentity.User),
	}
}

func (r *testUserRepo) Create(_ context.Context, u *domainIdentity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.Email] = u
	r.byID[u.ID] = u
	return nil
}

func (r *testUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domainIdentity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, domainIdentity.ErrUserNotFound
}

func (r *testUserRepo) GetByEmail(_ context.Context, email string) (*domainIdentity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.users[email]; ok {
		return u, nil
	}
	return nil, domainIdentity.ErrUserNotFound
}

func (r *testUserRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domainIdentity.UserStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		u.Status = status
		return nil
	}
	return domainIdentity.ErrUserNotFound
}

func (r *testUserRepo) UpdateLastLogin(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		now := time.Now().UTC()
		u.LastLoginAt = &now
		return nil
	}
	return domainIdentity.ErrUserNotFound
}

func (r *testUserRepo) ListUsers(_ context.Context, limit int) ([]*domainIdentity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]*domainIdentity.User, 0, len(r.byID))
	for _, u := range r.byID {
		res = append(res, u)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

type testAdminRepo struct {
	mu     sync.RWMutex
	admins map[string]*domainIdentity.Admin
	byID   map[uuid.UUID]*domainIdentity.Admin
	roles  map[uuid.UUID][]string
	perms  map[uuid.UUID][]string
}

func newTestAdminRepo() *testAdminRepo {
	return &testAdminRepo{
		admins: make(map[string]*domainIdentity.Admin),
		byID:   make(map[uuid.UUID]*domainIdentity.Admin),
		roles:  make(map[uuid.UUID][]string),
		perms:  make(map[uuid.UUID][]string),
	}
}

func (r *testAdminRepo) Create(_ context.Context, a *domainIdentity.Admin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.admins[a.Email] = a
	r.byID[a.ID] = a
	return nil
}

func (r *testAdminRepo) GetByID(_ context.Context, id uuid.UUID) (*domainIdentity.Admin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.byID[id]; ok {
		return a, nil
	}
	return nil, domainIdentity.ErrAdminNotFound
}

func (r *testAdminRepo) GetByEmail(_ context.Context, email string) (*domainIdentity.Admin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.admins[email]; ok {
		return a, nil
	}
	return nil, domainIdentity.ErrAdminNotFound
}

func (r *testAdminRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domainIdentity.AdminStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.byID[id]; ok {
		a.Status = status
		return nil
	}
	return domainIdentity.ErrAdminNotFound
}

func (r *testAdminRepo) UpdateLastLogin(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.byID[id]; ok {
		now := time.Now().UTC()
		a.LastLoginAt = &now
		return nil
	}
	return domainIdentity.ErrAdminNotFound
}

func (r *testAdminRepo) GetRoles(_ context.Context, adminID uuid.UUID) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.roles[adminID], nil
}

func (r *testAdminRepo) GetPermissions(_ context.Context, adminID uuid.UUID) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.perms[adminID], nil
}

func (r *testAdminRepo) AssignRole(_ context.Context, adminID uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (r *testAdminRepo) ListAdmins(_ context.Context) ([]*domainIdentity.Admin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]*domainIdentity.Admin, 0, len(r.byID))
	for _, a := range r.byID {
		res = append(res, a)
	}
	return res, nil
}

func (r *testAdminRepo) UpdateTwoFactor(_ context.Context, id uuid.UUID, enabled bool, secret *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.byID[id]; ok {
		a.TwoFactorEnabled = enabled
		a.TwoFactorSecret = secret
		return nil
	}
	return domainIdentity.ErrAdminNotFound
}

type testAuditRepo struct {
	mu     sync.RWMutex
	events []domainAudit.AuditEvent
}

func newTestAuditRepo() *testAuditRepo {
	return &testAuditRepo{events: make([]domainAudit.AuditEvent, 0)}
}

func (r *testAuditRepo) Create(_ context.Context, e *domainAudit.AuditEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, *e)
	return nil
}

func (r *testAuditRepo) List(_ context.Context, limit, offset int32) ([]domainAudit.AuditEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.events, nil
}

type testRBACRepo struct{}

func (r *testRBACRepo) SeedInitialRolesAndPermissions(_ context.Context) error {
	return nil
}

func (r *testRBACRepo) GetRoleByKey(_ context.Context, key string) (*domainIdentity.Role, error) {
	return &domainIdentity.Role{ID: uuid.New(), Key: key, NameKey: "role." + key}, nil
}

func setupTestRouter() (http.Handler, *testUserRepo, *testAdminRepo, *testAuditRepo, *session.Manager) {
	cfg := &config.Config{
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	userRepo := newTestUserRepo()
	adminRepo := newTestAdminRepo()
	auditRepo := newTestAuditRepo()
	rbacRepo := &testRBACRepo{}

	auditSvc := serviceAudit.NewService(auditRepo)
	sessMgr := session.NewManager(session.NewMemorySessionStore(), false)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)
	adminSvc := serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:     cfg,
		UserSvc:    userSvc,
		AdminSvc:   adminSvc,
		AuditSvc:   auditSvc,
		SessionMgr: sessMgr,
	})

	return router, userRepo, adminRepo, auditRepo, sessMgr
}

func TestAuthE2E(t *testing.T) {
	router, userRepo, adminRepo, auditRepo, _ := setupTestRouter()

	// 1. Unauthenticated /api/v1/auth/me returns 401
	reqMe := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	rrMe := httptest.NewRecorder()
	router.ServeHTTP(rrMe, reqMe)
	if rrMe.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated /me, got %d", rrMe.Code)
	}

	// 2. User Registration
	regBody := `{"email":"user@test.com","password":"Password123!","locale":"zh-CN","timezone":"UTC"}`
	reqReg := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	rrReg := httptest.NewRecorder()
	router.ServeHTTP(rrReg, reqReg)
	if rrReg.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for register, got %d, body: %s", rrReg.Code, rrReg.Body.String())
	}

	// 3. Duplicate User Registration returns 409
	reqDup := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(regBody))
	reqDup.Header.Set("Content-Type", "application/json")
	rrDup := httptest.NewRecorder()
	router.ServeHTTP(rrDup, reqDup)
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate register, got %d", rrDup.Code)
	}

	// 4. Invalid User Login returns 401
	badLogin := `{"email":"user@test.com","password":"WrongPassword!"}`
	reqBadLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(badLogin))
	reqBadLogin.Header.Set("Content-Type", "application/json")
	rrBadLogin := httptest.NewRecorder()
	router.ServeHTTP(rrBadLogin, reqBadLogin)
	if rrBadLogin.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rrBadLogin.Code)
	}

	// 5. Valid User Login returns 200, cookie, and csrf_token
	validLogin := `{"email":"user@test.com","password":"Password123!"}`
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(validLogin))
	reqLogin.Header.Set("Content-Type", "application/json")
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, reqLogin)
	if rrLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid login, got %d", rrLogin.Code)
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			CsrfToken string `json:"csrf_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rrLogin.Body.Bytes(), &loginResp)
	if loginResp.Data.CsrfToken == "" {
		t.Fatalf("expected non-empty csrf_token in login response, got body: %s", rrLogin.Body.String())
	}

	cookies := rrLogin.Result().Cookies()
	var userCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.UserSessionCookieName {
			userCookie = c
			break
		}
	}
	if userCookie == nil || !userCookie.HttpOnly {
		t.Fatalf("expected HttpOnly user session cookie")
	}

	// 6. Authenticated /me returns user profile
	reqAuthMe := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqAuthMe.AddCookie(userCookie)
	rrAuthMe := httptest.NewRecorder()
	router.ServeHTTP(rrAuthMe, reqAuthMe)
	if rrAuthMe.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated /me, got %d", rrAuthMe.Code)
	}

	// 7. Suspended / Disabled User cannot use protected endpoints
	u, _ := userRepo.GetByEmail(context.Background(), "user@test.com")
	_ = userRepo.UpdateStatus(context.Background(), u.ID, domainIdentity.UserStatusSuspended)

	reqSuspended := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqSuspended.AddCookie(userCookie)
	rrSuspended := httptest.NewRecorder()
	router.ServeHTTP(rrSuspended, reqSuspended)
	if rrSuspended.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for suspended user, got %d", rrSuspended.Code)
	}
	// Restore active status
	_ = userRepo.UpdateStatus(context.Background(), u.ID, domainIdentity.UserStatusActive)

	// 8. User Session CANNOT access Admin endpoints -> 401
	reqUserToAdmin := httptest.NewRequest("GET", "/api/v1/admin/auth/me", nil)
	reqUserToAdmin.AddCookie(userCookie)
	rrUserToAdmin := httptest.NewRecorder()
	router.ServeHTTP(rrUserToAdmin, reqUserToAdmin)
	if rrUserToAdmin.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when User session tries to access Admin endpoint, got %d", rrUserToAdmin.Code)
	}

	// 9. Admin Setup and Login
	adminID := uuid.New()
	adminRepo.admins["ops@test.com"] = &domainIdentity.Admin{
		ID:           adminID,
		Email:        "ops@test.com",
		PasswordHash: u.PasswordHash, // Reuse valid hash for convenience
		Status:       domainIdentity.AdminStatusActive,
		DisplayName:  "Ops Admin",
	}
	adminRepo.byID[adminID] = adminRepo.admins["ops@test.com"]
	adminRepo.roles[adminID] = []string{domainIdentity.RoleOperations}
	adminRepo.perms[adminID] = []string{"nodes.read"} // Lacks audit.read

	adminLoginBody := `{"email":"ops@test.com","password":"Password123!"}`
	reqAdminLogin := httptest.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBufferString(adminLoginBody))
	reqAdminLogin.Header.Set("Content-Type", "application/json")
	rrAdminLogin := httptest.NewRecorder()
	router.ServeHTTP(rrAdminLogin, reqAdminLogin)
	if rrAdminLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin login, got %d", rrAdminLogin.Code)
	}

	var adminCookie *http.Cookie
	for _, c := range rrAdminLogin.Result().Cookies() {
		if c.Name == session.AdminSessionCookieName {
			adminCookie = c
			break
		}
	}
	if adminCookie == nil {
		t.Fatalf("expected admin session cookie")
	}

	// 10. Admin Session CANNOT access User endpoints -> 401
	reqAdminToUser := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqAdminToUser.AddCookie(adminCookie)
	rrAdminToUser := httptest.NewRecorder()
	router.ServeHTTP(rrAdminToUser, reqAdminToUser)
	if rrAdminToUser.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when Admin session tries to access User endpoint, got %d", rrAdminToUser.Code)
	}

	// 11. Admin without required permission accessing /api/v1/admin/audit -> 403 Forbidden
	reqAuditForbidden := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	reqAuditForbidden.AddCookie(adminCookie)
	rrAuditForbidden := httptest.NewRecorder()
	router.ServeHTTP(rrAuditForbidden, reqAuditForbidden)
	if rrAuditForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for admin without audit.read permission, got %d", rrAuditForbidden.Code)
	}

	// 12. Super Admin CAN access /api/v1/admin/audit -> 200 OK
	superID := uuid.New()
	adminRepo.admins["super@test.com"] = &domainIdentity.Admin{
		ID:           superID,
		Email:        "super@test.com",
		PasswordHash: u.PasswordHash,
		Status:       domainIdentity.AdminStatusActive,
		DisplayName:  "Super Admin",
	}
	adminRepo.byID[superID] = adminRepo.admins["super@test.com"]
	adminRepo.roles[superID] = []string{domainIdentity.RoleSuperAdmin}

	superLoginBody := `{"email":"super@test.com","password":"Password123!"}`
	reqSuperLogin := httptest.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBufferString(superLoginBody))
	reqSuperLogin.Header.Set("Content-Type", "application/json")
	rrSuperLogin := httptest.NewRecorder()
	router.ServeHTTP(rrSuperLogin, reqSuperLogin)

	var superCookie *http.Cookie
	for _, c := range rrSuperLogin.Result().Cookies() {
		if c.Name == session.AdminSessionCookieName {
			superCookie = c
			break
		}
	}

	reqAuditAllowed := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	reqAuditAllowed.AddCookie(superCookie)
	rrAuditAllowed := httptest.NewRecorder()
	router.ServeHTTP(rrAuditAllowed, reqAuditAllowed)
	if rrAuditAllowed.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for super_admin accessing /audit, got %d, body: %s", rrAuditAllowed.Code, rrAuditAllowed.Body.String())
	}

	// 13. Audit events were generated
	if len(auditRepo.events) == 0 {
		t.Fatalf("expected audit events to be generated throughout tests")
	}
}
