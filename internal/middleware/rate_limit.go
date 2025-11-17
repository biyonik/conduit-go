// -----------------------------------------------------------------------------
// Rate Limiting Middleware
// -----------------------------------------------------------------------------
// Bu middleware, belirli bir IP adresinden veya kullanıcıdan gelen istekleri
// sınırlandırır. DDoS saldırıları, brute force attack'ler ve API abuse'e karşı
// koruma sağlar.
//
// Algoritma: Token Bucket
// Her kullanıcı/IP için bir "bucket" (kova) vardır. Bu bucket'ta belirli sayıda
// token bulunur. Her istek bir token harcar. Tokenlar zamanla yenilenir.
//
// Örnek: 100 req/min limiti
// - Bucket kapasitesi: 100 token
// - Yenilenme hızı: 100 token / 60 saniye = 1.67 token/saniye
// - Kullanıcı 5 saniyede 50 istek atarsa, 50 token harcanır
// - 30 saniye sonra bucket'ta ~50 token yenilenir
//
// PRODUCTION NOTU:
// Bu implementasyon in-memory. Multiple server instance'ları için Redis kullanın!
// -----------------------------------------------------------------------------

package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// rateLimitBucket, bir kullanıcı/IP için token bucket'ını temsil eder.
type rateLimitBucket struct {
	tokens         float64   // Mevcut token sayısı (float çünkü zamanla kademeli artar)
	lastRefillTime time.Time // Son token yenilenme zamanı
}

// RateLimiter, tüm bucket'ları yöneten yapıdır.
type RateLimiter struct {
	mu              sync.RWMutex
	buckets         map[string]*rateLimitBucket // key (IP veya userID) -> bucket mapping
	maxRequests     int                         // Maksimum istek sayısı (bucket kapasitesi)
	windowInSeconds int                         // Zaman penceresi (saniye)
	refillRate      float64                     // Token yenilenme hızı (token/saniye)
}

// NewRateLimiter, yeni bir RateLimiter oluşturur.
//
// Parametreler:
//   - maxRequests: Zaman penceresi içinde maksimum istek sayısı
//   - windowInSeconds: Zaman penceresi (saniye)
//
// Döndürür:
//   - *RateLimiter: Yeni rate limiter instance
//
// Örnek:
//
//	limiter := NewRateLimiter(100, 60) // 100 req/min
//	limiter := NewRateLimiter(1000, 3600) // 1000 req/hour
func NewRateLimiter(maxRequests int, windowInSeconds int) *RateLimiter {
	return &RateLimiter{
		buckets:         make(map[string]*rateLimitBucket),
		maxRequests:     maxRequests,
		windowInSeconds: windowInSeconds,
		refillRate:      float64(maxRequests) / float64(windowInSeconds),
	}
}

// refillTokens, bucket'taki token'ları zamanla yeniler.
//
// Parametreler:
//   - bucket: Yenilenecek bucket
//   - now: Şimdiki zaman
//
// Token Bucket Algoritması:
// Geçen süre boyunca üretilen token sayısı hesaplanır ve bucket'a eklenir.
// Bucket'ın kapasitesi maxRequests'i aşamaz.
func (rl *RateLimiter) refillTokens(bucket *rateLimitBucket, now time.Time) {
	// Son yenilenme zamanından bu yana geçen süre
	elapsed := now.Sub(bucket.lastRefillTime).Seconds()

	// Bu sürede üretilen token sayısı
	newTokens := elapsed * rl.refillRate

	// Token'ları bucket'a ekle (maksimum kapasiteyi aşmayacak şekilde)
	bucket.tokens = min(bucket.tokens+newTokens, float64(rl.maxRequests))
	bucket.lastRefillTime = now
}

// Allow, belirtilen key için bir isteğin izin verilip verilmeyeceğini kontrol eder.
//
// Parametre:
//   - key: Rate limiting key'i (genellikle IP adresi veya user ID)
//
// Döndürür:
//   - bool: İstek izin veriliyorsa true
//   - int: Kalan token sayısı
//   - time.Duration: Retry-After süresi (limit aşıldıysa)
//
// Algoritma:
// 1. Bucket'ı bul (yoksa oluştur)
// 2. Token'ları yenile
// 3. En az 1 token varsa izin ver ve token'ı harca
// 4. Token yoksa reddet ve retry-after süresini hesapla
func (rl *RateLimiter) Allow(key string) (bool, int, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Bucket'ı al veya oluştur
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &rateLimitBucket{
			tokens:         float64(rl.maxRequests),
			lastRefillTime: now,
		}
		rl.buckets[key] = bucket
	}

	// Token'ları yenile
	rl.refillTokens(bucket, now)

	// En az 1 token var mı kontrol et
	if bucket.tokens >= 1.0 {
		// Token'ı harca
		bucket.tokens -= 1.0
		return true, int(bucket.tokens), 0
	}

	// Token yok, retry-after süresini hesapla
	// 1 token oluşması için gereken süre
	retryAfter := time.Duration(1.0/rl.refillRate) * time.Second

	return false, 0, retryAfter
}

// cleanup, expired bucket'ları periyodik olarak temizler.
//
// MEMORY LEAK KORUNMASI:
// Bu fonksiyon goroutine olarak çalıştırılmalı. Aksi takdirde bucket'lar
// süresiz olarak memory'de kalır (memory leak).
//
// Kullanım:
//
//	go limiter.cleanup(10 * time.Minute)
func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()

		now := time.Now()
		for key, bucket := range rl.buckets {
			// windowInSeconds süresinden daha uzun süredir aktif olmayan bucket'ları sil
			if now.Sub(bucket.lastRefillTime) > time.Duration(rl.windowInSeconds)*time.Second*2 {
				delete(rl.buckets, key)
			}
		}

		rl.mu.Unlock()
	}
}

// RateLimit, rate limiting middleware'ini döndürür.
//
// Parametreler:
//   - maxRequests: Zaman penceresi içinde maksimum istek sayısı
//   - windowInSeconds: Zaman penceresi (saniye)
//
// Döndürür:
//   - Middleware: Rate limiting middleware
//
// Kullanım:
//
//	// Global rate limit: 100 req/min
//	r.Use(middleware.RateLimit(100, 60))
//
//	// API endpoint'ine özel limit: 10 req/min
//	apiGroup := r.Group("/api")
//	apiGroup.Use(middleware.RateLimit(10, 60))
//
// Response Headers:
// - X-RateLimit-Limit: Maksimum istek sayısı
// - X-RateLimit-Remaining: Kalan token sayısı
// - X-RateLimit-Reset: Bucket'ın tamamen dolacağı zaman (Unix timestamp)
// - Retry-After: Limit aşıldıysa, kaç saniye sonra tekrar deneyebilir (sadece 429 durumunda)
//
// Güvenlik Notu:
// IP-based rate limiting, reverse proxy arkasındaysa (nginx, CloudFlare)
// X-Forwarded-For header'ını kullanmalısınız. Ancak bu header spoof edilebilir,
// bu yüzden güvenilir bir proxy'den geldiğinden emin olun!
func RateLimit(maxRequests int, windowInSeconds int) Middleware {
	limiter := NewRateLimiter(maxRequests, windowInSeconds)

	// Cleanup goroutine'ini başlat (memory leak koruması)
	go limiter.cleanup(10 * time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Rate limiting key'ini belirle
			// Öncelik sırası:
			// 1. Authenticated user ID (varsa)
			// 2. IP adresi

			key := r.RemoteAddr

			// TODO: Authentication eklendikten sonra user ID'yi kullan
			// if userID := auth.GetUserID(r); userID != "" {
			//     key = "user:" + userID
			// }

			// İsteğe izin ver
			allowed, remaining, retryAfter := limiter.Allow(key)

			// Rate limit header'larını ekle (RFC 6585)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(windowInSeconds)*time.Second).Unix()))

			if !allowed {
				// Limit aşıldı, 429 Too Many Requests dön
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
				response.Error(w, http.StatusTooManyRequests, fmt.Sprintf("Rate limit aşıldı. %d saniye sonra tekrar deneyin.", int(retryAfter.Seconds())))
				return
			}

			// İzin verildi, devam et
			next.ServeHTTP(w, r)
		})
	}
}

// min, iki float64 değerinden küçük olanını döndürür.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
