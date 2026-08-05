---
name: Security Standards
description: Authentication, authorization, input validation, and security best practices
---

# 🔐 Security Standards

> **Principle:** Security is not a feature — it's a constraint. Every endpoint, every input, every response must assume hostile intent.

---

## 1. Authentication & Identity

### Device-Based Identity (Guest Systems)
- Every request must include identity header (e.g., `X-Device-ID`)
- Device ID is the minimum identity for guest/anonymous systems
- **Never trust client-generated IDs** as sole authorization — always verify server-side

### Token-Based Auth (Authenticated Systems)
- Use **JWT** or **opaque tokens** with short expiry
- Access token: 15–60 minutes
- Refresh token: 7–30 days
- Store refresh tokens server-side (database or Redis)
- **Never** store tokens in localStorage — use httpOnly cookies

### Password Handling
- **bcrypt** or **argon2** for password hashing (min cost 10)
- **Never** log passwords, even partially
- **Never** include passwords in API responses
- Enforce minimum password policy if applicable

---

## 2. Authorization (IDOR Prevention)

### Principle: Verify Ownership at Every Layer

```go
// ✅ Good — verify the requester owns the resource
func (s *Service) DeleteOrder(ctx context.Context, orderID, deviceID uuid.UUID) error {
    order, err := s.repo.FindByID(ctx, orderID)
    if err != nil {
        return err
    }
    if order.DeviceID != deviceID {
        return domain.ErrForbidden // Cannot delete others' orders
    }
    return s.repo.Delete(ctx, orderID)
}

// ❌ Bad — anyone with the order ID can delete
func (s *Service) DeleteOrder(ctx context.Context, orderID uuid.UUID) error {
    return s.repo.Delete(ctx, orderID)
}
```

### Rules
- **Every mutating endpoint** must verify the requester has permission
- **Never rely on URL obscurity** — UUIDs are not access control
- Use middleware for role/permission checks when applicable
- Admin endpoints must have separate auth middleware

---

## 3. Input Validation

### Dual Validation (Both Sides, Always)

| Layer | Tool | Purpose |
|-------|------|---------|
| Frontend | Zod / Yup | UX feedback, type safety |
| Backend | Go validator / custom | Security enforcement |

### What to Validate
- ✅ Data types (string, number, boolean)
- ✅ Length limits (min/max for strings, arrays)
- ✅ Number ranges (min/max, no negative amounts)
- ✅ Format (email, UUID, phone, URL)
- ✅ Enum values (status must be one of [open, locked, completed])
- ✅ Required vs optional fields
- ✅ Array size limits (prevent memory attacks)

### What to Sanitize
- HTML in user input → strip or escape
- SQL in parameters → parameterized queries ONLY
- File names → strip path traversal characters (`../`)
- JSON → validate schema before processing

---

## 4. API Security

### Rate Limiting
- **All public endpoints** must have rate limiting
- Suggested defaults:
  - Guest APIs: 60 requests/minute per device
  - Authenticated APIs: 120 requests/minute per user
  - Auth endpoints (login/register): 10 requests/minute per IP
- Return `429 Too Many Requests` with `Retry-After` header

### CORS (Cross-Origin Resource Sharing)
```go
// ✅ Whitelist specific origins
cors.AllowOrigins([]string{
    "https://yourdomain.com",
    "https://staging.yourdomain.com",
})

// ❌ Never in production
cors.AllowOrigins([]string{"*"})
```

- Development: Allow `localhost` origins
- Staging/Production: Whitelist specific domains only
- **Never** `Access-Control-Allow-Origin: *` on authenticated endpoints

### Request Size Limits
- JSON body: max 1MB (configurable)
- File uploads: max 10MB (configurable, per use case)
- URL query string: max 2048 characters

---

## 5. Response Security

### Never Expose in Responses
- ❌ Password hashes
- ❌ Secret keys / API keys
- ❌ Internal server paths / stack traces
- ❌ Database IDs that leak sequence (use UUID)
- ❌ Raw device information of other users
- ❌ Internal error messages (map to generic codes)

### Security Headers (HTTP)
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

---

## 6. Data Protection

### Sensitive Data at Rest
- Encrypt sensitive fields in database (PII, payment info)
- Use database-level encryption or application-level with envelope encryption
- Encryption keys in secret manager (not in code/config files)

### Sensitive Data in Transit
- **HTTPS only** — HTTP redirects to HTTPS
- TLS 1.2+ minimum
- No sensitive data in URL query parameters (use POST body)
- No sensitive data in server logs

### Secret Management
- **Never** commit secrets to git (use `.gitignore`, `.env.example`)
- Use environment variables or secret managers (Vault, AWS Secrets Manager)
- Rotate secrets periodically
- Different secrets per environment (dev ≠ staging ≠ prod)

---

## 7. Dependency Security

### Automated Scanning
- **Dependabot** or **Snyk** enabled on repository
- Weekly automated dependency updates
- **No HIGH or CRITICAL CVE** in production dependencies
- Review MEDIUM CVEs monthly

### Rules
- Pin dependency versions (lock files committed)
- Review new dependencies before adding (check maintenance, downloads, security track record)
- Prefer well-maintained, widely-used libraries
- Minimize dependency count — less surface area = less risk

---

## 8. Logging & Audit

### Security Event Logging
Log these events with structured data:
- Authentication attempts (success + failure)
- Authorization failures (403)
- Rate limiting triggers
- Input validation failures (potential attacks)
- Admin actions

### Log Rules
- **Never log** sensitive data (passwords, tokens, PII)
- Use structured logging (JSON format)
- Include `trace_id`, `user_id/device_id`, `action`, `result`
- Retain security logs for minimum 90 days
