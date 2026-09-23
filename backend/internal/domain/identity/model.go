package identity

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDisabled  UserStatus = "disabled"
)

type AdminStatus string

const (
	AdminStatusActive    AdminStatus = "active"
	AdminStatusSuspended AdminStatus = "suspended"
	AdminStatusDisabled  AdminStatus = "disabled"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Status          UserStatus `json:"status"`
	Locale          string     `json:"locale"`
	Timezone        string     `json:"timezone"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (u *User) CanAuthenticate() error {
	switch u.Status {
	case UserStatusSuspended:
		return ErrAccountSuspended
	case UserStatusDisabled:
		return ErrAccountDisabled
	case UserStatusActive:
		return nil
	default:
		return ErrAccountDisabled
	}
}

type Admin struct {
	ID               uuid.UUID   `json:"id"`
	Email            string      `json:"email"`
	PasswordHash     string      `json:"-"`
	Status           AdminStatus `json:"status"`
	DisplayName      string      `json:"display_name"`
	TwoFactorEnabled bool        `json:"two_factor_enabled"`
	TwoFactorSecret  *string     `json:"-"`
	LastLoginAt      *time.Time  `json:"last_login_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *Admin) CanAuthenticate() error {
	switch a.Status {
	case AdminStatusSuspended:
		return ErrAccountSuspended
	case AdminStatusDisabled:
		return ErrAccountDisabled
	case AdminStatusActive:
		return nil
	default:
		return ErrAccountDisabled
	}
}

type Role struct {
	ID        uuid.UUID `json:"id"`
	Key       string    `json:"key"`
	NameKey   string    `json:"name_key"`
	CreatedAt time.Time `json:"created_at"`
}

type Permission struct {
	ID  uuid.UUID `json:"id"`
	Key string    `json:"key"`
}

// System Roles
const (
	RoleSuperAdmin = "super_admin"
	RoleOperations = "operations"
	RoleFinance    = "finance"
	RoleSupport    = "support"
	RoleReadOnly   = "read_only"
)

// System Permissions (Sample baseline for Phase 1)
const (
	PermUsersRead      = "users.read"
	PermUsersUpdate    = "users.update"
	PermUsersSuspend   = "users.suspend"
	PermInstancesRead  = "instances.read"
	PermPaymentsRead   = "payments.read"
	PermLedgerRead     = "ledger.read"
	PermNodesRead      = "nodes.read"
	PermProvidersRead  = "providers.read"
	PermOperationsRead = "operations.read"
	PermTicketsRead    = "tickets.read"
	PermAuditRead      = "audit.read"
	PermAdminsManage   = "admins.manage"
	PermRolesManage    = "roles.manage"
	PermSettingsManage = "settings.manage"
)
