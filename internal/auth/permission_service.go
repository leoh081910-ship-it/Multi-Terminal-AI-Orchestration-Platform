package auth

import (
	"context"
	"fmt"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/permission"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/role"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/user"
)

// PermissionService handles permission checks and role assignments
type PermissionService struct {
	client *ent.Client
}

// NewPermissionService creates a new permission service
func NewPermissionService(client *ent.Client) *PermissionService {
	return &PermissionService{client: client}
}

// GetUserPermissions returns all permission names for a user via their roles
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	// Query user with roles and their permissions
	u, err := s.client.User.Query().
		Where(user.IDEQ(userID)).
		WithRoles(func(q *ent.RoleQuery) {
			q.WithPermissions()
		}).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	permSet := make(map[string]bool)
	for _, r := range u.Edges.Roles {
		for _, p := range r.Edges.Permissions {
			permSet[p.Name] = true
		}
	}

	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	return perms, nil
}

// HasPermission checks if a user has a specific permission
func (s *PermissionService) HasPermission(ctx context.Context, userID, permName string) (bool, error) {
	// Direct query: count roles that have this permission
	count, err := s.client.User.Query().
		Where(user.IDEQ(userID)).
		QueryRoles().
		Where(role.HasPermissionsWith(permission.NameEQ(permName))).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return count > 0, nil
}

// AssignRoleToUser assigns a role to a user
func (s *PermissionService) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	return s.client.User.UpdateOneID(userID).
		AddRoleIDs(roleID).
		Exec(ctx)
}

// RemoveRoleFromUser removes a role from a user
func (s *PermissionService) RemoveRoleFromUser(ctx context.Context, userID, roleID string) error {
	return s.client.User.UpdateOneID(userID).
		RemoveRoleIDs(roleID).
		Exec(ctx)
}

// GetUserRoles returns all roles for a user
func (s *PermissionService) GetUserRoles(ctx context.Context, userID string) ([]*ent.Role, error) {
	return s.client.User.Query().
		Where(user.IDEQ(userID)).
		QueryRoles().
		All(ctx)
}

// SeedDefaultPermissions creates standard permissions if they don't exist
func (s *PermissionService) SeedDefaultPermissions(ctx context.Context) error {
	defaultPermissions := []struct {
		name     string
		resource string
		action   string
		desc     string
	}{
		{"*", "*", "*", "Super admin - all permissions"},
		{"tasks:read", "tasks", "read", "Read task information"},
		{"tasks:write", "tasks", "write", "Create and update tasks"},
		{"tasks:delete", "tasks", "delete", "Delete tasks"},
		{"projects:read", "projects", "read", "Read project information"},
		{"projects:write", "projects", "write", "Create and update projects"},
		{"orgs:read", "orgs", "read", "Read organization information"},
		{"orgs:write", "orgs", "write", "Manage organizations"},
	}

	for _, p := range defaultPermissions {
		exists, err := s.client.Permission.Query().
			Where(permission.NameEQ(p.name)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			_, err := s.client.Permission.Create().
				SetName(p.name).
				SetResource(p.resource).
				SetAction(p.action).
				SetDescription(p.desc).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("failed to seed permission %s: %w", p.name, err)
			}
		}
	}
	return nil
}

// SeedDefaultRoles creates admin, developer, viewer roles with permissions
func (s *PermissionService) SeedDefaultRoles(ctx context.Context) error {
	// First ensure permissions exist
	if err := s.SeedDefaultPermissions(ctx); err != nil {
		return err
	}

	// Get permission IDs
	allPerm, _ := s.client.Permission.Query().All(ctx)
	tasksRead, _ := s.client.Permission.Query().Where(permission.NameEQ("tasks:read")).Only(ctx)
	tasksWrite, _ := s.client.Permission.Query().Where(permission.NameEQ("tasks:write")).Only(ctx)
	projectsRead, _ := s.client.Permission.Query().Where(permission.NameEQ("projects:read")).Only(ctx)
	projectsWrite, _ := s.client.Permission.Query().Where(permission.NameEQ("projects:write")).Only(ctx)

	// Admin role (all permissions)
	adminExists, _ := s.client.Role.Query().Where(role.NameEQ("admin")).Exist(ctx)
	if !adminExists {
		adminPerms := make([]*ent.Permission, len(allPerm))
		for i, p := range allPerm {
			adminPerms[i] = p
		}
		_, err := s.client.Role.Create().
			SetID("admin").
			SetName("admin").
			SetDescription("Administrator - full access").
			AddPermissions(adminPerms...).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create admin role: %w", err)
		}
	}

	// Developer role
	devExists, _ := s.client.Role.Query().Where(role.NameEQ("developer")).Exist(ctx)
	if !devExists {
		devPerms := []*ent.Permission{tasksRead, tasksWrite, projectsRead, projectsWrite}
		_, err := s.client.Role.Create().
			SetID("developer").
			SetName("developer").
			SetDescription("Developer - can manage tasks and projects").
			AddPermissions(devPerms...).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create developer role: %w", err)
		}
	}

	// Viewer role
	viewerExists, _ := s.client.Role.Query().Where(role.NameEQ("viewer")).Exist(ctx)
	if !viewerExists {
		viewerPerms := []*ent.Permission{tasksRead, projectsRead}
		_, err := s.client.Role.Create().
			SetID("viewer").
			SetName("viewer").
			SetDescription("Viewer - read-only access").
			AddPermissions(viewerPerms...).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create viewer role: %w", err)
		}
	}

	return nil
}
