# 🔐 Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Security Features

### Phase 1 (Current)

#### ✅ SQL Injection Protection
- All user inputs are bound using prepared statements
- Column/table identifiers are validated against whitelist patterns
- SQL operators are whitelist-controlled
- Direction parameters (ASC/DESC) are strictly validated

#### ✅ CSRF Protection
- Token-based validation for state-changing requests (POST, PUT, DELETE)
- Timing-attack resistant token comparison
- Session-based token storage
- 2-hour token expiration

#### ✅ Rate Limiting
- Token bucket algorithm prevents brute force attacks
- IP-based rate limiting (configurable per route)
- Default: 100 requests/minute
- Automatic cleanup prevents memory leaks

#### ✅ Memory Safety
- Scanner cache cleanup (30-minute idle timeout)
- Graceful shutdown ensures no resource leaks
- Automatic database connection pooling

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please follow these steps:

### 🚨 DO NOT create a public GitHub issue

Instead:

1. **Email us directly**: ahmet.altun60@gmail.com
2. **Subject line**: `[SECURITY] Conduit-Go Vulnerability Report`
3. **Include**:
    - Description of the vulnerability
    - Steps to reproduce
    - Potential impact
    - Suggested fix (if any)

### Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 7 days
- **Fix timeline**: Depends on severity
    - Critical: 24-48 hours
    - High: 7 days
    - Medium: 30 days
    - Low: 90 days

## Security Best Practices

### For Developers

#### 1. Always Use Prepared Statements
```go
// ✅ SAFE
db.Table("users").Where("email", "=", userInput).Get(&users)

// ❌ NEVER DO THIS
db.Exec("SELECT * FROM users WHERE email = '" + userInput + "'")
```

#### 2. Validate All User Input
```go
schema := validation.Make().Shape(map[string]validation.Type{
    "email": types.String().Required().Email(),
    "age": types.Number().Min(18).Max(120).Integer(),
})

result := schema.Validate(request.Body())
if result.HasErrors() {
    return response.Error(w, 422, result.Errors())
}
```

#### 3. Use CSRF Protection
```go
// Middleware (already applied globally in main.go)
r.Use(middleware.CSRFProtection())

// Frontend: Include token in requests
fetch('/api/users', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': getCsrfToken(),
    },
    body: JSON.stringify(data),
})
```

#### 4. Implement Rate Limiting
```go
// Global rate limit
r.Use(middleware.RateLimit(100, 60))

// Stricter limit for sensitive endpoints
authGroup := r.Group("/auth")
authGroup.Use(middleware.RateLimit(5, 60)) // 5 login attempts/minute
```

#### 5. Never Log Sensitive Data
```go
// ❌ NEVER LOG
logger.Printf("User password: %s", password)
logger.Printf("Credit card: %s", cardNumber)

// ✅ SAFE
logger.Printf("User logged in: %s", userID)
logger.Printf("Payment processed: %s", transactionID)
```

### For Production Deployment

#### 1. Use HTTPS
```go
srv := &http.Server{
    Addr:      ":443",
    Handler:   r,
    TLSConfig: tlsConfig,
}
srv.ListenAndServeTLS("cert.pem", "key.pem")
```

#### 2. Set Secure Headers
```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Strict-Transport-Security", "max-age=31536000")
```

#### 3. Use Environment Variables
```bash
# Never commit secrets to Git!
DB_PASSWORD=super_secret_password
JWT_SECRET=ultra_random_secret_key
```

#### 4. Enable Database SSL
```go
DB_DSN=user:pass@tcp(host:3306)/db?tls=true&parseTime=true
```

#### 5. Monitor & Alert

- Set up error tracking (Sentry, Rollbar)
- Monitor failed login attempts
- Alert on unusual traffic patterns
- Regular security audits

## Known Limitations

### Current Version (Phase 1)

1. **Session Storage**: Currently in-memory (not suitable for multiple instances)
    - **Mitigation**: Phase 3 will add Redis session storage

2. **Rate Limiting**: IP-based only
    - **Mitigation**: Phase 2 will add user-based rate limiting

3. **No Built-in Authentication**
    - **Mitigation**: Phase 2 will add JWT & session-based auth

4. **HTTPS Not Enforced**
    - **Mitigation**: Use reverse proxy (nginx, Caddy) in production

## Security Checklist

Before deploying to production:

- [ ] All database queries use prepared statements
- [ ] CSRF protection enabled
- [ ] Rate limiting configured
- [ ] HTTPS enabled
- [ ] Secure headers set
- [ ] Secrets in environment variables (not in code)
- [ ] Database uses SSL/TLS
- [ ] Error tracking configured
- [ ] Logging excludes sensitive data
- [ ] Regular security updates scheduled

## Disclosure Policy

- We follow **responsible disclosure**
- Security researchers will be credited (if desired)
- Critical vulnerabilities will be patched within 48 hours
- CVE numbers will be assigned for significant vulnerabilities

## Contact

- **Security Email**: ahmet.altun60@gmail.com
- **PGP Key**: Available upon request

---

*Last updated: 2025-11-17*