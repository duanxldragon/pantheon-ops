# Tenant Management API Documentation

Version: 1.0  
Base URL: `/api/v1`

## Overview

The Tenant Management API provides endpoints for managing multi-tenant resources in Pantheon Base. All tenant management operations require `platform_ops` role except for user-facing endpoints.

## Authentication

All endpoints require authentication via Bearer token:

```
Authorization: Bearer <your-jwt-token>
```

## Endpoints

### 1. Create Tenant

**POST** `/tenants`

Creates a new tenant.

**Authorization**: Requires `platform_ops` role

**Request Body**:
```json
{
  "code": "acme-corp",
  "name": "ACME Corporation",
  "plan": "enterprise",
  "metadata": "{\"industry\":\"technology\"}"
}
```

**Response** (201 Created):
```json
{
  "id": 1,
  "code": "acme-corp",
  "name": "ACME Corporation",
  "status": "active",
  "plan": "enterprise",
  "metadata": "{\"industry\":\"technology\"}",
  "created_at": "2026-09-29T10:00:00Z",
  "updated_at": "2026-09-29T10:00:00Z"
}
```

**Errors**:
- `400 Bad Request`: Invalid input
- `409 Conflict`: Tenant code already exists

---

### 2. List Tenants

**GET** `/tenants`

Lists all tenants with pagination and filtering.

**Authorization**: Requires `platform_ops` role

**Query Parameters**:
- `status` (optional): Filter by status (`active`, `suspended`, `deleted`)
- `plan` (optional): Filter by plan (`free`, `basic`, `professional`, `enterprise`)
- `page` (optional): Page number (default: 1)
- `size` (optional): Page size (default: 20, max: 100)

**Response** (200 OK):
```json
{
  "items": [
    {
      "id": 1,
      "code": "acme-corp",
      "name": "ACME Corporation",
      "status": "active",
      "plan": "enterprise",
      "created_at": "2026-09-29T10:00:00Z",
      "updated_at": "2026-09-29T10:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "size": 20
}
```

---

### 3. Get Tenant

**GET** `/tenants/:id`

Retrieves a tenant by ID.

**Authorization**: Requires `platform_ops` role

**Response** (200 OK):
```json
{
  "id": 1,
  "code": "acme-corp",
  "name": "ACME Corporation",
  "status": "active",
  "plan": "enterprise",
  "created_at": "2026-09-29T10:00:00Z",
  "updated_at": "2026-09-29T10:00:00Z"
}
```

**Errors**:
- `404 Not Found`: Tenant not found

---

### 4. Update Tenant

**PUT** `/tenants/:id`

Updates tenant properties.

**Authorization**: Requires `platform_ops` role

**Request Body**:
```json
{
  "name": "ACME Corporation Ltd",
  "status": "suspended",
  "plan": "professional"
}
```

All fields are optional. Only provided fields will be updated.

**Response** (200 OK):
```json
{
  "message": "tenant updated successfully"
}
```

**Errors**:
- `404 Not Found`: Tenant not found

---

### 5. Delete Tenant

**DELETE** `/tenants/:id`

Marks a tenant as deleted (soft delete).

**Authorization**: Requires `platform_ops` role

**Response** (200 OK):
```json
{
  "message": "tenant deleted successfully"
}
```

**Errors**:
- `400 Bad Request`: Tenant has active members
- `404 Not Found`: Tenant not found

**Note**: A tenant can only be deleted if it has no active members. Remove all members first.

---

### 6. Add Tenant Member

**POST** `/tenants/:id/members`

Adds a user to a tenant.

**Authorization**: Requires `platform_ops` role

**Request Body**:
```json
{
  "user_id": 100,
  "role": "admin"
}
```

**Roles**:
- `owner`: Full control over tenant
- `admin`: Administrative access
- `member`: Standard member access

**Response** (201 Created):
```json
{
  "message": "member added successfully"
}
```

**Errors**:
- `404 Not Found`: Tenant or user not found
- `409 Conflict`: User is already a member

---

### 7. List Tenant Members

**GET** `/tenants/:id/members`

Lists one bounded page of active tenant members, ordered by creation time and ID. The response envelope is stable for all consumers; the `/members/page` path below remains as a compatibility alias.

**Authorization**: Requires `platform_ops` role

**Query parameters**: `page` (default 1, minimum 1), `pageSize` (default 20, maximum 100). Invalid or non-positive values use the defaults; larger page sizes are capped at 100.

**Response** (200 OK):
```json
{
  "items": [
    {
      "id": 1,
      "tenant_id": 1,
      "user_id": 100,
      "role": "owner",
      "status": "active",
      "created_at": "2026-09-29T10:00:00Z",
      "updated_at": "2026-09-29T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

**GET** `/tenants/:id/members/page`

Compatibility alias for the same bounded pagination contract and response envelope.
### 8. Remove Tenant Member

**DELETE** `/tenants/:id/members/:user_id`

Removes a user from a tenant.

**Authorization**: Requires `platform_ops` role

**Response** (200 OK):
```json
{
  "message": "member removed successfully"
}
```

**Errors**:
- `404 Not Found`: Membership not found

---

### 9. Get My Tenants

**GET** `/tenants/my`

Returns all tenants the current authenticated user belongs to.

**Authorization**: Requires authentication (any authenticated user)

**Response** (200 OK):
```json
[
  {
    "id": 1,
    "code": "acme-corp",
    "name": "ACME Corporation",
    "status": "active",
    "plan": "enterprise",
    "created_at": "2026-09-29T10:00:00Z",
    "updated_at": "2026-09-29T10:00:00Z"
  }
]
```

---

## Usage Examples

### Create a new tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "acme-corp",
    "name": "ACME Corporation",
    "plan": "enterprise"
  }'
```

### Add admin user to tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants/1/members \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 100,
    "role": "owner"
  }'
```

### List my tenants

```bash
curl -X GET http://localhost:8080/api/v1/tenants/my \
  -H "Authorization: Bearer <token>"
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "error description"
}
```

Common HTTP status codes:
- `200 OK`: Success
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid input
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict
- `500 Internal Server Error`: Server error
