package auth

import (
	"net/http"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
)

// RBACMiddleware provides role-based access control
type RBACMiddleware struct {
	permService *PermissionService
	authMW      *Middleware
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(client *ent.Client, authMW *Middleware) *RBACMiddleware {
	return &RBACMiddleware{
		permService: NewPermissionService(client),
		authMW:      authMW,
	}
}

// RequirePermission returns middleware that checks if the authenticated user has a specific permission
func (m *RBACMiddleware) RequirePermission(permName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.authMW.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			has, err := m.permService.HasPermission(r.Context(), userID, permName)
			if err != nil {
				http.Error(w, "permission check failed", http.StatusInternalServerError)
				return
			}
			if !has {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// RequireAnyPermission returns middleware that checks if user has any of the listed permissions
func (m *RBACMiddleware) RequireAnyPermission(permNames ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.authMW.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			userPerms, err := m.permService.GetUserPermissions(r.Context(), userID)
			if err != nil {
				http.Error(w, "permission check failed", http.StatusInternalServerError)
				return
			}

			// Create a map for quick lookup
			permMap := make(map[string]bool)
			for _, p := range userPerms {
				permMap[p] = true
			}

			for _, required := range permNames {
				if permMap[required] {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "insufficient permissions", http.StatusForbidden)
		}))
	}
}

// RequireRole checks if user has a specific role (by role name)
func (m *RBACMiddleware) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.authMW.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			roles, err := m.permService.GetUserRoles(r.Context(), userID)
			if err != nil {
				http.Error(w, "role check failed", http.StatusInternalServerError)
				return
			}

			for _, role := range roles {
				if role.Name == roleName {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "insufficient role", http.StatusForbidden)
		}))
	}
}
