package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"vps-billing/internal/domain/identity"
)

type PostgresUserRepository struct {
	q *Queries
}

func NewPostgresUserRepository(q *Queries) *PostgresUserRepository {
	return &PostgresUserRepository{q: q}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *identity.User) error {
	row, err := r.q.CreateUser(ctx, CreateUserParams{
		ID:           toPgUUID(user.ID),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Status:       string(user.Status),
		Locale:       user.Locale,
		Timezone:     user.Timezone,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = row.CreatedAt.Time
	user.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	row, err := r.q.GetUserByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return toDomainUser(row), nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*identity.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return toDomainUser(row), nil
}

func (r *PostgresUserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status identity.UserStatus) error {
	_, err := r.q.UpdateUserStatus(ctx, UpdateUserStatusParams{
		ID:     toPgUUID(id),
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	err := r.q.UpdateUserLastLogin(ctx, toPgUUID(id))
	if err != nil {
		return fmt.Errorf("failed to update user last login: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) ListUsers(ctx context.Context, limit int) ([]*identity.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.q.ListUsers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	users := make([]*identity.User, len(rows))
	for i, row := range rows {
		users[i] = toDomainUser(row)
	}
	return users, nil
}

func toDomainUser(row Users) *identity.User {
	return &identity.User{
		ID:              fromPgUUID(row.ID),
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Status:          identity.UserStatus(row.Status),
		Locale:          row.Locale,
		Timezone:        row.Timezone,
		EmailVerifiedAt: fromPgTimestamptz(row.EmailVerifiedAt),
		LastLoginAt:     fromPgTimestamptz(row.LastLoginAt),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}
