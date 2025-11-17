# 🤝 Contributing to Conduit-Go

Thank you for your interest in contributing to Conduit-Go! We welcome contributions from everyone.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Documentation](#documentation)

---

## 📜 Code of Conduct

### Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Our Standards

**Examples of behavior that contributes to a positive environment:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

**Examples of unacceptable behavior:**
- The use of sexualized language or imagery
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information without explicit permission
- Other conduct which could reasonably be considered inappropriate

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by contacting the project team at **ahmet.altun60@gmail.com**. All complaints will be reviewed and investigated promptly and fairly.

---

## 🚀 Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
```bash
   git clone https://github.com/YOUR_USERNAME/conduit-go.git
   cd conduit-go
```

3. **Add upstream remote:**
```bash
   git remote add upstream https://github.com/biyonik/conduit-go.git
```

4. **Create a feature branch:**
```bash
   git checkout -b feature/your-feature-name
```

---

## 🛠️ Development Setup

### Prerequisites

- Go 1.25 or higher
- MySQL 8.0 or higher
- Redis 7.0 or higher (for Phase 3+)
- Docker & Docker Compose (recommended)

### Installation
```bash
# 1. Install dependencies
go mod download

# 2. Copy environment file
cp .env.example .env

# 3. Edit .env with your credentials
nano .env

# 4. Start Docker services
docker-compose up -d

# 5. Wait for services to be ready
sleep 10

# 6. Run the application
make run
```

### Verify Setup
```bash
# Check if server is running
curl http://localhost:8000/health

# Expected response:
# {"success":true,"data":{"status":"healthy",...}}
```

---

## 💻 How to Contribute

### Types of Contributions

We welcome:
- 🐛 **Bug fixes**
- ✨ **New features**
- 📚 **Documentation improvements**
- 🧪 **Test coverage**
- 🎨 **Code refactoring**
- 🌍 **Translations**
- 💡 **Feature suggestions**

### Before You Start

1. **Check existing issues** to avoid duplicate work
2. **Open an issue** to discuss major changes
3. **Follow the coding standards** (see below)
4. **Write tests** for new features
5. **Update documentation** if needed

---

## 📝 Coding Standards

### Go Style Guide

We follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

### File Structure
```go
// -----------------------------------------------------------------------------
// Package Title
// -----------------------------------------------------------------------------
// Brief description of what this file does.
//
// Detailed explanation if needed.
// -----------------------------------------------------------------------------

package packagename

import (
    // Standard library imports first
    "context"
    "net/http"
    
    // External imports second
    "github.com/some/package"
    
    // Internal imports last
    "github.com/biyonik/conduit-go/internal/..."
)
```

### Function Documentation
```go
// FunctionName, kısa açıklama (Türkçe).
//
// Detaylı açıklama (Türkçe)...
//
// Parametreler:
//   - param1: Açıklama
//   - param2: Açıklama
//
// Döndürür:
//   - type: Açıklama
//   - error: Hata durumu
//
// Örnek:
//   result, err := FunctionName(param1, param2)
//   if err != nil {
//       return err
//   }
//
// Güvenlik Notu (varsa):
// - Önemli güvenlik uyarıları
func FunctionName(param1 Type, param2 Type) (Type, error) {
    // Implementation
}
```

### Naming Conventions

- **Variables**: camelCase (`userName`, `accessToken`)
- **Constants**: UPPER_SNAKE_CASE (`MAX_RETRY_COUNT`)
- **Structs**: PascalCase (`UserController`, `AuthService`)
- **Interfaces**: PascalCase with "-er" suffix (`Validator`, `Handler`)
- **Files**: snake_case (`auth_controller.go`, `user_repository.go`)

### Error Handling
```go
// ✅ GOOD: Descriptive error messages
if err != nil {
    logger.Printf("❌ Failed to connect to database: %v", err)
    return fmt.Errorf("database connection failed: %w", err)
}

// ❌ BAD: Generic errors
if err != nil {
    return err
}
```

### Logging Standards

Use emojis for quick visual identification:
```go
logger.Println("📝 Normal log message")
logger.Printf("✅ Success: %s", detail)
logger.Printf("⚠️  Warning: %s", warning)
logger.Printf("❌ Error: %v", err)
logger.Printf("🔐 Authentication event")
logger.Printf("🔄 Token refresh")
logger.Printf("📧 Email sent")
logger.Printf("💾 Data saved")
```

### Security Best Practices

1. **Never log sensitive data** (passwords, tokens, credit cards)
2. **Always use prepared statements** for SQL queries
3. **Validate all user inputs** before processing
4. **Use bcrypt** for password hashing (never plain text)
5. **Keep dependencies updated** for security patches

---

## 📝 Commit Guidelines

### Commit Message Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code formatting (no logic change)
- **refactor**: Code refactoring
- **test**: Adding or updating tests
- **chore**: Maintenance tasks

### Examples
```bash
# Good commit messages
git commit -m "feat(auth): add JWT refresh token rotation"
git commit -m "fix(validation): handle empty email field"
git commit -m "docs(readme): update installation instructions"
git commit -m "test(auth): add login integration tests"

# Bad commit messages
git commit -m "fixed stuff"
git commit -m "updates"
git commit -m "WIP"
```

### Detailed Example
```
feat(queue): implement Redis-based queue system

- Add Queue interface with Dispatch, Process, and Retry methods
- Implement RedisQueue with connection pooling
- Add Worker with graceful shutdown support
- Include retry mechanism with exponential backoff

Closes #123
```

---

## 🔄 Pull Request Process

### Before Submitting

1. **Sync with upstream:**
```bash
   git fetch upstream
   git rebase upstream/main
```

2. **Run tests:**
```bash
   make test
```

3. **Check code quality:**
```bash
   make lint
   make fmt
```

4. **Update documentation** if needed

### PR Template

When opening a PR, include:
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] No new warnings

## Related Issue
Closes #123

## Screenshots (if applicable)
[Add screenshots here]

## Additional Notes
[Any additional information]
```

### Review Process

1. At least **one maintainer approval** required
2. All **CI checks must pass**
3. **Conflicts must be resolved**
4. **Code coverage** should not decrease

### After Approval

Your PR will be merged by a maintainer. Thank you for your contribution! 🎉

---

## 🧪 Testing

### Running Tests
```bash
# Run all tests
make test

# Run specific test file
go test -v ./tests/auth_test.go

# Run with coverage
make test-coverage

# Run security audit
make security
```

### Writing Tests
```go
func TestFeatureName_Success(t *testing.T) {
    // Setup
    // ...
    
    // Execute
    result, err := Function(input)
    
    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Test Coverage Goals

- **Minimum**: 70% code coverage
- **Target**: 80% code coverage
- **Critical paths**: 100% coverage (auth, security, payments)

---

## 📚 Documentation

### What to Document

1. **Code comments**: Complex logic, algorithms, workarounds
2. **Function documentation**: Parameters, return values, examples
3. **README updates**: For new features or changed behavior
4. **API documentation**: For new endpoints
5. **Migration guides**: For breaking changes

### Documentation Style

- **Clear and concise**: Avoid jargon
- **Examples included**: Show practical usage
- **Security notes**: Highlight security implications
- **Language**: Turkish for comments, English for README

---

## 🏆 Recognition

Contributors will be recognized in:
- **README.md** Contributors section
- **CHANGELOG.md** for each release
- **GitHub Releases** notes

---

## 📞 Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and ideas
- **Email**: ahmet.altun60@gmail.com

---

## 📄 License

By contributing to Conduit-Go, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

**Thank you for contributing to Conduit-Go!** 🚀

Every contribution, no matter how small, makes a difference.

