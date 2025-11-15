// Package request, HTTP isteklerinin daha okunabilir, daha yönetilebilir
// ve framework seviyesinde (Laravel, Symfony gibi) hissettiren bir yapı
// ile ele alınmasını sağlamak amacıyla tasarlanmış küçük ve kullanışlı
// bir yardımcı pakettir. Bu paket sayesinde, gelen istek nesnesi ile
// ilgili en sık kullanılan işlemler; içerik tipinin JSON olup olmadığını
// kontrol etmek, Bearer Token ayrıştırmak, query parametresi okumak ve
// route parametrelerine erişmek gibi fonksiyonlar üzerinden temiz ve
// anlaşılır bir biçimde yapılabilir.
//
// Modern web uygulamalarında, request doğrulama ve işleme adımları
// oldukça kritik olduğundan, bu paketin sağladığı basit ama etkili
// soyutlama (abstraction), hem yazılım mimarisini sadeleştirir hem de
// tekrarlanan kodların önüne geçer. Aşağıdaki sınıf ve fonksiyonlar,
// profesyonel geliştiricilerin ihtiyaç duyacağı şekilde detaylı
// açıklamalarla zenginleştirilmiştir.
package request

import (
	"net/http"
	"strings"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// RouteParamsKey, Go context içinde route parametrelerini güvenli bir şekilde
// saklamak için kullanılan özel anahtar tipidir.
// string yerine struct{} kullanmak key çakışmalarını önler.
type RequestParamsKeyType struct{}

// requestParamsKey global key instance
var RequestParamsKey = RequestParamsKeyType{}

// Request yapısı, http.Request yapısının üzerine inşa edilmiş bir sarmalayıcıdır
// (wrapper). Bu modelin amacı, standart http.Request nesnesine ek
// fonksiyonellik kazandırmak ve API geliştirme süreçlerinde daha fazla
// kolaylık sunmaktır.
//
// Bu yapı sayesinde, gelen istek üzerinde sık yapılan işlemler daha okunabilir
// ve daha kısa kodlarla gerçekleştirilebilir.
type Request struct {
	*http.Request
}

// New, alınan *http.Request nesnesini bizim Request modelimize dönüştüren
// bir yapıcı (constructor) fonksiyondur. Bu fonksiyon sayesinde dışarıdan
// her zaman standart http.Request değil, geliştirilmiş Request yapısı ile
// işlem yapılması sağlanarak ek fonksiyonların kullanılmasına imkân tanınır.
//
// Parametre:
//   - r: Orijinal HTTP istek nesnesi
//
// Döndürür:
//   - *Request: Geliştirilmiş Request yapısı
func New(r *http.Request) *Request {
	return &Request{Request: r}
}

// IsJSON, gelen HTTP isteğinin Content-Type başlığında "application/json"
// içerip içermediğini kontrol eden bir fonksiyondur. API geliştirme
// süreçlerinde, gövde içeriğinin JSON olup olmadığını bilmek sıklıkla
// doğrulama akışlarının ilk adımıdır.
//
// Döndürür:
//   - bool: İçerik tipi JSON ise true, değilse false.
func (r *Request) IsJSON() bool {
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(contentType, "application/json")
}

// BearerToken, Authorization başlığından Bearer Token değerini güvenli ve
// kontrollü bir biçimde ayrıştırmak için kullanılan fonksiyondur.
//
// HTTP Authorization header formatı örneği:
//
//	Authorization: Bearer <TOKEN>
//
// Fonksiyon Akışı:
//  1. Authorization başlığı alınır.
//  2. Başlık boş ise token olmadığı varsayılır ve boş string döndürülür.
//  3. Başlık boşluk karakterlerine göre bölünür.
//  4. Format beklendiği gibi değilse boş string döner.
//  5. Her şey doğruysa token kısmı döndürülür.
//
// Döndürür:
//   - string: Geçerli bearer token ya da boş string.
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
// üzerinden değer okumayı kolaylaştıran fonksiyondur.
//
// Örnek URL:
//
//	/users?page=2&sort=name
//
// Fonksiyon Akışı:
//  1. İstenen query anahtarı URL üzerinden okunur.
//  2. Değer yoksa, fonksiyon geliştirici tarafından verilen
//     defaultValue değerini döndürür.
//  3. Değer bulunursa direkt olarak döndürülür.
//
// Parametreler:
//   - key: Okunmak istenen query parametresi.
//   - defaultValue: Parametre bulunamazsa dönecek varsayılan değer.
//
// Döndürür:
//   - string: Query parametre değeri veya varsayılan değer.
func (r *Request) Query(key string, defaultValue string) string {
	vals, exists := r.URL.Query()[key]
	if !exists || len(vals) == 0 {
		return defaultValue
	}
	return vals[0]
}

// RouteParam, route parametrelerini almak için kullanılan fonksiyondur.
// Bu fonksiyon, router tarafından context'e yerleştirilen parametreleri
// güvenli bir şekilde okur.
//
// Örnek kullanım: /users/{id} -> RouteParam("id")
//
// Parametre:
//   - key: İstenen route parametre anahtarı.
//
// Döndürür:
//   - string: Parametre değeri veya boş string.
func (r *Request) RouteParam(key string) string {
	params, ok := r.Context().Value(RequestParamsKey).(map[string]string)
	if !ok {
		return ""
	}
	return params[key]
}
