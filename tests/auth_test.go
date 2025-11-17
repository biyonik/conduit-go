// -----------------------------------------------------------------------------
// Authentication Tests
// -----------------------------------------------------------------------------
// Bu dosya, authentication sisteminin integration testlerini içerir.
// -----------------------------------------------------------------------------

package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/internal/controllers"
	"github.com/biyonik/conduit-go/internal/middleware"
	"github.com/biyonik/conduit-go/internal/models"
	"github.com/biyonik/conduit-go/internal/router"
	"github.com/biyonik/conduit-go/pkg/auth"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
)

// setupTestRouter, test için router ve controller'ları hazırlar.
func setupTestRouter(t *testing.T) (*router.Router, *controllers.AuthController, func()) {
	// Container setup
	c := container.New()

	cfg := config.Load()
	c.Register(func(c *container.Container) (*config.Config, error) {
		return cfg, nil
	})

	db, err := database.Connect(cfg.DB.DSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	c.Register(func(c *container.Container) (*sql.DB, error) {
		return db, nil
	})

	c.Register(func(c *container.Container) (database.Grammar, error) {
		return database.NewMySQLGrammar(), nil
	})

	c.Register(controllers.NewAuthController)

	authController := c.MustGet(reflect.TypeOf((*controllers.AuthController)(nil))).(*controllers.AuthController)

	// Router setup
	r := router.New()

	// Cleanup function
	cleanup := func() {
		db.Close()
	}

	return r, authController, cleanup
}

// TestRegister_Success, başarılı kullanıcı kaydını test eder.
func TestRegister_Success(t *testing.T) {
	r, authController, cleanup := setupTestRouter(t)
	defer cleanup()

	r.POST("/api/auth/register", authController.Register)

	// Unique email oluştur (test her çalıştığında farklı)
	email := "testuser_" + time.Now().Format("20060102150405") + "@example.com"

	reqBody := map[string]string{
		"name":             "Test User",
		"email":            email,
		"password":         "Secret123!",
		"password_confirm": "Secret123!",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}

	data := response["data"].(map[string]interface{})
	if data["access_token"] == nil {
		t.Error("Expected access_token in response")
	}

	if data["refresh_token"] == nil {
		t.Error("Expected refresh_token in response")
	}
}

// TestRegister_DuplicateEmail, duplicate email ile kayıt testini yapar.
func TestRegister_DuplicateEmail(t *testing.T) {
	r, authController, cleanup := setupTestRouter(t)
	defer cleanup()

	r.POST("/api/auth/register", authController.Register)

	// Aynı email ile iki kez kayıt dene
	email := "duplicate_" + time.Now().Format("20060102150405") + "@example.com"

	reqBody := map[string]string{
		"name":             "Test User",
		"email":            email,
		"password":         "Secret123!",
		"password_confirm": "Secret123!",
	}

	// İlk kayıt (başarılı olmalı)
	body, _ := json.Marshal(reqBody)
	req1 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Errorf("First registration should succeed, got %d", w1.Code)
	}

	// İkinci kayıt (başarısız olmalı)
	body, _ = json.Marshal(reqBody)
	req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422 for duplicate email, got %d", w2.Code)
	}
}

// TestRegister_WeakPassword, zayıf şifre ile kayıt testini yapar.
func TestRegister_WeakPassword(t *testing.T) {
	r, authController, cleanup := setupTestRouter(t)
	defer cleanup()

	r.POST("/api/auth/register", authController.Register)

	reqBody := map[string]string{
		"name":             "Test User",
		"email":            "weakpass@example.com",
		"password":         "weak", // Çok basit şifre
		"password_confirm": "weak",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422 for weak password, got %d", w.Code)
	}
}

// TestLogin_Success, başarılı login testini yapar.
func TestLogin_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := models.NewUserRepository(db, database.NewMySQLGrammar())

	// Test user oluştur
	email := "logintest_" + time.Now().Format("20060102150405") + "@example.com"
	user := &models.User{
		Name:     "Login Test",
		Email:    email,
		Password: auth.MustHash("Secret123!"),
		Status:   "active",
	}
	userID, _ := userRepo.Create(user)
	defer userRepo.Delete(userID)

	// Login dene
	r, authController, _ := setupTestRouter(t)
	r.POST("/api/auth/login", authController.Login)

	reqBody := map[string]string{
		"email":    email,
		"password": "Secret123!",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}

	data := response["data"].(map[string]interface{})
	if data["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
}

// TestLogin_InvalidCredentials, geçersiz şifre ile login testini yapar.
func TestLogin_InvalidCredentials(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := models.NewUserRepository(db, database.NewMySQLGrammar())

	// Test user oluştur
	email := "invalidcreds_" + time.Now().Format("20060102150405") + "@example.com"
	user := &models.User{
		Name:     "Invalid Creds Test",
		Email:    email,
		Password: auth.MustHash("CorrectPassword123!"),
		Status:   "active",
	}
	userID, _ := userRepo.Create(user)
	defer userRepo.Delete(userID)

	// Yanlış şifre ile login dene
	r, authController, _ := setupTestRouter(t)
	r.POST("/api/auth/login", authController.Login)

	reqBody := map[string]string{
		"email":    email,
		"password": "WrongPassword123!",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestProtectedRoute_WithValidToken, geçerli token ile protected route erişimini test eder.
func TestProtectedRoute_WithValidToken(t *testing.T) {
	r, authController, cleanup := setupTestRouter(t)
	defer cleanup()

	r.GET("/api/auth/profile", authController.Profile).
		Middleware(middleware.Auth())

	// Token oluştur
	token, _ := auth.GenerateToken(123, "test@example.com", "user", nil)

	req := httptest.NewRequest("GET", "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// User database'de olmadığı için 404 dönecek, ama bu authentication geçti demektir
	// Authentication başarısız olsaydı 401 dönerdi
	if w.Code == http.StatusUnauthorized {
		t.Errorf("Valid token should not return 401, got %d", w.Code)
	}
}

// TestProtectedRoute_WithoutToken, token olmadan protected route erişimini test eder.
func TestProtectedRoute_WithoutToken(t *testing.T) {
	r, authController, cleanup := setupTestRouter(t)
	defer cleanup()

	r.GET("/api/auth/profile", authController.Profile).
		Middleware(middleware.Auth())

	req := httptest.NewRequest("GET", "/api/auth/profile", nil)
	// Authorization header yok

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without token, got %d", w.Code)
	}
}

// TestProtectedRoute_WithExpiredToken, expired token ile erişim testini yapar.
func TestProtectedRoute_WithExpiredToken(t *testing.T) {
	r, authController, cleanup := setupTestRouter(t)
	defer cleanup()

	r.GET("/api/auth/profile", authController.Profile).
		Middleware(middleware.Auth())

	// Expired token oluştur (geçmiş tarih)
	expiredConfig := &auth.JWTConfig{
		Secret:         "test-secret",
		Issuer:         "test",
		ExpirationTime: -1 * time.Hour, // 1 saat önce expire olmuş
	}

	token, _ := auth.GenerateToken(123, "test@example.com", "user", expiredConfig)

	req := httptest.NewRequest("GET", "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for expired token, got %d", w.Code)
	}
}

// TestPasswordHash, password hashing fonksiyonlarını test eder.
func TestPasswordHash(t *testing.T) {
	password := "MySecretPassword123!"

	// Hash oluştur
	hash, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("Hash generation failed: %v", err)
	}

	// Hash boş olmamalı
	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Doğru şifre check edilmeli
	if !auth.Check(password, hash) {
		t.Error("Password check should return true for correct password")
	}

	// Yanlış şifre false dönmeli
	if auth.Check("WrongPassword", hash) {
		t.Error("Password check should return false for incorrect password")
	}
}

// TestRoleMiddleware, role-based authorization testini yapar.
func TestRoleMiddleware(t *testing.T) {
	r := router.New()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	r.GET("/admin", testHandler).
		Middleware(middleware.Auth()).
		Middleware(middleware.Admin())

	// Admin token
	adminToken, _ := auth.GenerateToken(1, "admin@example.com", "admin", nil)
	req1 := httptest.NewRequest("GET", "/admin", nil)
	req1.Header.Set("Authorization", "Bearer "+adminToken)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Admin should have access, got %d", w1.Code)
	}

	// User token (admin değil)
	userToken, _ := auth.GenerateToken(2, "user@example.com", "user", nil)
	req2 := httptest.NewRequest("GET", "/admin", nil)
	req2.Header.Set("Authorization", "Bearer "+userToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Non-admin user should be forbidden, got %d", w2.Code)
	}
}
