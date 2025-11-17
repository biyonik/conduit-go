# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned for Phase 3
- Redis cache system
- Queue system with workers
- Event system
- Mail system
- File storage (Local & S3)
- Session management

## [0.2.0] - 2025-01-18

### Added (Phase 2: Authentication & Authorization)
- JWT-based authentication system
- User registration with strong password validation
- User login with bcrypt password hashing
- Access token & refresh token management
- Token rotation for enhanced security
- Password reset flow with secure tokens
- User profile management
- Password change functionality
- Role-based authorization (Admin, Editor, User)
- Protected routes with authentication middleware
- Role-based middleware for authorization
- Comprehensive authentication tests

### Changed
- Router now supports method chaining for middleware
- Request wrapper enhanced with JSON parsing
- Response helpers standardized across the project

### Security
- Bcrypt password hashing (cost factor: 12)
- JWT token security with HS256 algorithm
- Rate limiting on authentication endpoints (10 req/min)
- CSRF protection on state-changing requests
- User enumeration attack prevention in password reset

## [0.1.0] - 2025-01-15

### Added (Phase 1: Security & Stability)
- SQL injection protection with whitelist validation
- CSRF token-based protection
- Rate limiting with token bucket algorithm
- Graceful shutdown with 30-second timeout
- Memory leak protection with automatic cleanup
- Query Builder with MySQL grammar support
- Transaction support
- Result scanner with struct mapping
- Validation system with multiple types
- DI Container for dependency management
- Router with middleware support
- Panic recovery middleware
- CORS middleware
- Logging middleware
- Docker Compose setup (MySQL, Redis, phpMyAdmin, Mailhog)
- Comprehensive test suite
- Security tests
- Integration tests

### Security
- Prepared statements for all SQL queries
- Identifier sanitization
- Direction parameter validation
- CSRF token with timing-attack resistance
- IP-based rate limiting

## [0.0.1] - 2025-01-10

### Added
- Initial project setup
- Basic project structure
- Go module initialization
- README with project goals

---

## Version History

- **v0.2.0**: Authentication & Authorization (Phase 2)
- **v0.1.0**: Security & Stability (Phase 1)
- **v0.0.1**: Initial Release

---

[Unreleased]: https://github.com/biyonik/conduit-go/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/biyonik/conduit-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/biyonik/conduit-go/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/biyonik/conduit-go/releases/tag/v0.0.1
```

---

## 📦 Dosya Konumları:
```
conduit-go/
├── CONTRIBUTING.md      # ✅ YENİ
├── LICENSE              # ✅ YENİ
├── CHANGELOG.md         # ✅ BONUS
├── README.md            # (Mevcut)
├── SECURITY.md          # (Mevcut)
└── ...