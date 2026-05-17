# 03 — Authentication

## Strategy

Local auth with email/password. Session-based (not JWT) for simplicity with server-rendered pages.

## Flows

### Register

```
POST /auth/register
Body: email, password, confirm_password

1. Validate input (email format, password min 8 chars, passwords match)
2. Check email not already taken
3. Hash password with bcrypt (cost 12)
4. Create User record
5. Create empty UserProfile record
6. Create session
7. Set session cookie
8. Redirect to /profile (first-time setup)
```

### Login

```
POST /auth/login
Body: email, password

1. Find user by email
2. Compare bcrypt hash
3. If invalid → return error partial (HTMX swap)
4. Create session
5. Set session cookie
6. Redirect to / (dashboard)
```

### Logout

```
POST /auth/logout

1. Delete session from DB
2. Clear cookie
3. Redirect to /login
```

## Session Storage

Database table:

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| user_id | uuid | FK users(id) |
| token | varchar(64) | unique, indexed |
| expires_at | timestamptz | |
| created_at | timestamptz | |

- Token: 32 random bytes, hex encoded (64 chars)
- Expiry: 7 days from creation
- Cookie name: `session`
- Cookie flags: httpOnly, secure (in prod), sameSite=lax, path=/

## Middleware

```go
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
    // 1. Read "session" cookie
    // 2. Lookup session by token, check not expired
    // 3. Load user from session.user_id
    // 4. Set user in request context
    // 5. If any step fails → redirect to /login (or 401 for HTMX)
}
```

For HTMX requests (detected via `HX-Request` header), return 401 with `HX-Redirect: /login` header instead of a 302.

## Password Rules

- Minimum 8 characters
- No maximum (bcrypt truncates at 72 bytes, which is fine)
- No complexity requirements (length is king)

## Security

- Bcrypt cost factor: 12
- Session token: crypto/rand, 32 bytes
- Cookie: httpOnly, secure, sameSite=lax
- CSRF: not needed for same-site cookie + HTMX (same-origin requests only)
- Rate limiting: 5 login attempts per minute per IP (future phase)
