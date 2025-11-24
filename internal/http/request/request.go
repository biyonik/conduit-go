// Package request, HTTP isteklerinin daha okunabilir, daha yönetilebilir
// ve framework seviyesinde (Laravel, Symfony gibi) hissettiren bir yapı
// ile ele alınmasını sağlar.
package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/biyonik/conduit-go/pkg/auth"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// RouteParamsKey, Go context içinde route parametrelerini güvenli bir şekilde
// saklamak için kullanılan özel anahtar tipidir.
type RequestParamsKeyType struct{}

// requestParamsKey global key instance
var RequestParamsKey = RequestParamsKeyType{}

// Request yapısı, http.Request yapısının üzerine inşa edilmiş bir sarmalayıcıdır.
type Request struct {
	*http.Request
}

// New, alınan *http.Request nesnesini bizim Request modelimize dönüştüren
// bir yapıcı fonksiyondur.
func New(r *http.Request) *Request {
	return &Request{Request: r}
}

// IsJSON, gelen HTTP isteğinin Content-Type başlığında "application/json"
// içerip içermediğini kontrol eder.
func (r *Request) IsJSON() bool {
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/json")
}

// BearerToken, Authorization başlığından Bearer Token değerini güvenli ve
// kontrollü bir biçimde ayrıştırır.
func (r *Request) BearerToken() string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// Query, gelen HTTP isteğinin URL query parametrelerinden bir anahtar
// üzerinden değer okumayı kolaylaştırır.
func (r *Request) Query(key string, defaultValue string) string {
	vals, exists := r.URL.Query()[key]
	if !exists || len(vals) == 0 {
		return defaultValue
	}
	return vals[0]
}

// RouteParam, route parametrelerini almak için kullanılır.
func (r *Request) RouteParam(key string) string {
	params, ok := r.Context().Value(RequestParamsKey).(map[string]string)
	if !ok {
		return ""
	}
	return params[key]
}

// ParseJSON, request body'deki JSON'ı parse eder ve verilen struct'a doldurur.
//
// Parametre:
//   - dest: JSON'ın parse edileceği struct pointer
//
// Döndürür:
//   - error: Parse hatası varsa
//
// Örnek:
//
//	var reqData LoginRequest
//	if err := r.ParseJSON(&reqData); err != nil {
//	    return errors.New("invalid JSON")
//	}
//
// Güvenlik Notu:
// - Request body'yi limit'le (10MB varsayılan)
// - Malicious JSON attack'lere karşı koruma
func (r *Request) ParseJSON(dest interface{}) error {
	// Request body'yi oku (maksimum 10MB)
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return err
	}
	defer r.Body.Close()

	// JSON parse et
	if err := json.Unmarshal(body, dest); err != nil {
		return err
	}

	return nil
}

// GetIP, client'ın IP adresini döndürür.
// Reverse proxy arkasındaysa X-Forwarded-For header'ını kontrol eder.
//
// Döndürür:
//   - string: Client IP adresi
//
// Güvenlik Notu:
// X-Forwarded-For header'ı spoof edilebilir!
// Sadece güvenilir reverse proxy'lerden geliyorsa kullanın.
func (r *Request) GetIP() string {
	// X-Forwarded-For header'ı varsa kullan (reverse proxy arkasında)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// İlk IP'yi al (client IP)
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// X-Real-IP header'ı varsa kullan
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// RemoteAddr kullan (standart)
	ip := r.RemoteAddr
	// Port'u kaldır (:8080 gibi)
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// UserAgent, client'ın User-Agent header'ını döndürür.
func (r *Request) UserAgent() string {
	return r.Header.Get("User-Agent")
}

// Accepts, client'ın Accept header'ında belirtilen content type'ı
// kabul edip etmediğini kontrol eder.
//
// Parametre:
//   - contentType: Kontrol edilecek content type
//
// Döndürür:
//   - bool: Client bu content type'ı kabul ediyorsa true
//
// Örnek:
//
//	if r.Accepts("application/json") {
//	    return response.JSON(w, data)
//	}
func (r *Request) Accepts(contentType string) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, contentType) || strings.Contains(accept, "*/*")
}

// AuthUser retrieves the authenticated user from the request context.
//
// This method extracts the user that was set by the Auth middleware.
// It returns an error if the user is not authenticated or if the context
// value is not of the correct type.
//
// Returns:
//   - auth.User: The authenticated user
//   - error: Error if user is not authenticated
//
// Example:
//
//	func (c *ProfileController) Show(w http.ResponseWriter, r *Request) {
//	    user, err := r.AuthUser()
//	    if err != nil {
//	        response.Unauthorized(w, "")
//	        return
//	    }
//	    userID := user.GetID()
//	}
//
// This eliminates the repetitive pattern:
//
//	contextUser := r.Context().Value("user")
//	if contextUser == nil {
//	    conduitRes.Error(w, 401, "Unauthorized")
//	    return
//	}
//	authUser, ok := contextUser.(auth.User)
//	if !ok {
//	    conduitRes.Error(w, 401, "Unauthorized")
//	    return
//	}
func (r *Request) AuthUser() (auth.User, error) {
	contextUser := r.Context().Value("user")
	if contextUser == nil {
		return nil, errors.New("unauthorized: no user in context")
	}

	authUser, ok := contextUser.(auth.User)
	if !ok {
		return nil, errors.New("unauthorized: invalid user type")
	}

	return authUser, nil
}

// MustAuthUser retrieves the authenticated user from context, panicking if not found.
//
// This is useful in handlers that are guaranteed to have authentication middleware
// applied. Use with caution.
//
// Returns:
//   - auth.User: The authenticated user
//
// Panics:
//   - If user is not authenticated or type assertion fails
//
// Example:
//
//	func (c *ProfileController) Show(w http.ResponseWriter, r *Request) {
//	    user := r.MustAuthUser() // Panic if not authenticated
//	    userID := user.GetID()
//	}
func (r *Request) MustAuthUser() auth.User {
	user, err := r.AuthUser()
	if err != nil {
		panic(err)
	}
	return user
}

// AuthUserID retrieves the authenticated user's ID from context.
//
// This is a convenience method that's faster than AuthUser() when you only
// need the user ID.
//
// Returns:
//   - int64: User ID (0 if not authenticated)
//   - error: Error if user is not authenticated
//
// Example:
//
//	userID, err := r.AuthUserID()
//	if err != nil {
//	    response.Unauthorized(w, "")
//	    return
//	}
func (r *Request) AuthUserID() (int64, error) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		return 0, errors.New("unauthorized: no user_id in context")
	}

	id, ok := userID.(int64)
	if !ok {
		return 0, errors.New("unauthorized: invalid user_id type")
	}

	return id, nil
}

// AuthUserEmail retrieves the authenticated user's email from context.
//
// Returns:
//   - string: User email (empty if not authenticated)
//   - error: Error if user is not authenticated
//
// Example:
//
//	email, err := r.AuthUserEmail()
//	if err != nil {
//	    response.Unauthorized(w, "")
//	    return
//	}
func (r *Request) AuthUserEmail() (string, error) {
	email := r.Context().Value("user_email")
	if email == nil {
		return "", errors.New("unauthorized: no user_email in context")
	}

	str, ok := email.(string)
	if !ok {
		return "", errors.New("unauthorized: invalid user_email type")
	}

	return str, nil
}

// AuthUserRole retrieves the authenticated user's role from context.
//
// Returns:
//   - string: User role (empty if not authenticated)
//   - error: Error if user is not authenticated
//
// Example:
//
//	role, err := r.AuthUserRole()
//	if err != nil {
//	    response.Unauthorized(w, "")
//	    return
//	}
//	if role != "admin" {
//	    response.Forbidden(w, "")
//	    return
//	}
func (r *Request) AuthUserRole() (string, error) {
	role := r.Context().Value("user_role")
	if role == nil {
		return "", errors.New("unauthorized: no user_role in context")
	}

	str, ok := role.(string)
	if !ok {
		return "", errors.New("unauthorized: invalid user_role type")
	}

	return str, nil
}

// IsAuthenticated checks if the request has an authenticated user.
//
// This is a simple boolean check without error handling.
//
// Returns:
//   - bool: true if user is authenticated, false otherwise
//
// Example:
//
//	if !r.IsAuthenticated() {
//	    response.Unauthorized(w, "")
//	    return
//	}
func (r *Request) IsAuthenticated() bool {
	_, err := r.AuthUser()
	return err == nil
}
