// -----------------------------------------------------------------------------
// Authentication Middleware
// -----------------------------------------------------------------------------
// Bu middleware, JWT token doğrulaması yaparak kullanıcının authenticate
// olup olmadığını kontrol eder.
//
// Laravel'deki auth middleware'ine benzer şekilde çalışır.
// -----------------------------------------------------------------------------

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/pkg/auth"
)

// Auth, JWT authentication middleware'ini döndürür.
//
// Bu middleware:
// 1. Authorization header'ından JWT token'ı çıkarır
// 2. Token'ı doğrular
// 3. Token geçerliyse user bilgisini context'e ekler
// 4. Token geçersizse 401 Unauthorized döner
//
// Kullanım:
//
//	// Global auth (tüm route'lar için)
//	r.Use(middleware.Auth())
//
//	// Belirli route'lar için
//	profileGroup := r.Group("/api/profile")
//	profileGroup.Use(middleware.Auth())
//
//	// Tek bir route için
//	r.GET("/api/secret", SecretHandler).Middleware(middleware.Auth())
//
// Frontend Kullanımı:
//
//	fetch('/api/profile', {
//	    headers: {
//	        'Authorization': 'Bearer ' + accessToken
//	    }
//	})
//
// Context'e Eklenen Değerler:
// - "user": auth.User interface implementasyonu
// - "user_id": int64 (kullanıcı ID'si)
// - "user_email": string (kullanıcı email'i)
// - "user_role": string (kullanıcı rolü)
func Auth() Middleware {
	return AuthWithConfig(nil)
}

// AuthWithConfig, özel JWT config ile authentication middleware'i döndürür.
//
// Parametre:
//   - config: JWT configuration (nil ise default kullanılır)
//
// Örnek:
//
//	customConfig := &auth.JWTConfig{
//	    Secret: os.Getenv("JWT_SECRET"),
//	    ExpirationTime: 2 * time.Hour,
//	}
//	r.Use(middleware.AuthWithConfig(customConfig))
func AuthWithConfig(config *auth.JWTConfig) Middleware {
	if config == nil {
		config = auth.DefaultJWTConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Authorization header'ı al
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "Authorization header gerekli")
				return
			}

			// 2. Bearer token'ı extract et
			token := extractBearerToken(authHeader)
			if token == "" {
				response.Error(w, http.StatusUnauthorized, "Geçersiz Authorization format (Bearer token bekleniyor)")
				return
			}

			// 3. Token'ı parse et ve doğrula
			claims, err := auth.ParseToken(token, config)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "Geçersiz veya süresi dolmuş token")
				return
			}

			// 4. Refresh token ile normal endpoint'lere erişmeye izin verme
			if claims.Role == "refresh" {
				response.Error(w, http.StatusUnauthorized, "Refresh token bu endpoint için kullanılamaz")
				return
			}

			// 5. User bilgisini context'e ekle
			user := &auth.AuthenticatedUser{
				ID:    claims.UserID,
				Email: claims.Email,
				Role:  claims.Role,
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "user", user)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "user_email", claims.Email)
			ctx = context.WithValue(ctx, "user_role", claims.Role)

			// 6. Request'i güncellenmiş context ile devam ettir
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth, opsiyonel authentication middleware'idir.
// Token varsa user bilgisini context'e ekler, yoksa da next handler'a geçer.
//
// Kullanım Senaryosu:
// Bazı endpoint'ler hem authenticated hem unauthenticated user'lar için
// çalışabilir, ama davranış farklıdır.
//
// Örnek:
// - Genel blog listesi herkese açık
// - Ama authenticated user kendi drafts'larını da görür
//
// Kullanım:
//
//	r.GET("/api/posts", PostListHandler).Middleware(middleware.OptionalAuth())
//
//	// Handler içinde:
//	func PostListHandler(w http.ResponseWriter, r *Request) {
//	    user := r.Context().Value("user")
//	    if user != nil {
//	        // Authenticated user, drafts da göster
//	    } else {
//	        // Guest user, sadece published posts
//	    }
//	}
func OptionalAuth() Middleware {
	return OptionalAuthWithConfig(nil)
}

// OptionalAuthWithConfig, özel config ile opsiyonel auth middleware'i döndürür.
func OptionalAuthWithConfig(config *auth.JWTConfig) Middleware {
	if config == nil {
		config = auth.DefaultJWTConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authorization header var mı?
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Token yok, guest olarak devam et
				next.ServeHTTP(w, r)
				return
			}

			// Token varsa parse et
			token := extractBearerToken(authHeader)
			if token == "" {
				// Format hatalı, guest olarak devam et
				next.ServeHTTP(w, r)
				return
			}

			// Token'ı doğrula
			claims, err := auth.ParseToken(token, config)
			if err != nil {
				// Token geçersiz, guest olarak devam et
				next.ServeHTTP(w, r)
				return
			}

			// Token geçerli, user bilgisini context'e ekle
			user := &auth.AuthenticatedUser{
				ID:    claims.UserID,
				Email: claims.Email,
				Role:  claims.Role,
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "user", user)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "user_email", claims.Email)
			ctx = context.WithValue(ctx, "user_role", claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractBearerToken, Authorization header'ından Bearer token'ı çıkarır.
//
// Header formatı: "Bearer eyJhbGc..."
//
// Parametre:
//   - authHeader: Authorization header değeri
//
// Döndürür:
//   - string: Extract edilen token (boş ise format hatalı)
func extractBearerToken(authHeader string) string {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		return ""
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// GetAuthUser, context'ten authenticated user'ı döndürür.
// Middleware içinde veya handler'larda kullanılabilir.
//
// Parametre:
//   - ctx: Request context
//
// Döndürür:
//   - auth.User: Authenticated user (nil ise user authenticate değil)
//
// Örnek:
//
//	func SomeHandler(w http.ResponseWriter, r *Request) {
//	    user := middleware.GetAuthUser(r.Context())
//	    if user == nil {
//	        // User not authenticated
//	        return
//	    }
//	    userID := user.GetID()
//	}
func GetAuthUser(ctx context.Context) auth.User {
	user := ctx.Value("user")
	if user == nil {
		return nil
	}

	authUser, ok := user.(auth.User)
	if !ok {
		return nil
	}

	return authUser
}

// GetUserID, context'ten user ID'yi döndürür.
// GetAuthUser'a göre daha hızlıdır (type assertion gerekmez).
func GetUserID(ctx context.Context) int64 {
	userID := ctx.Value("user_id")
	if userID == nil {
		return 0
	}

	id, ok := userID.(int64)
	if !ok {
		return 0
	}

	return id
}

// GetUserEmail, context'ten user email'ini döndürür.
func GetUserEmail(ctx context.Context) string {
	email := ctx.Value("user_email")
	if email == nil {
		return ""
	}

	str, ok := email.(string)
	if !ok {
		return ""
	}

	return str
}

// GetUserRole, context'ten user role'ünü döndürür.
func GetUserRole(ctx context.Context) string {
	role := ctx.Value("user_role")
	if role == nil {
		return ""
	}

	str, ok := role.(string)
	if !ok {
		return ""
	}

	return str
}
