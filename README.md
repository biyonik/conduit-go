# 🚀 Conduit-Go Framework

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

Modern, Laravel-inspired web framework for Go. Built with security, performance, and developer experience in mind.

## ✨ Features

### 🔒 Phase 1: Security & Stability (✅ COMPLETED)

- **SQL Injection Protection**
- **CSRF Protection**
- **Rate Limiting**
- **Graceful Shutdown**
- **Memory Leak Protection**

### 🔐 Phase 2: Authentication & Authorization (✅ COMPLETED)

- **JWT Authentication**
    - Access & refresh tokens
    - Token rotation
    - Secure token storage

- **User Management**
    - Registration with validation
    - Login with bcrypt password hashing
    - Profile management
    - Password change

- **Password Reset**
    - Forgot password flow
    - Secure reset tokens (1-hour expiry)
    - Email notifications (ready for Phase 3)

- **Role-Based Authorization**
    - Admin, Editor, User roles
    - Protected routes
    - Policy-based access control

### 🔄 Phase 3: Advanced Features (✅ COMPLETED)

#### Queue System
- **Redis Queue**
    - Push/Later (immediate/delayed dispatch)
    - Pop (blocking job fetch)
    - Failed job handling
    - Retry mechanism with exponential backoff

- **Job System**
    - Job interface with Handle() and Failed()
    - Serialization/deserialization
    - Job metadata (ID, attempts, queue name)
    - Job registry for type mapping

- **Worker**
    - Multiple queue support
    - Graceful shutdown
    - Concurrent processing
    - Failed job tracking

#### Event System
- **Event Dispatcher**
    - Laravel-inspired event-driven architecture
    - Multiple listeners per event
    - Async/sync dispatch
    - Thread-safe operations

- **Event Features**
    - Built-in events (user, email, payment, cache)
    - Custom event creation
    - Conditional listeners
    - Async listeners for slow operations
    - Event statistics and monitoring

#### Mail System
- **SMTP Driver**
    - Send emails via any SMTP server
    - Support for Gmail, SendGrid, AWS SES, Mailhog
    - HTML & plain text emails
    - File attachments
    - Multiple recipients (To, Cc, Bcc)

- **Message Builder**
    - Fluent API for email construction
    - Priority levels (High, Normal, Low)
    - Custom headers support
    - Reply-To support

- **Drivers**
    - SMTP (production)
    - Log (development/testing)

#### Storage System
- **Local Storage**
    - Local filesystem operations
    - Path traversal protection
    - Stream support for large files
    - Directory management
    - URL generation

- **Storage Features**
    - Upload/download files
    - File existence checks
    - File size and metadata
    - Unique name generation
    - Image detection helpers

## 📋 Usage Examples

### Event System

```go
// Create dispatcher
dispatcher := events.NewDispatcher(logger)

// Register listener
dispatcher.Listen("user.registered", events.ListenerFunc(func(e events.Event) error {
    user := e.Payload().(*models.User)
    log.Printf("New user: %s", user.Email)
    return nil
}))

// Dispatch event
event := events.NewUserRegisteredEvent(user)
dispatcher.Dispatch(event)

// Async dispatch (non-blocking)
dispatcher.DispatchAsync(event)
```

### Mail System

```go
// Configure SMTP (Mailhog for development)
config := &mail.SMTPConfig{
    Host: "localhost",
    Port: 1025,
    From: mail.Address{Email: "noreply@conduit.com", Name: "Conduit"},
}
mailer := mail.NewSMTPMailer(config, logger)

// Send email
message := mail.NewMessage().
    To("user@example.com", "John Doe").
    Subject("Welcome to Conduit!").
    Body("Thank you for joining.").
    Html("<h1>Welcome!</h1>")

err := mailer.Send(message)
```

### Storage System

```go
// Initialize local storage
storage, _ := storage.NewLocalStorage("/var/www/uploads", logger)
storage.SetBaseURL("https://cdn.myapp.com")

// Upload file
imageData := []byte{...}
storage.Put("avatars/user-1.jpg", imageData)

// Get URL
url := storage.Url("avatars/user-1.jpg")
// → "https://cdn.myapp.com/avatars/user-1.jpg"

// Stream large file
file, _ := os.Open("large-video.mp4")
storage.PutFile("videos/video.mp4", file)

// Download
data, _ := storage.Get("avatars/user-1.jpg")
```

### Queue System

```go
// Create a job
emailJob := jobs.NewSendEmailJob(
    "user@example.com",
    "Welcome to Conduit-Go",
    "Hello! Welcome to our platform.",
)

// Push to queue (immediate)
queue.Push(emailJob, "emails")

// Push to queue (delayed)
queue.Later(5*time.Minute, emailJob, "emails")
```

### Running Workers
```bash
# Start worker for default queue
make worker

# Start worker for specific queues
make worker-emails

# Start worker for all queues
make worker-all

# Or directly
go run cmd/worker/main.go emails notifications
```

### Creating Custom Jobs
```go
package jobs

import (
    "encoding/json"
    "github.com/biyonik/conduit-go/pkg/queue"
)

type MyCustomJob struct {
    queue.BaseJob
    Data string `json:"data"`
}

func (j *MyCustomJob) Handle() error {
    // Your job logic here
    return nil
}

func (j *MyCustomJob) Failed(err error) error {
    // Failed job handling
    return nil
}

func (j *MyCustomJob) GetPayload() ([]byte, error) {
    return json.Marshal(j)
}

func (j *MyCustomJob) SetPayload(data []byte) error {
    return json.Unmarshal(data, j)
}

// Register the job type
func init() {
    queue.RegisterJob("*jobs.MyCustomJob", func() queue.Job {
        return &MyCustomJob{}
    })
}
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25 or higher
- MySQL 8.0 or higher
- Docker (optional)

### Installation
```bash
# Clone repository
git clone https://github.com/yourusername/conduit-go.git
cd conduit-go

# Environment setup
cp .env.example .env
nano .env  # Edit with your credentials

# Start Docker services
docker-compose up -d
sleep 10  # Wait for MySQL

# Run application
make run
```

Server runs on: `http://localhost:8000`

## 📖 API Documentation

### Authentication Endpoints

#### Register
```http
POST /api/auth/register
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "Secret123!",
  "password_confirm": "Secret123!"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "user": {
      "id": 123,
      "name": "John Doe",
      "email": "john@example.com"
    },
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

#### Login
```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "Secret123!"
}
```

#### Get Profile (Protected)
```http
GET /api/auth/profile
Authorization: Bearer {access_token}
```

#### Refresh Token
```http
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}
```

#### Forgot Password
```http
POST /api/auth/forgot-password
Content-Type: application/json

{
  "email": "john@example.com"
}
```

#### Reset Password
```http
POST /api/auth/reset-password
Content-Type: application/json

{
  "token": "abc123...",
  "email": "john@example.com",
  "password": "NewSecret123!",
  "password_confirm": "NewSecret123!"
}
```

### Protected Routes

All `/api/v1/*` routes require authentication:
```http
GET /api/v1/check
Authorization: Bearer {access_token}
```

### Admin Routes

Admin-only routes require admin role:
```http
GET /api/admin/users
Authorization: Bearer {admin_access_token}
```

## 💻 Usage Examples

### Frontend Integration (React/Vue/Angular)
```javascript
// Register
const registerResponse = await fetch('http://localhost:8000/api/auth/register', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    name: 'John Doe',
    email: 'john@example.com',
    password: 'Secret123!',
    password_confirm: 'Secret123!',
  }),
});

const { data } = await registerResponse.json();
localStorage.setItem('access_token', data.access_token);
localStorage.setItem('refresh_token', data.refresh_token);

// Protected API call
const profileResponse = await fetch('http://localhost:8000/api/auth/profile', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
  },
});

// Handle token expiration
if (profileResponse.status === 401) {
  // Refresh token
  const refreshResponse = await fetch('http://localhost:8000/api/auth/refresh', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      refresh_token: localStorage.getItem('refresh_token'),
    }),
  });
  
  const { data } = await refreshResponse.json();
  localStorage.setItem('access_token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token);
  
  // Retry original request
}
```

## 🧪 Testing
```bash
# Run all tests
make test

# Run auth tests only
go test -v ./tests -run Auth

# Run with coverage
make test-coverage

# Security audit
make security
```

## 🔐 Security Best Practices

### Password Requirements

- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 number
- At least 1 special character

### Token Management

- Access tokens expire in 1 hour
- Refresh tokens expire in 7 days
- Tokens use HS256 algorithm
- Secret keys must be stored in environment variables

### Rate Limiting

- Public auth endpoints: 10 requests/minute
- Protected API endpoints: 50 requests/minute
- Admin endpoints: 30 requests/minute

## 📦 Postman Collection

Import `postman/Conduit-Go-API.postman_collection.json` to test all endpoints.

Variables:
- `base_url`: http://localhost:8000
- `access_token`: Auto-populated after login
- `refresh_token`: Auto-populated after login

## 🛠️ Development
```bash
# Format code
make fmt

# Run linter
make lint

# Build binary
make build

# Run with hot reload
make run
```

## 🚦 Roadmap

- [x] **Phase 1: Security & Stability**
- [x] **Phase 2: Authentication & Authorization**
- [ ] **Phase 3: Advanced Features** (Next)
    - Queue system (Redis)
    - Event system
    - Cache facade
    - Mail system
    - File storage
    - Email verification

- [ ] **Phase 4: Developer Experience**
    - CLI tool
    - Code generators
    - Migration system
    - Testing helpers

- [ ] **Phase 5: Production Ready**
    - Distributed tracing
    - Metrics & monitoring
    - Load balancing
    - Docker orchestration

## 📝 Environment Variables
```bash
# Application
APP_NAME=Conduit-Go
APP_ENV=development
PORT=8000

# Database
DB_DSN=user:pass@tcp(localhost:3306)/conduit_go?parseTime=true

# JWT
JWT_SECRET=your-super-secret-key
JWT_EXPIRATION=3600

# Rate Limiting
RATE_LIMIT_MAX_REQUESTS=100
RATE_LIMIT_WINDOW_SECONDS=60
```

## 🤝 Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) to learn about our development process, how to propose bugfixes and improvements, and how to build and test your changes.

### Contributors

Thank you to all the people who have contributed to Conduit-Go! 🎉

<!-- ALL-CONTRIBUTORS-LIST:START -->
- [@biyonik](https://github.com/biyonik) - Creator & Maintainer
<!-- ALL-CONTRIBUTORS-LIST:END -->

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


## 👨‍💻 Author

**Ahmet Altun**
- Email: ahmet.altun60@gmail.com
- GitHub: [@biyonik](https://github.com/biyonik)
- LinkedIn: [linkedin.com/in/biyonik](https://linkedin.com/in/biyonik)

---

Built with ❤️ and ☕ in Turkey