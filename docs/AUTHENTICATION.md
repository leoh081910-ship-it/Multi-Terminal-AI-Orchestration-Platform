# Authentication API Documentation

## Overview

The AI Orchestration Platform uses token-based authentication with Bearer tokens. All authenticated endpoints require a valid API token in the Authorization header.

## Authentication Flow

1. **Bootstrap**: Create initial admin user and token using the bootstrap command
2. **Token Creation**: Authenticated users can create additional tokens
3. **API Access**: Include token in Authorization header for all API requests
4. **Token Management**: Revoke or delete tokens as needed

## Bootstrap Command

Create the initial admin user and token:

```bash
# Set admin credentials via environment variables
export ADMIN_USERNAME=admin
export ADMIN_EMAIL=admin@example.com
export ADMIN_PASSWORD=your-secure-password

# Run bootstrap
./bin/bootstrap.exe
```

The bootstrap command will output an admin token. **Save this token immediately** - it will not be shown again.

## API Endpoints

### Base URL

All authentication endpoints are under `/api/v1/auth/tokens`

### Create Token

Create a new API token for the authenticated user.

**Endpoint**: `POST /api/v1/auth/tokens`

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "name": "My API Token",
  "description": "Token for CI/CD pipeline",
  "scopes": ["tasks:read", "tasks:write"],
  "expires_in": 86400
}
```

**Fields**:
- `name` (required): Human-readable token name
- `description` (optional): Token description
- `scopes` (optional): Array of permission scopes
- `expires_in` (optional): Expiration time in seconds

**Response** (201 Created):
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "name": "My API Token",
    "description": "Token for CI/CD pipeline",
    "user_id": "user-123",
    "scopes": ["tasks:read", "tasks:write"],
    "created_at": "2026-05-03T03:00:00Z",
    "expires_at": "2026-05-04T03:00:00Z",
    "revoked": false
  }
}
```

**Note**: The `token` field is only returned on creation. Store it securely.

### List Tokens

List all tokens for the authenticated user.

**Endpoint**: `GET /api/v1/auth/tokens`

**Authentication**: Required (Bearer token)

**Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "My API Token",
      "description": "Token for CI/CD pipeline",
      "user_id": "user-123",
      "scopes": ["tasks:read", "tasks:write"],
      "created_at": "2026-05-03T03:00:00Z",
      "expires_at": "2026-05-04T03:00:00Z",
      "last_used_at": "2026-05-03T03:30:00Z",
      "revoked": false
    }
  ]
}
```

### Revoke Token

Revoke a token, preventing further use.

**Endpoint**: `POST /api/v1/auth/tokens/{id}/revoke`

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "reason": "Token compromised"
}
```

**Response** (204 No Content)

### Delete Token

Permanently delete a token.

**Endpoint**: `DELETE /api/v1/auth/tokens/{id}`

**Authentication**: Required (Bearer token)

**Response** (204 No Content)

## Using Tokens

Include the token in the Authorization header with Bearer scheme:

```bash
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  http://localhost:8080/api/v1/tasks
```

## Token Scopes

Scopes control what actions a token can perform:

- `*`: All permissions (admin)
- `tasks:read`: Read task information
- `tasks:write`: Create and update tasks
- `tasks:delete`: Delete tasks
- `projects:read`: Read project information
- `projects:write`: Create and update projects
- `orgs:read`: Read organization information
- `orgs:write`: Manage organizations

## Security Best Practices

1. **Store Tokens Securely**: Never commit tokens to version control
2. **Use Short Expiration**: Set appropriate expiration times for tokens
3. **Rotate Regularly**: Create new tokens and revoke old ones periodically
4. **Scope Appropriately**: Grant minimum required permissions
5. **Revoke Immediately**: Revoke tokens if compromised
6. **Monitor Usage**: Check `last_used_at` to identify unused tokens

## Error Responses

### 401 Unauthorized

Missing or invalid token:
```json
{
  "success": false,
  "error": "invalid or expired token"
}
```

### 403 Forbidden

Insufficient permissions:
```json
{
  "success": false,
  "error": "insufficient permissions"
}
```

### 400 Bad Request

Invalid request:
```json
{
  "success": false,
  "error": "name is required"
}
```

## Migration Guide

### Existing APIs

Currently, most API endpoints do not require authentication. To enable authentication:

1. **Optional Authentication**: Use `OptionalAuthenticate` middleware to accept both authenticated and unauthenticated requests
2. **Gradual Migration**: Migrate endpoints one by one to required authentication
3. **Deprecation Notice**: Announce deprecation timeline for unauthenticated access

### Example Migration

Before:
```go
r.Get("/api/v1/tasks", handler.ListTasks)
```

After (optional auth):
```go
r.With(authMiddleware.OptionalAuthenticate).Get("/api/v1/tasks", handler.ListTasks)
```

After (required auth):
```go
r.With(authMiddleware.Authenticate).Get("/api/v1/tasks", handler.ListTasks)
```

With scope requirements:
```go
r.With(
  authMiddleware.Authenticate,
  authMiddleware.RequireScopes("tasks:read"),
).Get("/api/v1/tasks", handler.ListTasks)
```
