// -----------------------------------------------------------------------------
// Middleware Package
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın HTTP istek yaşam döngüsüne müdahale eden ve hem
// güvenlik hem de gözlemlenebilirlik açısından kritik rol oynayan middleware
// yapısını içerir. Laravel veya Symfony gibi framework'lerde yer alan
// “HTTP Kernel” mantığının Go'ya uyarlanmış, sade fakat güçlü bir modelidir.
//
// Middleware yapısı, bir http.Handler'ı alıp yeni bir http.Handler üreten
// fonksiyonlardan oluşur. Böylece istek işlenmeden önce veya sonra ek işlemler
// gerçekleştirmek mümkündür. Logging, Authentication, Rate Limiting gibi birçok
// özellik bu yapının üzerine kolayca inşa edilir.
//
// Bu dosyada özellikle Logging middleware'i bulunmaktadır. Bu middleware,
// uygulamaya gelen her isteğin giriş ve çıkışında log tutarak debugging,
// performans analizi ve monitoring açısından önemli bir görev üstlenir.
// -----------------------------------------------------------------------------

package middleware

import (
	"log"
	"net/http"
	"time"
)

// Middleware, bir sonraki http.Handler'ı alıp onu yeni bir handler olarak
// saran fonksiyon tipidir. Bu, Go'nun net/http mimarisinde cross-cutting
// concerns oluşturmanın standart yoludur. Örneğin: logging, authentication,
// panic recovery gibi işlemler bu yapı sayesinde route'lardan bağımsız çalışır.
type Middleware func(next http.Handler) http.Handler

// Logging, gelen her HTTP isteğini kaydeden basit ama etkili bir middleware'dir.
// İstek işlenmeden önce method ve path loglanır, işlem tamamlandıktan sonra ise
// geçen süre ile birlikte tekrar log yazılır.
//
// Bu sayede hangi isteğin ne kadar sürede işlendiği gerçek zamanlı olarak takip
// edilebilir. Uygulama performansı, debugging ihtiyaçları ve API izleme açısından
// oldukça değerlidir.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // İşlem başlangıç zamanı

		log.Printf("-> %s %s", r.Method, r.URL.Path) // İstek girişi logu

		next.ServeHTTP(w, r) // Bir sonraki handler'ı çalıştır

		// İşlem bitiş logu, toplam süre ile birlikte
		log.Printf("<- %s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
