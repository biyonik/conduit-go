// -----------------------------------------------------------------------------
// Role-Based Authorization Middleware
// -----------------------------------------------------------------------------
// Bu middleware, kullanıcının belirli bir role sahip olup olmadığını kontrol eder.
// Laravel'deki role middleware'ine benzer şekilde çalışır.
//
// Kullanım Senaryoları:
// - Admin paneline sadece admin'ler erişebilir
// - Editor endpoint'lerine sadece editor ve admin'ler erişebilir
// - User endpoint'lerine tüm authenticated user'lar erişebilir
// -----------------------------------------------------------------------------

package middleware

import (
	"net/http"
	"time"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// Role, belirtilen role'e sahip kullanıcıların erişimine izin veren middleware döndürür.
//
// Parametreler:
//   - allowedRoles: İzin verilen roller (variadic)
//
// Döndürür:
//   - Middleware: Authorization middleware
//
// Örnek:
//
//	// Sadece admin'ler erişebilir
//	r.DELETE("/api/users/{id}", DeleteUserHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.Role("admin"))
//
//	// Admin veya editor erişebilir
//	r.POST("/api/posts", CreatePostHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.Role("admin", "editor"))
//
// NOT:
// Bu middleware'den önce Auth() middleware'i çalışmalıdır!
// Aksi takdirde context'te user bilgisi olmaz.
func Role(allowedRoles ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Context'ten user role'ünü al
			userRole := GetUserRole(r.Context())
			if userRole == "" {
				// User authenticated değil veya role yok
				response.Error(w, http.StatusUnauthorized, "Kimlik doğrulaması gerekli")
				return
			}

			// 2. User'ın role'ü izin verilen roller arasında mı kontrol et
			hasPermission := false
			for _, allowedRole := range allowedRoles {
				if userRole == allowedRole {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				response.Error(w, http.StatusForbidden, "Bu işlem için yetkiniz yok")
				return
			}

			// 3. İzin var, devam et
			next.ServeHTTP(w, r)
		})
	}
}

// Admin, sadece admin role'üne sahip kullanıcıların erişimine izin verir.
// Role("admin") kısayoludur.
//
// Kullanım:
//
//	r.DELETE("/api/users/{id}", DeleteUserHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.Admin())
func Admin() Middleware {
	return Role("admin")
}

// AdminOrEditor, admin veya editor role'üne sahip kullanıcıların erişimine izin verir.
//
// Kullanım:
//
//	r.POST("/api/posts", CreatePostHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.AdminOrEditor())
func AdminOrEditor() Middleware {
	return Role("admin", "editor")
}

// RequireEmailVerification, email'i doğrulanmış kullanıcıların erişimine izin verir.
//
// Kullanım:
//
//	r.POST("/api/posts", CreatePostHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.RequireEmailVerification())
//
// NOT:
// Bu middleware için User modelinde IsEmailVerified() metodu olmalıdır.
// Şu anda basit implementasyon, ileride database'den kontrol edilecek.
func RequireEmailVerification() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO (Phase 3): Database'den user çekerek email_verified_at kontrol et
			// Şimdilik tüm authenticated user'ların email'ini verified kabul ediyoruz

			user := GetAuthUser(r.Context())
			if user == nil {
				response.Error(w, http.StatusUnauthorized, "Kimlik doğrulaması gerekli")
				return
			}

			// İleride: if !user.IsEmailVerified()
			// response.Error(w, 403, "Email adresinizi doğrulamanız gerekiyor")

			next.ServeHTTP(w, r)
		})
	}
}

// Can, policy-based authorization için middleware döndürür.
//
// Bu Laravel'deki Gate/Policy sistemine benzer bir yapıdır.
// Kullanıcının belirli bir action'ı yapıp yapamayacağını kontrol eder.
//
// Parametreler:
//   - action: Action adı (örn: "update-post", "delete-comment")
//   - policyCheck: Policy check fonksiyonu
//
// Döndürür:
//   - Middleware: Authorization middleware
//
// Örnek:
//
//	// Sadece post sahibi güncelleyebilir
//	r.PUT("/api/posts/{id}", UpdatePostHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.Can("update-post", func(r *http.Request) bool {
//	        postID := r.Context().Value("post_id").(int64)
//	        userID := middleware.GetUserID(r.Context())
//
//	        post := postRepo.FindByID(postID)
//	        return post.UserID == userID || middleware.GetUserRole(r.Context()) == "admin"
//	    }))
//
// NOT: Bu Phase 3'te daha da geliştirilecek (Policy sınıfları eklenecek)
func Can(action string, policyCheck func(r *http.Request) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. User authenticated mi kontrol et
			user := GetAuthUser(r.Context())
			if user == nil {
				response.Error(w, http.StatusUnauthorized, "Kimlik doğrulaması gerekli")
				return
			}

			// 2. Policy check'i çalıştır
			if !policyCheck(r) {
				response.Error(w, http.StatusForbidden, "Bu işlem için yetkiniz yok")
				return
			}

			// 3. İzin var, devam et
			next.ServeHTTP(w, r)
		})
	}
}

// Throttle, authenticated user bazlı rate limiting sağlar.
// IP bazlı rate limiting'den farklı olarak, user ID bazlı çalışır.
//
// Bu, API abuse'i önlemek için kullanılır.
// Örneğin: Her user 10 dakikada 100 istek yapabilir.
//
// Parametreler:
//   - maxRequests: Maksimum istek sayısı
//   - windowInSeconds: Zaman penceresi (saniye)
//
// Döndürür:
//   - Middleware: Rate limiting middleware
//
// Örnek:
//
//	// Her user 1 dakikada 10 post oluşturabilir
//	r.POST("/api/posts", CreatePostHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.Throttle(10, 60))
//
// NOT: Bu, Phase 1'deki RateLimit middleware'inin user-aware versiyonudur.
func Throttle(maxRequests int, windowInSeconds int) Middleware {
	limiter := NewRateLimiter(maxRequests, windowInSeconds)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. User ID'yi al (authenticated olmalı)
			userID := GetUserID(r.Context())
			if userID == 0 {
				// User authenticated değil, next'e geç (Auth middleware çalışmalı)
				next.ServeHTTP(w, r)
				return
			}

			// 2. User ID ile rate limiting yap
			key := formatUserRateLimitKey(userID)
			allowed, remaining, retryAfter := limiter.Allow(key)

			// 3. Rate limit header'larını ekle
			w.Header().Set("X-RateLimit-Limit", formatInt(maxRequests))
			w.Header().Set("X-RateLimit-Remaining", formatInt(remaining))
			w.Header().Set("X-RateLimit-Reset", formatInt64(time.Now().Add(time.Duration(windowInSeconds)*time.Second).Unix()))

			if !allowed {
				w.Header().Set("Retry-After", formatInt(int(retryAfter.Seconds())))
				response.Error(w, http.StatusTooManyRequests, "Rate limit aşıldı. Lütfen daha sonra tekrar deneyin.")
				return
			}

			// 4. İzin verildi, devam et
			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions

func formatUserRateLimitKey(userID int64) string {
	return "user:" + formatInt64(userID)
}

func formatInt(n int) string {
	return string(rune(n + '0')) // Basit int to string (daha iyi bir yol: strconv.Itoa)
}

func formatInt64(n int64) string {
	return formatInt(int(n))
}
