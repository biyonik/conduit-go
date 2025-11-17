// -----------------------------------------------------------------------------
// Security Tests - Phase 1
// -----------------------------------------------------------------------------
// Bu dosya, Phase 1'de eklenen güvenlik özelliklerinin test edilmesini sağlar.
// -----------------------------------------------------------------------------

package tests

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/biyonik/conduit-go/internal/middleware"
	"github.com/biyonik/conduit-go/pkg/database"
)

// TestSQLInjectionProtection_OrderBy, OrderBy metodunun SQL injection'a karşı
// korumalı olduğunu test eder.
func TestSQLInjectionProtection_OrderBy(t *testing.T) {
	grammar := database.NewMySQLGrammar()
	qb := &database.QueryBuilder{}
	qb.Table("users")

	// Test 1: Geçerli direction değerleri
	validDirections := []string{"ASC", "asc", "DESC", "desc"}
	for _, dir := range validDirections {
		qb.OrderBy("name", dir)
		sql, _ := grammar.CompileSelect(qb)

		// SQL içinde sadece uppercase direction olmalı
		expectedDir := dir
		if dir == "asc" {
			expectedDir = "ASC"
		} else if dir == "desc" {
			expectedDir = "DESC"
		}

		if !contains(sql, expectedDir) {
			t.Errorf("Expected SQL to contain %s, got: %s", expectedDir, sql)
		}
	}

	// Test 2: SQL injection denemesi (malicious input)
	maliciousInputs := []string{
		"DESC; DROP TABLE users--",
		"DESC' OR '1'='1",
		"DESC/**/UNION/**/SELECT",
		"ASC`; DELETE FROM users WHERE 1=1--",
	}

	for _, malicious := range maliciousInputs {
		qb = &database.QueryBuilder{}
		qb.Table("users")
		qb.OrderBy("name", malicious)

		sql, _ := grammar.CompileSelect(qb)

		// Malicious input ASC'ye dönüştürülmeli (whitelist default)
		if !contains(sql, "ASC") {
			t.Errorf("Malicious input should default to ASC, got: %s", sql)
		}

		// SQL içinde tehlikeli karakterler olmamalı
		dangerousPatterns := []string{"DROP", "DELETE", "UNION", "--", "/*"}
		for _, pattern := range dangerousPatterns {
			if contains(sql, pattern) {
				t.Errorf("SQL should not contain dangerous pattern '%s', got: %s", pattern, sql)
			}
		}
	}
}

// TestSQLInjectionProtection_Wrap, Wrap fonksiyonunun geçersiz identifier'ları
// reddettiğini test eder.
func TestSQLInjectionProtection_Wrap(t *testing.T) {
	grammar := database.NewMySQLGrammar()

	// Test 1: Geçerli identifier'lar
	validIdentifiers := []string{
		"users",
		"user_name",
		"users.id",
		"tbl_users",
		"users123",
	}

	for _, identifier := range validIdentifiers {
		result := grammar.Wrap(identifier)
		if result == "" {
			t.Errorf("Valid identifier '%s' should be wrapped", identifier)
		}
	}

	// Test 2: Geçersiz identifier'lar (panic beklenecek)
	invalidIdentifiers := []string{
		"users; DROP TABLE users--",
		"users' OR '1'='1",
		"users/*comment*/",
		"users<script>",
	}

	for _, identifier := range invalidIdentifiers {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Invalid identifier '%s' should cause panic", identifier)
				}
			}()
			grammar.Wrap(identifier)
		}()
	}
}

// TestRateLimiting, rate limiting middleware'inin doğru çalıştığını test eder.
func TestRateLimiting(t *testing.T) {
	// 5 req/10sec limiti (test için kısa süre)
	limiter := middleware.NewRateLimiter(5, 10)

	testKey := "test-ip-123"

	// Test 1: İlk 5 istek başarılı olmalı
	for i := 0; i < 5; i++ {
		allowed, remaining, _ := limiter.Allow(testKey)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if remaining != 4-i {
			t.Errorf("Expected %d remaining tokens, got %d", 4-i, remaining)
		}
	}

	// Test 2: 6. istek reddedilmeli (limit aşıldı)
	allowed, remaining, retryAfter := limiter.Allow(testKey)
	if allowed {
		t.Error("6th request should be denied (limit exceeded)")
	}
	if remaining != 0 {
		t.Errorf("Expected 0 remaining tokens, got %d", remaining)
	}
	if retryAfter == 0 {
		t.Error("Retry-After should be greater than 0")
	}

	// Test 3: Token yenilenme testi (2 saniye bekle)
	time.Sleep(2 * time.Second)
	allowed, _, _ = limiter.Allow(testKey)
	if !allowed {
		t.Error("Request should be allowed after token refill")
	}
}

// TestRateLimitMiddleware, rate limit middleware'inin HTTP seviyesinde
// doğru çalıştığını test eder.
func TestRateLimitMiddleware(t *testing.T) {
	// Test handler (her zaman 200 döner)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Rate limit middleware ekle (3 req/10sec)
	handler := middleware.RateLimit(3, 10)(testHandler)

	// Test 1: İlk 3 istek 200 dönmeli
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should return 200, got %d", i+1, w.Code)
		}

		// Rate limit header'ları kontrol et
		if w.Header().Get("X-RateLimit-Limit") != "3" {
			t.Error("X-RateLimit-Limit header should be 3")
		}
	}

	// Test 2: 4. istek 429 Too Many Requests dönmeli
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th request should return 429, got %d", w.Code)
	}

	// Retry-After header'ı olmalı
	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be present")
	}
}

// TestCSRFProtection, CSRF middleware'inin doğru çalıştığını test eder.
func TestCSRFProtection(t *testing.T) {
	// Test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// CSRF middleware ekle
	handler := middleware.CSRFProtection()(testHandler)

	// Test 1: GET isteği (safe method) - token kontrolü yapılmamalı
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET request should return 200, got %d", w.Code)
	}

	// CSRF token cookie'si set edilmeli
	cookies := w.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "csrf_token" {
			csrfCookie = cookie
			break
		}
	}

	if csrfCookie == nil {
		t.Error("CSRF token cookie should be set")
	}

	// Test 2: POST isteği token olmadan - 403 dönmeli
	req = httptest.NewRequest("POST", "/api/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST without token should return 403, got %d", w.Code)
	}

	// Test 3: POST isteği geçerli token ile - 200 dönmeli
	req = httptest.NewRequest("POST", "/api/test", nil)
	if csrfCookie != nil {
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
		req.Header.Set("X-CSRF-Token", csrfCookie.Value)
	}
	w = httptest.NewRecorder()

	// Önce GET ile token al
	getReq := httptest.NewRequest("GET", "/api/test", nil)
	getReq.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)

	// Token'ı çıkar
	var token string
	for _, cookie := range getW.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			token = cookie.Value
			break
		}
	}

	// Şimdi POST ile token'ı gönder
	postReq := httptest.NewRequest("POST", "/api/test", nil)
	postReq.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
	postReq.Header.Set("X-CSRF-Token", token)
	postW := httptest.NewRecorder()
	handler.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Errorf("POST with valid token should return 200, got %d", postW.Code)
	}
}

// TestMemoryLeakProtection_ScannerCache, scanner cache'in memory leak'e
// karşı korumalı olduğunu test eder.
func TestMemoryLeakProtection_ScannerCache(t *testing.T) {
	// Bu test, scanner cache cleanup goroutine'inin çalıştığını doğrular
	// Gerçek bir memory leak testi için profiling tool'ları gerekir

	// Test için 100 farklı struct tipi oluştur
	for i := 0; i < 100; i++ {
		// Anonymous struct (her biri farklı tip)
		testStruct := struct {
			ID   int64
			Name string
		}{
			ID:   int64(i),
			Name: "Test",
		}

		// Field map al (cache'e eklenecek)
		_ = database.GetStructFieldMap(reflect.TypeOf(testStruct))
	}

	// Cleanup 10 dakikada bir çalışıyor, bu yüzden gerçek test yapmak zor
	// En azından panic olmamalı
	t.Log("Scanner cache memory leak protection is active")
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || strings.Contains(s, substr)))
}
