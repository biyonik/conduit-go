// -----------------------------------------------------------------------------
// CORS Middleware
// -----------------------------------------------------------------------------
// Bu dosya, web uygulamalarında sıkça ihtiyaç duyulan CORS (Cross-Origin
// Resource Sharing) yönetimini sağlayan middleware'i içerir. Modern frontend
// uygulamaları ile backend servislerinin farklı domain'ler üzerinde çalıştığı
// senaryolarda, tarayıcı güvenlik politikaları sebebiyle CORS ayarları kritik
// bir rol oynar.
//
// Laravel, Symfony gibi framework'lerde otomatik olarak sağlanan bu özellik,
// burada Go'nun net/http altyapısı üzerine temiz, sade ve esnek bir şekilde
// uygulanmıştır. Middleware tasarımı sayesinde hem global hem de route bazlı
// olarak CORS eklemek mümkündür.
//
// CORSMiddleware, gelen her isteğe uygun "Access-Control-Allow-*" başlıklarını
// ekler ve tarayıcıların preflight (OPTIONS) isteklerine doğru yanıt verilmesini
// sağlar. Böylece uygulama güvenli ve kontrollü bir şekilde cross-origin istek
// alabilir.
// -----------------------------------------------------------------------------

package middleware

import (
	"net/http"
)

// CORSMiddleware, belirli bir origin'e izin veren CORS yapılandırmasını geri
// döndüren bir middleware üreticisidir. allowedOrigin parametresi ile, hangi
// domain'in API'ye erişim sağlayabileceği kontrol edilir.
//
// Middleware'in yaptığı işlemler:
//   - Access-Control-Allow-Origin başlığını belirlemek
//   - Preflight (OPTIONS) isteklerini otomatik olarak yanıtlamak
//   - Tarayıcının ihtiyaç duyduğu güvenlik başlıklarını eklemek
//
// Bu yapı, frontend uygulamalarıyla API'nin problemsiz şekilde iletişim
// kurmasını sağlar.
func CORSMiddleware(allowedOrigin string) Middleware {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Her isteğe Access-Control-Allow-Origin başlığı eklenir.
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

			// Tarayıcı, bazı isteklerden önce OPTIONS methodu ile "preflight"
			// kontrolü yapar. Bu durumda sunucu izin verilen method ve header'ları
			// bildirmeli ve 204 döndürmelidir.
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

				w.WriteHeader(http.StatusNoContent) // 204 — içeriksiz başarılı yanıt
				return                              // İşlemi burada sonlandır
			}

			// OPTIONS dışındaki istekler normal handler'a yönlendirilir.
			next.ServeHTTP(w, r)
		})
	}
}
