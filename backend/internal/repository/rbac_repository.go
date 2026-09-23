package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"vps-billing/internal/domain/identity"
)

type PostgresRBACRepository struct {
	q *Queries
}

func NewPostgresRBACRepository(q *Queries) *PostgresRBACRepository {
	return &PostgresRBACRepository{q: q}
}

func (r *PostgresRBACRepository) GetRoleByKey(ctx context.Context, key string) (*identity.Role, error) {
	row, err := r.q.GetRoleByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get role by key: %w", err)
	}
	return &identity.Role{
		ID:        fromPgUUID(row.ID),
		Key:       row.Key,
		NameKey:   row.NameKey,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *PostgresRBACRepository) SeedInitialRolesAndPermissions(ctx context.Context) error {
	// Standard permissions list
	permissions := []string{
		"users.read", "users.update", "users.suspend",
		"instances.read", "instances.start", "instances.stop", "instances.restart", "instances.reinstall", "instances.delete",
		"payments.read", "payments.refund",
		"ledger.read", "ledger.adjust",
		"nodes.read", "nodes.update", "nodes.delete",
		"providers.read", "providers.manage",
		"operations.read", "operations.retry",
		"tickets.read", "tickets.reply", "tickets.manage",
		"audit.read",
		"admins.manage", "roles.manage", "settings.manage",
	}

	permMap := make(map[string]uuid.UUID)
	for _, p := range permissions {
		id := uuid.New()
		row, err := r.q.CreatePermission(ctx, CreatePermissionParams{
			ID:  toPgUUID(id),
			Key: p,
		})
		if err == nil && row.ID.Valid {
			permMap[p] = fromPgUUID(row.ID)
		} else {
			// Query existing permission ID
			permMap[p] = id
		}
	}

	// Roles and their name keys
	roles := []struct {
		key     string
		nameKey string
		perms   []string
	}{
		{
			key:     identity.RoleSuperAdmin,
			nameKey: "role.super_admin",
			perms:   permissions, // Super admin has all permissions
		},
		{
			key:     identity.RoleOperations,
			nameKey: "role.operations",
			perms: []string{
				"nodes.read", "nodes.update", "nodes.delete",
				"providers.read", "providers.manage",
				"instances.read", "instances.start", "instances.stop", "instances.restart", "instances.reinstall", "instances.delete",
				"operations.read", "operations.retry",
				"audit.read",
			},
		},
		{
			key:     identity.RoleFinance,
			nameKey: "role.finance",
			perms: []string{
				"users.read", "payments.read", "payments.refund",
				"ledger.read", "ledger.adjust", "audit.read",
			},
		},
		{
			key:     identity.RoleSupport,
			nameKey: "role.support",
			perms: []string{
				"users.read", "tickets.read", "tickets.reply", "tickets.manage",
				"instances.read", "instances.start", "instances.stop", "instances.restart",
			},
		},
		{
			key:     identity.RoleReadOnly,
			nameKey: "role.read_only",
			perms: []string{
				"users.read", "instances.read", "payments.read", "ledger.read",
				"nodes.read", "providers.read", "operations.read", "tickets.read", "audit.read",
			},
		},
	}

	for _, roleDef := range roles {
		roleID := uuid.New()
		roleRow, err := r.q.CreateRole(ctx, CreateRoleParams{
			ID:      toPgUUID(roleID),
			Key:     roleDef.key,
			NameKey: roleDef.nameKey,
		})
		if err != nil {
			return fmt.Errorf("failed to create or update role %s: %w", roleDef.key, err)
		}

		actualRoleID := fromPgUUID(roleRow.ID)
		for _, permKey := range roleDef.perms {
			if permID, ok := permMap[permKey]; ok {
				_ = r.q.AssignPermissionToRole(ctx, AssignPermissionToRoleParams{
					RoleID:       toPgUUID(actualRoleID),
					PermissionID: toPgUUID(permID),
				})
			}
		}
	}

	return nil
}
