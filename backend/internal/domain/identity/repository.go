package identity

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, limit int) ([]*User, error)
}

type AdminRepository interface {
	Create(ctx context.Context, admin *Admin) error
	GetByID(ctx context.Context, id uuid.UUID) (*Admin, error)
	GetByEmail(ctx context.Context, email string) (*Admin, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status AdminStatus) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	GetRoles(ctx context.Context, adminID uuid.UUID) ([]string, error)
	GetPermissions(ctx context.Context, adminID uuid.UUID) ([]string, error)
	AssignRole(ctx context.Context, adminID uuid.UUID, roleID uuid.UUID) error
	ListAdmins(ctx context.Context) ([]*Admin, error)
	UpdateTwoFactor(ctx context.Context, id uuid.UUID, enabled bool, secret *string) error
}

type RBACRepository interface {
	SeedInitialRolesAndPermissions(ctx context.Context) error
	GetRoleByKey(ctx context.Context, key string) (*Role, error)
}
