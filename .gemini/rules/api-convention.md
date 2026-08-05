---
name: API Convention
description: RESTful API design standards for backend services
---

# 📡 API Convention

> **Principle:** Consistency over creativity. Every API must be predictable, self-documenting, and backward-compatible.

---

## 1. URL Structure

```
/{api_prefix}/v{version}/{resource}
```

### Naming Rules
- **Resource names**: `snake_case`, plural nouns — `order_items`, `sessions`, `participants`
- **No verbs in URL**: ❌ `/getUser` → ✅ `GET /users/:id`
- **Nested resources max 2 levels**: `/sessions/:id/participants` (OK) — `/sessions/:id/participants/:pid/orders/:oid` (❌ flatten it)
- **Query params**: `snake_case` — `?page_size=20&sort_by=created_at`

### Versioning
- URL-based: `/api/v1/...`
- Breaking changes → bump version. Non-breaking → keep current version
- Support previous version for minimum **3 months** after deprecation notice

---

## 2. HTTP Methods

| Method | Purpose | Idempotent | Request Body |
|--------|---------|:----------:|:------------:|
| `GET` | Read resource(s) | ✅ | ❌ |
| `POST` | Create resource | ❌ | ✅ |
| `PUT` | Full replace | ✅ | ✅ |
| `PATCH` | Partial update | ✅ | ✅ |
| `DELETE` | Remove resource | ✅ | ❌ |

### Rules
- `POST` returns `201 Created` with the created resource
- `PUT/PATCH` returns `200 OK` with the updated resource
- `DELETE` returns `204 No Content` (no body)
- `GET` collection returns `200 OK` with array (empty array `[]` if none, never `null`)

---

## 3. JSON Response Envelope

### Success Response
```json
{
  "data": { ... },
  "meta": {
    "request_id": "req_abc123",
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

### Collection Response (Paginated)
```json
{
  "data": [ ... ],
  "meta": {
    "request_id": "req_abc123",
    "timestamp": "2025-01-15T10:30:00Z",
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_items": 150,
      "total_pages": 8
    }
  }
}
```

### Error Response
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Human-readable description",
    "details": [
      { "field": "email", "reason": "invalid_format" }
    ],
    "trace_id": "trace_xyz789"
  }
}
```

---

## 4. Status Codes

### Success
| Code | When |
|------|------|
| `200` | Successful GET, PUT, PATCH |
| `201` | Successful POST (resource created) |
| `204` | Successful DELETE (no body) |

### Client Error
| Code | When |
|------|------|
| `400` | Validation error, malformed request |
| `401` | Missing or invalid authentication |
| `403` | Authenticated but not authorized |
| `404` | Resource not found |
| `409` | Conflict (duplicate, state violation) |
| `422` | Semantic error (valid JSON but business logic rejected) |
| `429` | Rate limit exceeded |

### Server Error
| Code | When |
|------|------|
| `500` | Internal server error (unexpected) |
| `503` | Service unavailable (maintenance, overload) |

### Rules
- Never return `200` with an error body
- Never return `500` for validation errors
- Always include `error.code` (machine-readable) + `error.message` (human-readable)

---

## 5. Request Headers (Mandatory)

| Header | Purpose | Required |
|--------|---------|:--------:|
| `Content-Type` | `application/json` | ✅ |
| `X-Request-ID` | Client-generated request tracing | Recommended |
| `X-Device-ID` | Device identification for guest auth | Per project |
| `Authorization` | Bearer token (if auth enabled) | Per project |

---

## 6. Pagination

- Default: **cursor-based** for large datasets, **offset-based** for admin/dashboard
- Default `page_size`: 20, Max: 100
- Always return `pagination` object in `meta`

### Cursor-based (preferred)
```
GET /items?cursor=eyJpZCI6MTAwfQ&page_size=20
```

### Offset-based
```
GET /items?page=2&page_size=20
```

---

## 7. Filtering & Sorting

```
GET /orders?status=pending&sort_by=created_at&sort_order=desc
```

- Filter params = exact field names in `snake_case`
- Sort: `sort_by` + `sort_order` (`asc` | `desc`)
- Date range: `created_after=2025-01-01&created_before=2025-02-01`

---

## 8. Naming Conventions Summary

| Element | Convention | Example |
|---------|-----------|---------|
| URL path | `snake_case` | `/order_items` |
| JSON fields | `snake_case` | `created_at`, `total_amount` |
| Error codes | `UPPER_SNAKE_CASE` | `VALIDATION_FAILED` |
| Query params | `snake_case` | `page_size`, `sort_by` |
| Headers | `X-Pascal-Case` | `X-Device-ID` |

---

## 9. API Documentation

- Every endpoint must have inline documentation
- Include: description, request/response examples, error cases
- API changelog maintained per version
