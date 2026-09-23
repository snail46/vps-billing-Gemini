package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"vps-billing/internal/domain/identity"
)

type PostgresAdminRepository struct {
	q *Queries
}

func NewPostgresAdminRepository(q *Queries) *PostgresAdminRepository {
	return &PostgresAdminRepository{q: q}
}

func (r *PostgresAdminRepository) Create(ctx context.Context, admin *identity.Admin) error {
	row, err := r.q.CreateAdmin(ctx, CreateAdminParams{
		ID:               toPgUUID(admin.ID),
		Email:            admin.Email,
		PasswordHash:     admin.PasswordHash,
		Status:           string(admin.Status),
		DisplayName:      toPgText(admin.DisplayName),
		TwoFactorEnabled: admin.TwoFactorEnabled,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	admin.CreatedAt = row.CreatedAt.Time
	admin.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *PostgresAdminRepository) GetByID(ctx context.Context, id uuid.UUID) (*identity.Admin, error) {
	row, err := r.q.GetAdminByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identity.ErrAdminNotFound
		}
		return nil, fmt.Errorf("failed to get admin by id: %w", err)
	}

	return toDomainAdmin(row), nil
}

func (r *PostgresAdminRepository) GetByEmail(ctx context.Context, email string) (*identity.Admin, error) {
	row, err := r.q.GetAdminByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identity.ErrAdminNotFound
		}
		return nil, fmt.Errorf("failed to get admin by email: %w", err)
	}

	return toDomainAdmin(row), nil
}

func (r *PostgresAdminRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status identity.AdminStatus) error {
	_, err := r.q.UpdateAdminStatus(ctx, UpdateAdminStatusParams{
		ID:     toPgUUID(id),
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("failed to update admin status: %w", err)
	}
	return nil
}

func (r *PostgresAdminRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	err := r.q.UpdateAdminLastLogin(ctx, toPgUUID(id))
	if err != nil {
		return fmt.Errorf("failed to update admin last login: %w", err)
	}
	return nil
}

func (r *PostgresAdminRepository) GetRoles(ctx context.Context, adminID uuid.UUID) ([]string, error) {
	roles, err := r.q.GetAdminRoles(ctx, toPgUUID(adminID))
	if err != nil {
		return nil, fmt.Errorf("failed to get admin roles: %w", err)
	}

	result := make([]string, len(roles))
	for i, r := range roles {
		result[i] = r.Key
	}
	return result, nil
}

func (r *PostgresAdminRepository) GetPermissions(ctx context.Context, adminID uuid.UUID) ([]string, error) {
	perms, err := r.q.GetAdminPermissions(ctx, toPgUUID(adminID))
	if err != nil {
		return nil, fmt.Errorf("failed to get admin permissions: %w", err)
	}
	return perms, nil
}

func (r *PostgresAdminRepository) AssignRole(ctx context.Context, adminID uuid.UUID, roleID uuid.UUID) error {
	return r.q.AssignRoleToAdmin(ctx, AssignRoleToAdminParams{
		AdminID: toPgUUID(adminID),
		RoleID:  toPgUUID(roleID),
	})
}

func (r *PostgresAdminRepository) ListAdmins(ctx context.Context) ([]*identity.Admin, error) {
	rows, err := r.q.ListAdmins(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	admins := make([]*identity.Admin, len(rows))
	for i, row := range rows {
		admins[i] = toDomainAdmin(row)
	}
	return admins, nil
}

func (r *PostgresAdminRepository) UpdateTwoFactor(ctx context.Context, id uuid.UUID, enabled bool, secret *string) error {
	sec := ""
	if secret != nil {
		sec = *secret
	}
	return r.q.UpdateAdminTwoFactor(ctx, UpdateAdminTwoFactorParams{
		ID:               toPgUUID(id),
		TwoFactorEnabled: enabled,
		TwoFactorSecret:  toPgText(sec),
	})
}

func toDomainAdmin(row Admins) *identity.Admin {
	displayName := ""
	if row.DisplayName.Valid {
		displayName = row.DisplayName.String
	}

	var secret *string
	if row.TwoFactorSecret.Valid && row.TwoFactorSecret.String != "" {
		s := row.TwoFactorSecret.String
		secret = &s
	}

	return &identity.Admin{
		ID:               fromPgUUID(row.ID),
		Email:            row.Email,
		PasswordHash:     row.PasswordHash,
		Status:           identity.AdminStatus(row.Status),
		DisplayName:      displayName,
		TwoFactorEnabled: row.TwoFactorEnabled,
		TwoFactorSecret:  secret,
		LastLoginAt:      fromPgTimestamptz(row.LastLoginAt),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}
