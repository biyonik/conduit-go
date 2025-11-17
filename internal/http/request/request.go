// Package request, HTTP isteklerinin daha okunabilir, daha yönetilebilir
// ve framework seviyesinde (Laravel, Symfony gibi) hissettiren bir yapı
// ile ele alınmasını sağlar.
package request

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
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
