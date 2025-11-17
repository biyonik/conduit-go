// -----------------------------------------------------------------------------
// CSRF (Cross-Site Request Forgery) Protection Middleware
// -----------------------------------------------------------------------------
// Bu middleware, POST, PUT, DELETE, PATCH gibi state-changing HTTP metodları
// için CSRF token doğrulaması yapar. Laravel'deki VerifyCsrfToken middleware'ine
// benzer şekilde çalışır.
//
// CSRF saldırısı nedir?
// Kullanıcının authentication cookie'si hala geçerliyken, kötü niyetli bir site
// kullanıcı adına istek gönderebilir. CSRF token bunu önler çünkü her form
// submission'ı için unique bir token gerekir.
//
// Nasıl kullanılır?
// 1. Session'da CSRF token oluştur
// 2. Form'a hidden input olarak ekle: <input type="hidden" name="_token" value="...">
// 3. Bu middleware her POST/PUT/DELETE isteğinde token'ı doğrular
//
// Güvenlik Notu:
// - Token'lar en az 32 byte uzunluğunda ve kriptografik olarak güvenli olmalı
// - Token'lar session'a bağlı olmalı (her kullanıcı için farklı)
// - Token'lar expire olmalı (örn: 2 saat)
// -----------------------------------------------------------------------------

package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// CSRFToken, bir CSRF token'ı ve onun expire zamanını tutar.
type CSRFToken struct {
	Value     string
	ExpiresAt time.Time
}

// CSRFStore, CSRF token'larını session bazlı olarak saklayan basit bir in-memory store'dur.
//
// PRODUCTION NOTU:
// Production'da bu store yerine Redis veya veritabanı kullanılmalıdır.
// Çünkü multiple server instance'larında her server'ın kendi memory'si vardır.
type CSRFStore struct {
	mu     sync.RWMutex
	tokens map[string]*CSRFToken // sessionID -> token mapping
}

// Global CSRF token store
var csrfStore = &CSRFStore{
	tokens: make(map[string]*CSRFToken),
}

// generateCSRFToken, kriptografik olarak güvenli bir CSRF token üretir.
//
// Döndürür:
//   - string: Base64 encoded 32-byte random token
//   - error: Random byte üretimi başarısız olursa
//
// Güvenlik Notu:
// crypto/rand kullanılıyor (math/rand DEĞİL!). Bu, tahmin edilemez tokenlar üretir.
func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// getSessionID, request'ten session ID'yi çıkarır.
//
// GEÇICÎ IMPLEMENTASYON:
// Şu an sadece cookie'den okuyor. İleride session package eklendiğinde
// bu fonksiyon session.GetID() gibi bir şey çağıracak.
//
// Parametre:
//   - r: HTTP request
//
// Döndürür:
//   - string: Session ID (cookie'den veya boş string)
func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// setSessionID, response'a session ID cookie'sini ekler.
//
// GEÇICÎ IMPLEMENTASYON:
// İleride session package bu işi yapacak.
//
// Parametreler:
//   - w: HTTP response writer
//   - sessionID: Set edilecek session ID
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

// GetToken, session için CSRF token'ı döndürür. Yoksa yeni bir tane oluşturur.
//
// Parametre:
//   - sessionID: Session ID
//
// Döndürür:
//   - string: CSRF token
//
// Güvenlik Notu:
// Token'lar 2 saat sonra expire oluyor. Production'da bu süre config'den okunmalı.
func (cs *CSRFStore) GetToken(sessionID string) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Mevcut token'ı kontrol et
	if token, exists := cs.tokens[sessionID]; exists {
		// Token expire olmamışsa dön
		if time.Now().Before(token.ExpiresAt) {
			return token.Value
		}
		// Expire olmuşsa sil
		delete(cs.tokens, sessionID)
	}

	// Yeni token oluştur
	tokenValue, err := generateCSRFToken()
	if err != nil {
		// Fallback: basit bir token (PRODUCTION'DA ASLA KULLANILMAMALI!)
		tokenValue = base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}

	cs.tokens[sessionID] = &CSRFToken{
		Value:     tokenValue,
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}

	return tokenValue
}

// ValidateToken, verilen token'ın session için geçerli olup olmadığını kontrol eder.
//
// Parametreler:
//   - sessionID: Session ID
//   - token: Doğrulanacak token
//
// Döndürür:
//   - bool: Token geçerliyse true
//
// Güvenlik Notu:
// Token karşılaştırması subtle.ConstantTimeCompare kullanıyor.
// Bu, timing attack'lere karşı koruma sağlar.
func (cs *CSRFStore) ValidateToken(sessionID string, token string) bool {
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

// CSRFProtection, CSRF token doğrulaması yapan middleware'i döndürür.
//
// Döndürür:
//   - Middleware: CSRF protection middleware
//
// Nasıl Çalışır:
// 1. GET/HEAD/OPTIONS istekleri için sadece token cookie'sini set eder
// 2. POST/PUT/DELETE/PATCH için token doğrulaması yapar
// 3. Token geçersizse 403 Forbidden döner
//
// Kullanım:
//
//	r.Use(middleware.CSRFProtection())
//
// Frontend'de token kullanımı:
//
//	// Cookie'den token'ı oku
//	const token = document.cookie.split('; ')
//	    .find(row => row.startsWith('csrf_token='))
//	    ?.split('=')[1];
//
//	// POST isteğinde gönder
//	fetch('/api/users', {
//	    method: 'POST',
//	    headers: {
//	        'X-CSRF-Token': token,
//	    },
//	    body: JSON.stringify({...}),
//	});
//
// Güvenlik Notu:
// Bu middleware mutlaka session middleware'den SONRA çalışmalıdır.
func CSRFProtection() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Session ID'yi al (yoksa oluştur)
			sessionID := getSessionID(r)
			if sessionID == "" {
				// Yeni session oluştur (geçici: random string)
				sessionBytes := make([]byte, 16)
				rand.Read(sessionBytes)
				sessionID = base64.URLEncoding.EncodeToString(sessionBytes)
				setSessionID(w, sessionID)
			}

			// CSRF token'ı oluştur/al
			csrfToken := csrfStore.GetToken(sessionID)

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
			// Token'ı üç yerden alabilir: header, form, query
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
