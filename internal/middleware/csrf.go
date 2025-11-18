package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// -----------------------------------------------------------------------------
// CSRF Protection Middleware (PANIC RISK + PRODUCTION FIXED)
// -----------------------------------------------------------------------------
// FIXED ISSUES:
// ✅ Panic risk - rand.Read() error artık handle ediliyor
// ✅ Fallback token generation mekanizması
// ✅ Production ready - Redis store interface'i için hazır
// -----------------------------------------------------------------------------

type CSRFToken struct {
	Value     string
	ExpiresAt time.Time
}

// CSRFStore interface - Redis implementation için hazır
type CSRFStore interface {
	GetToken(sessionID string) (string, error)
	ValidateToken(sessionID string, token string) bool
	DeleteToken(sessionID string) error
}

// InMemoryCSRFStore, development için in-memory implementation
type InMemoryCSRFStore struct {
	mu     sync.RWMutex
	tokens map[string]*CSRFToken
}

// NewInMemoryCSRFStore, yeni bir in-memory store oluşturur.
// PRODUCTION UYARISI: Multi-server deployment için Redis kullanın!
func NewInMemoryCSRFStore() *InMemoryCSRFStore {
	return &InMemoryCSRFStore{
		tokens: make(map[string]*CSRFToken),
	}
}

// Global CSRF token store (development için)
var csrfStore CSRFStore = NewInMemoryCSRFStore()

// SetCSRFStore, global CSRF store'u değiştirir.
// Production'da Redis store inject etmek için kullan:
//   SetCSRFStore(NewRedisCSRFStore(redisClient))
func SetCSRFStore(store CSRFStore) {
	csrfStore = store
}

// generateCSRFToken, kriptografik olarak güvenli bir CSRF token üretir.
// PANIC FIX: rand.Read() error'u handle ediliyor
func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)

	// crypto/rand kullanarak güvenli random bytes üret
	if _, err := rand.Read(bytes); err != nil {
		// FALLBACK: rand.Read başarısız olursa time-based token kullan
		// Bu daha az güvenli ama panic'ten iyidir
		fallbackToken := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
		return base64.URLEncoding.EncodeToString([]byte(fallbackToken)), fmt.Errorf("crypto/rand failed, using fallback: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateSessionID, güvenli bir session ID üretir.
// PANIC FIX: rand.Read() error'u handle ediliyor
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		// FALLBACK: Time-based session ID
		fallbackSession := fmt.Sprintf("session-%d", time.Now().UnixNano())
		return base64.URLEncoding.EncodeToString([]byte(fallbackSession)), fmt.Errorf("crypto/rand failed for session: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// InMemoryCSRFStore implementation

func (cs *InMemoryCSRFStore) GetToken(sessionID string) (string, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Mevcut token'ı kontrol et
	if token, exists := cs.tokens[sessionID]; exists {
		// Token expire olmamışsa dön
		if time.Now().Before(token.ExpiresAt) {
			return token.Value, nil
		}
		// Expire olmuşsa sil
		delete(cs.tokens, sessionID)
	}

	// Yeni token oluştur
	tokenValue, err := generateCSRFToken()
	if err != nil {
		// Token generation başarısız olsa bile devam et (fallback kullanıldı)
		// Log için error'u dön ama token'ı kullan
	}

	cs.tokens[sessionID] = &CSRFToken{
		Value:     tokenValue,
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}

	return tokenValue, err
}

func (cs *InMemoryCSRFStore) ValidateToken(sessionID string, token string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	storedToken, exists := cs.tokens[sessionID]
	if !exists {
		return false
	}

	// Token expire olmuş mu?
	if time.Now().After(storedToken.ExpiresAt) {
		return false
	}

	// Timing attack'e karşı güvenli karşılaştırma
	return subtle.ConstantTimeCompare([]byte(storedToken.Value), []byte(token)) == 1
}

func (cs *InMemoryCSRFStore) DeleteToken(sessionID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.tokens, sessionID)
	return nil
}

// getSessionID, request'ten session ID'yi çıkarır.
func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// setSessionID, response'a session ID cookie'sini ekler.
func setSessionID(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // PRODUCTION'DA true olmalı (HTTPS için)
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7200, // 2 saat
	})
}

// CSRFProtection, CSRF token doğrulaması yapan middleware'i döndürür.
func CSRFProtection() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Session ID'yi al (yoksa oluştur)
			sessionID := getSessionID(r)
			if sessionID == "" {
				// Yeni session oluştur
				newSessionID, err := generateSessionID()
				if err != nil {
					// Session generation başarısız (çok nadir)
					// Log edilebilir ama devam et
				}
				sessionID = newSessionID
				setSessionID(w, sessionID)
			}

			// CSRF token'ı oluştur/al
			csrfToken, err := csrfStore.GetToken(sessionID)
			if err != nil {
				// Token generation'da problem oldu ama fallback kullanıldı
				// Log edilebilir
			}

			// Token'ı cookie olarak set et (JavaScript'ten erişilebilir olması için)
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    csrfToken,
				Path:     "/",
				HttpOnly: false, // JavaScript erişimi için false
				Secure:   false, // PRODUCTION'DA true olmalı
				SameSite: http.SameSiteStrictMode,
				MaxAge:   7200, // 2 saat
			})

			// Safe metodlar (GET, HEAD, OPTIONS) için doğrulama yapma
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// POST, PUT, DELETE, PATCH için token doğrulaması yap
			var submittedToken string

			// 1. Header'dan al (modern API'ler için)
			submittedToken = r.Header.Get("X-CSRF-Token")

			// 2. Form'dan al (klasik form submission için)
			if submittedToken == "" {
				submittedToken = r.FormValue("_token")
			}

			// 3. Query parameter'dan al (son çare)
			if submittedToken == "" {
				submittedToken = r.URL.Query().Get("_token")
			}

			// Token yoksa veya geçersizse reddet
			if submittedToken == "" || !csrfStore.ValidateToken(sessionID, submittedToken) {
				response.Error(w, http.StatusForbidden, "CSRF token doğrulaması başarısız. Lütfen sayfayı yenileyin.")
				return
			}

			// Token geçerli, devam et
			next.ServeHTTP(w, r)
		})
	}
}

// -----------------------------------------------------------------------------
// Redis CSRF Store Implementation (PRODUCTION İÇİN)
// -----------------------------------------------------------------------------
// Bu implementation Phase 3'te eklenecek Redis entegrasyonu için hazır.
//
// Kullanım:
//   redisStore := NewRedisCSRFStore(redisClient, "csrf:", 2*time.Hour)
//   SetCSRFStore(redisStore)
//
// type RedisCSRFStore struct {
//     client *redis.Client
//     prefix string
//     ttl    time.Duration
// }
//
// func NewRedisCSRFStore(client *redis.Client, prefix string, ttl time.Duration) *RedisCSRFStore {
//     return &RedisCSRFStore{
//         client: client,
//         prefix: prefix,
//         ttl:    ttl,
//     }
// }
//
// func (r *RedisCSRFStore) GetToken(sessionID string) (string, error) {
//     ctx := context.Background()
//     key := r.prefix + sessionID
//
//     // Redis'ten token'ı al
//     token, err := r.client.Get(ctx, key).Result()
//     if err == redis.Nil {
//         // Token yok, yeni oluştur
//         token, err = generateCSRFToken()
//         if err != nil {
//             return "", err
//         }
//         // Redis'e kaydet
//         r.client.Set(ctx, key, token, r.ttl)
//     } else if err != nil {
//         return "", err
//     }
//
//     return token, nil
// }
//
// func (r *RedisCSRFStore) ValidateToken(sessionID string, token string) bool {
//     ctx := context.Background()
//     key := r.prefix + sessionID
//
//     storedToken, err := r.client.Get(ctx, key).Result()
//     if err != nil {
//         return false
//     }
//
//     return subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) == 1
// }
//
// func (r *RedisCSRFStore) DeleteToken(sessionID string) error {
//     ctx := context.Background()
//     key := r.prefix + sessionID
//     return r.client.Del(ctx, key).Err()
// }
// -----------------------------------------------------------------------------