# 🚀 Conduit-Go Framework

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

Modern, Laravel-inspired web framework for Go. Built with security, performance, and developer experience in mind.

## ✨ Features

### 🔒 Phase 1: Security & Stability (✅ COMPLETED)

- **SQL Injection Protection**
    - Whitelist-based operator validation
    - Prepared statement bindings
    - Identifier sanitization
    - Direction parameter validation

- **CSRF Protection**
    - Token-based validation
    - Cookie + header support
    - Session-based token storage
    - Timing-attack resistant

- **Rate Limiting**
    - Token bucket algorithm
    - IP-based limiting
    - Configurable limits
    - Memory leak protection

- **Graceful Shutdown**
    - Signal handling (SIGINT, SIGTERM)
    - Active request completion
    - Clean resource cleanup
    - 30-second timeout

- **Memory Leak Protection**
    - Scanner cache cleanup
    - Automatic garbage collection
    - Resource monitoring

## 🏗️ Architecture
```
conduit-go/
├── cmd/api/              # Application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── controllers/      # HTTP controllers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Domain models
│   └── router/           # HTTP router
├── pkg/
│   ├── container/        # DI container
│   ├── database/         # Query builder & ORM
│   └── validation/       # Request validation
├── scripts/db/           # Database scripts
└── tests/                # Test files
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25 or higher
- MySQL 8.0 or higher
- Docker (optional)

### Installation
```bash
# Clone the repository
git clone https://github.com/yourusername/conduit-go.git
cd conduit-go

# Copy environment file
cp .env.example .env

# Edit .env with your database credentials
nano .env

# Install dependencies
go mod download

# Start Docker services (MySQL, Redis, etc.)
docker-compose up -d

# Wait for MySQL to be ready
sleep 10

# Run the application
make run
# or
go run cmd/api/main.go
```

The application will start on `http://localhost:8000`

### Using Docker Only
```bash
# Start all services including the Go app
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 📖 Usage Examples

### Basic Routing
```go
r := router.New()

// Simple GET route
r.GET("/", func(w http.ResponseWriter, r *conduitReq.Request) {
    response.Success(w, 200, "Hello World!", nil)
})

// Route with parameter
r.GET("/users/{id}", func(w http.ResponseWriter, r *conduitReq.Request) {
    id := r.RouteParam("id")
    // ...
})

// Route groups
api := r.Group("/api")
api.GET("/users", UserListHandler)
api.POST("/users", UserCreateHandler)
```

### Query Builder
```go
// Select
var users []User
db.Table("users").
    Where("status", "=", "active").
    OrderBy("created_at", "DESC").
    Limit(10).
    Get(&users)

// Insert
result, err := db.ExecInsert(map[string]interface{}{
    "name": "John Doe",
    "email": "john@example.com",
})
lastID, _ := result.LastInsertId()

// Update
db.Table("users").
    Where("id", "=", 1).
    ExecUpdate(map[string]interface{}{
        "name": "Jane Doe",
    })

// Delete
db.Table("users").
    Where("id", "=", 1).
    ExecDelete()

// Transactions
tx, _ := database.BeginTransaction(db, grammar)
tx.NewBuilder().Table("users").ExecInsert(...)
tx.NewBuilder().Table("posts").ExecInsert(...)
tx.Commit() // or tx.Rollback()
```

### Validation
```go
schema := validation.Make().Shape(map[string]validation.Type{
    "name":  types.String().Required().Min(3).Max(255),
    "email": types.String().Required().Email(),
    "age":   types.Number().Min(18).Integer(),
})

result := schema.Validate(data)
if result.HasErrors() {
    // Handle validation errors
    response.Error(w, 422, result.Errors())
    return
}

validData := result.ValidData()
```

### Middleware
```go
// Global middleware
r.Use(middleware.PanicRecovery(logger))
r.Use(middleware.Logging)
r.Use(middleware.CORSMiddleware("*"))
r.Use(middleware.CSRFProtection())
r.Use(middleware.RateLimit(100, 60)) // 100 req/min

// Route-specific middleware
apiGroup := r.Group("/api")
apiGroup.Use(middleware.RateLimit(50, 60)) // Stricter limit for API
```

## 🧪 Testing
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run security tests
go test -v ./tests -run Security

# Run integration tests (requires database)
go test -v ./tests -run Integration
```

## 🔐 Security

### SQL Injection Prevention
```go
// ✅ SAFE: Uses prepared statements
db.Table("users").Where("email", "=", userInput).Get(&users)

// ✅ SAFE: Direction is whitelist-validated
db.Table("users").OrderBy("name", userInput).Get(&users)

// ❌ UNSAFE: Never do this
db.Query("SELECT * FROM users WHERE email = '" + userInput + "'")
```

### CSRF Protection
```html
<!-- Include CSRF token in forms -->
<form method="POST" action="/api/users">
    <input type="hidden" name="_token" value="{{.csrfToken}}">
    <!-- form fields -->
</form>

<!-- Or in JavaScript -->
<script>
const token = document.cookie.match(/csrf_token=([^;]+)/)[1];
fetch('/api/users', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': token,
        'Content-Type': 'application/json',
    },
    body: JSON.stringify({...}),
});
</script>
```

### Rate Limiting

The framework automatically handles rate limiting. Configure limits in `.env`:
```bash
RATE_LIMIT_MAX_REQUESTS=100
RATE_LIMIT_WINDOW_SECONDS=60
```

## 📊 Monitoring

### Health Check
```bash
curl http://localhost:8000/health
```

Response:
```json
{
    "success": true,
    "data": {
        "status": "healthy",
        "version": "1.0.0",
        "database": "connected"
    }
}
```

### Metrics (Future Phase)

- Request rate
- Response times
- Error rates
- Database query performance

## 🛠️ Development
```bash
# Format code
make fmt

# Run linter
make lint

# Security check
make security

# Build binary
make build

# Clean build artifacts
make clean
```

## 🚦 Roadmap

- [x] **Phase 1: Security & Stability**
    - SQL injection protection
    - CSRF protection
    - Rate limiting
    - Graceful shutdown
    - Memory leak fixes

- [ ] **Phase 2: Authentication & Authorization** (Next)
    - JWT authentication
    - Session-based auth
    - Role-based access control (RBAC)
    - Policy-based authorization

- [ ] **Phase 3: Advanced Features**
    - Queue system
    - Event system
    - Cache facade
    - Mail system
    - File storage

- [ ] **Phase 4: Developer Experience**
    - CLI tool (Cobra-based)
    - Code generators
    - Migration system
    - Testing helpers

- [ ] **Phase 5: Production Ready**
    - Distributed tracing
    - Metrics & monitoring
    - Load balancing
    - Docker orchestration

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👨‍💻 Author

**Ahmet Altun**
- Email: ahmet.altun60@gmail.com
- GitHub: [@biyonik](https://github.com/biyonik)
- LinkedIn: [linkedin.com/in/biyonik](https://linkedin.com/in/biyonik)

## 🙏 Acknowledgments

- Inspired by [Laravel](https://laravel.com/) and [Symfony](https://symfony.com/)
- Built with ❤️ and ☕