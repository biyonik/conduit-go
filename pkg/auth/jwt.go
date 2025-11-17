// -----------------------------------------------------------------------------
// JWT (JSON Web Token) Package
// -----------------------------------------------------------------------------
// Bu dosya, JWT token'larının oluşturulması, parse edilmesi ve doğrulanması
// için fonksiyonlar sağlar.
//
// JWT nedir?
// JSON Web Token, kullanıcı authentication'ı için kullanılan bir standarttır.
// Stateless olduğu için API authentication'da çok popülerdir.
//
// JWT Yapısı:
// Header.Payload.Signature
// eyJhbGc...eyJ1c2V...SflKxw
//
// JWT Avantajları:
// - Stateless (sunucu tarafında session tutmaya gerek yok)
// - Cross-domain çalışır (CORS friendly)
// - Mobile app'ler için ideal
// - Microservice'ler arası auth için kullanılabilir
//
// JWT Dezavantajları:
// - Revoke etmek zor (expire olmadan geçersiz kılamazsınız)
// - Payload boyutu büyük (her istekte gönderilir)
// - XSS saldırılarına karşı hassas (localStorage kullanımında)
//
// Güvenlik Best Practices:
// 1. Secret key'i environment variable'da tutun (asla kodda!)
// 2. HTTPS kullanın (token'ı plain text göndermeyin)
// 3. Short expiration time kullanın (1 saat)
// 4. Refresh token mekanizması ekleyin
// 5. Token'ı localStorage yerine httpOnly cookie'de saklayın
// -----------------------------------------------------------------------------

package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims, JWT token'ın payload'ında taşınan bilgileri temsil eder.
//
// Standart Claims (jwt.RegisteredClaims):
//   - iss (issuer): Token'ı oluşturan
//   - sub (subject): Token'ın konusu (genellikle user ID)
//   - aud (audience): Token'ın hedef kitlesi
//   - exp (expiration): Token'ın geçerlilik süresi
//   - nbf (not before): Token'ın geçerli olmaya başlama zamanı
//   - iat (issued at): Token'ın oluşturulma zamanı
//   - jti (JWT ID): Token'ın benzersiz ID'si
//
// Custom Claims:
//   - UserID: Kullanıcı ID'si (veritabanından user çekmek için)
//   - Email: Kullanıcı email'i
//   - Role: Kullanıcı rolü (authorization için)
type JWTClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTConfig, JWT token oluşturma ve doğrulama ayarlarını içerir.
type JWTConfig struct {
	Secret           string        // Token imzalama için secret key
	Issuer           string        // Token issuer (genellikle app adı)
	ExpirationTime   time.Duration // Access token geçerlilik süresi
	RefreshExpiresIn time.Duration // Refresh token geçerlilik süresi
}

// DefaultJWTConfig, varsayılan JWT ayarlarını döndürür.
//
// Production'da bu ayarlar environment variable'lardan okunmalıdır!
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		Secret:           "your-super-secret-jwt-key-change-this-in-production",
		Issuer:           "conduit-go",
		ExpirationTime:   1 * time.Hour,      // 1 saat
		RefreshExpiresIn: 7 * 24 * time.Hour, // 7 gün
	}
}

// GenerateToken, kullanıcı bilgileri ile yeni bir JWT access token oluşturur.
//
// Parametreler:
//   - userID: Kullanıcı ID'si
//   - email: Kullanıcı email'i
//   - role: Kullanıcı rolü (admin, user, editor, vb.)
//   - config: JWT configuration (nil ise default kullanılır)
//
// Döndürür:
//   - string: Oluşturulan JWT token
//   - error: Token oluşturma başarısız olursa
//
// Örnek:
//
//	token, err := auth.GenerateToken(123, "user@example.com", "user", nil)
//	// token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
//
// Frontend Kullanımı:
//
//	fetch('/api/profile', {
//	    headers: {
//	        'Authorization': 'Bearer ' + token
//	    }
//	})
func GenerateToken(userID int64, email, role string, config *JWTConfig) (string, error) {
	if config == nil {
		config = DefaultJWTConfig()
	}

	// Şimdiki zaman
	now := time.Now()

	// Claims oluştur
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.Issuer,
			Subject:   email,
			ExpiresAt: jwt.NewNumericDate(now.Add(config.ExpirationTime)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Token oluştur (HS256 algoritması)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Token'ı imzala
	tokenString, err := token.SignedString([]byte(config.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken, uzun ömürlü bir refresh token oluşturur.
//
// Refresh Token nedir?
// Access token'lar kısa ömürlüdür (1 saat). Kullanıcıyı her saat login yapmaya
// zorlamak kötü UX'tir. Refresh token, yeni access token almak için kullanılır.
//
// Akış:
// 1. Kullanıcı login olur
// 2. Hem access token hem refresh token alır
// 3. Access token expire olunca refresh token ile yeni access token alır
// 4. Refresh token da expire olunca tekrar login olması gerekir
//
// Güvenlik Notu:
// - Refresh token'ı httpOnly cookie'de saklayın (XSS koruması)
// - Refresh token kullanıldığında yeni refresh token oluşturun (rotation)
// - Şüpheli aktivite varsa tüm refresh token'ları revoke edin
func GenerateRefreshToken(userID int64, email string, config *JWTConfig) (string, error) {
	if config == nil {
		config = DefaultJWTConfig()
	}

	now := time.Now()

	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   "refresh", // Refresh token'ı ayırt etmek için
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.Issuer,
			Subject:   email,
			ExpiresAt: jwt.NewNumericDate(now.Add(config.RefreshExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken, JWT token string'ini parse eder ve claims'leri döndürür.
//
// Parametre:
//   - tokenString: Parse edilecek JWT token
//   - config: JWT configuration (nil ise default kullanılır)
//
// Döndürür:
//   - *JWTClaims: Token'dan extract edilen claims
//   - error: Parse veya doğrulama başarısız olursa
//
// Örnek:
//
//	claims, err := auth.ParseToken(tokenFromHeader, nil)
//	if err != nil {
//	    return errors.New("invalid token")
//	}
//	userID := claims.UserID
//
// Hata Durumları:
// - Token format hatası
// - İmza doğrulama hatası (tampered token)
// - Expire olmuş token
// - Not before zamanı henüz gelmemiş
func ParseToken(tokenString string, config *JWTConfig) (*JWTClaims, error) {
	if config == nil {
		config = DefaultJWTConfig()
	}

	// Token'ı parse et
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// İmza algoritmasını kontrol et (algorithm confusion attack koruması)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Claims'leri extract et
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ValidateToken, JWT token'ın geçerli olup olmadığını kontrol eder.
//
// Bu fonksiyon ParseToken'ın basitleştirilmiş halidir. Sadece token'ın
// geçerli olup olmadığını döndürür, claims'lere erişim sağlamaz.
//
// Parametre:
//   - tokenString: Kontrol edilecek JWT token
//   - config: JWT configuration (nil ise default kullanılır)
//
// Döndürür:
//   - bool: Token geçerliyse true
//
// Örnek:
//
//	if !auth.ValidateToken(token, nil) {
//	    return errors.New("invalid or expired token")
//	}
func ValidateToken(tokenString string, config *JWTConfig) bool {
	_, err := ParseToken(tokenString, config)
	return err == nil
}

// ExtractTokenFromHeader, HTTP Authorization header'ından JWT token'ı çıkarır.
//
// Header formatı:
//
//	Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
//
// Parametre:
//   - authHeader: Authorization header değeri
//
// Döndürür:
//   - string: Extract edilen token (boş ise header yok veya format hatalı)
//
// Örnek:
//
//	token := auth.ExtractTokenFromHeader(r.Header.Get("Authorization"))
//	if token == "" {
//	    return errors.New("missing authorization header")
//	}
func ExtractTokenFromHeader(authHeader string) string {
	// "Bearer " prefix'ini kontrol et
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}
