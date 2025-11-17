// -----------------------------------------------------------------------------
// Auth Guard System
// -----------------------------------------------------------------------------
// Bu dosya, farklı authentication stratejilerini (JWT, Session, API Key)
// yönetmek için Guard pattern'ini implement eder.
//
// Laravel'deki auth()->guard('web') ve auth()->guard('api') konseptine benzer.
//
// Guard nedir?
// Guard, authentication mekanizmasını soyutlar. Böylece aynı uygulamada
// birden fazla auth stratejisi kullanabilirsiniz:
// - Web UI için session-based auth
// - API için JWT-based auth
// - Webhook'lar için API key auth
//
// Kullanım Örneği:
//   jwtGuard := auth.NewJWTGuard(config)
//   user, err := jwtGuard.Authenticate(token)
// -----------------------------------------------------------------------------

package auth

import (
	"errors"
)

// User, authentication'dan dönen kullanıcı bilgilerini temsil eder.
// Bu interface, farklı user model'larının auth sistemi ile çalışmasını sağlar.
type User interface {
	GetID() int64
	GetEmail() string
	GetRole() string
}

// Guard, authentication işlemlerini tanımlayan arayüzdür.
// Her guard (JWT, Session, vb.) bu interface'i implement eder.
type Guard interface {
	// Authenticate, verilen credential ile kullanıcıyı doğrular
	Authenticate(credential string) (User, error)

	// User, mevcut authenticated user'ı döndürür (session/context'ten)
	User() (User, error)

	// Check, kullanıcının authenticate olup olmadığını kontrol eder
	Check() bool

	// Logout, kullanıcıyı logout yapar
	Logout() error
}

// JWTGuard, JWT token-based authentication implementasyonudur.
type JWTGuard struct {
	config      *JWTConfig
	currentUser User // Mevcut request'teki authenticated user (cache)
}

// NewJWTGuard, yeni bir JWTGuard instance'ı oluşturur.
func NewJWTGuard(config *JWTConfig) *JWTGuard {
	if config == nil {
		config = DefaultJWTConfig()
	}
	return &JWTGuard{
		config: config,
	}
}

// Authenticate, JWT token'ı doğrular ve user bilgilerini döndürür.
//
// NOT: Bu implementasyon sadece token'ı parse eder. Gerçek bir user
// nesnesi döndürmez çünkü database'e erişimi yok. Controller'da
// claims.UserID ile database'den user çekilmelidir.
//
// İleride bu metod şu şekilde geliştirilecek:
//
//	user, err := jwtGuard.Authenticate(token, userRepository)
func (g *JWTGuard) Authenticate(tokenString string) (User, error) {
	// Token'ı parse et
	claims, err := ParseToken(tokenString, g.config)
	if err != nil {
		return nil, errors.New("invalid or expired token")
	}

	// Claims'den basit bir user objesi oluştur
	// (Gerçek implementasyonda database'den user çekilir)
	user := &AuthenticatedUser{
		ID:    claims.UserID,
		Email: claims.Email,
		Role:  claims.Role,
	}

	// Cache'e kaydet
	g.currentUser = user

	return user, nil
}

// User, mevcut authenticated user'ı döndürür.
func (g *JWTGuard) User() (User, error) {
	if g.currentUser == nil {
		return nil, errors.New("user not authenticated")
	}
	return g.currentUser, nil
}

// Check, kullanıcının authenticate olup olmadığını kontrol eder.
func (g *JWTGuard) Check() bool {
	return g.currentUser != nil
}

// Logout, JWT için anlamsızdır (stateless).
// Client tarafında token'ı silmek yeterlidir.
func (g *JWTGuard) Logout() error {
	g.currentUser = nil
	return nil
}

// AuthenticatedUser, guard'dan dönen basit user implementasyonudur.
type AuthenticatedUser struct {
	ID    int64
	Email string
	Role  string
}

func (u *AuthenticatedUser) GetID() int64 {
	return u.ID
}

func (u *AuthenticatedUser) GetEmail() string {
	return u.Email
}

func (u *AuthenticatedUser) GetRole() string {
	return u.Role
}

// SessionGuard, session-based authentication implementasyonudur.
// (Phase 3'te tam implementasyon gelecek)
type SessionGuard struct {
	// TODO: Session store eklenecek
}

// NewSessionGuard, yeni bir SessionGuard instance'ı oluşturur.
func NewSessionGuard() *SessionGuard {
	return &SessionGuard{}
}

func (g *SessionGuard) Authenticate(credential string) (User, error) {
	return nil, errors.New("not implemented yet")
}

func (g *SessionGuard) User() (User, error) {
	return nil, errors.New("not implemented yet")
}

func (g *SessionGuard) Check() bool {
	return false
}

func (g *SessionGuard) Logout() error {
	return errors.New("not implemented yet")
}
