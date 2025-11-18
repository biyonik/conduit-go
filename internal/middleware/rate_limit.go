package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// -----------------------------------------------------------------------------
// Rate Limiting Middleware (MEMORY LEAK FIXED)
// -----------------------------------------------------------------------------
// FIXED ISSUES:
// ✅ Goroutine leak - cleanup artık gracefully durdurulabiliyor
// ✅ Context-based shutdown mekanizması
// ✅ Limiter registry ile multiple limiter'ların lifecycle management
// -----------------------------------------------------------------------------

type rateLimitBucket struct {
	tokens         float64
	lastRefillTime time.Time
}

type RateLimiter struct {
	mu              sync.RWMutex
	buckets         map[string]*rateLimitBucket
	maxRequests     int
	windowInSeconds int
	refillRate      float64
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// Global limiter registry - graceful shutdown için
var (
	limiterRegistry   = make(map[*RateLimiter]bool)
	limiterRegistryMu sync.Mutex
)

// NewRateLimiter, yeni bir RateLimiter oluşturur.
func NewRateLimiter(maxRequests int, windowInSeconds int) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	limiter := &RateLimiter{
		buckets:         make(map[string]*rateLimitBucket),
		maxRequests:     maxRequests,
		windowInSeconds: windowInSeconds,
		refillRate:      float64(maxRequests) / float64(windowInSeconds),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Registry'e ekle
	limiterRegistryMu.Lock()
	limiterRegistry[limiter] = true
	limiterRegistryMu.Unlock()

	// Cleanup goroutine'ini başlat
	limiter.startCleanup()

	return limiter
}

// startCleanup, cleanup goroutine'ini başlatır.
func (rl *RateLimiter) startCleanup() {
	rl.wg.Add(1)
	go rl.cleanupLoop()
}

// cleanupLoop, periyodik olarak expired bucket'ları temizler.
func (rl *RateLimiter) cleanupLoop() {
	defer rl.wg.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.ctx.Done():
			// Graceful shutdown
			return
		}
	}
}

// cleanup, expired bucket'ları temizler.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for key, bucket := range rl.buckets {
		// windowInSeconds süresinden daha uzun süredir aktif olmayan bucket'ları sil
		if now.Sub(bucket.lastRefillTime) > time.Duration(rl.windowInSeconds)*time.Second*2 {
			delete(rl.buckets, key)
			cleaned++
		}
	}

	// Debug için - production'da kaldırılabilir
	_ = cleaned
}

// Stop, rate limiter'ı gracefully durdurur.
func (rl *RateLimiter) Stop() {
	// Registry'den çıkar
	limiterRegistryMu.Lock()
	delete(limiterRegistry, rl)
	limiterRegistryMu.Unlock()

	// Goroutine'i durdur
	rl.cancel()
	rl.wg.Wait()
}

// StopAllLimiters, tüm aktif rate limiter'ları durdurur.
// Bu fonksiyon main.go'daki shutdown hook'undan çağrılmalı.
func StopAllLimiters() {
	limiterRegistryMu.Lock()
	limiters := make([]*RateLimiter, 0, len(limiterRegistry))
	for limiter := range limiterRegistry {
		limiters = append(limiters, limiter)
	}
	limiterRegistryMu.Unlock()

	// Tüm limiter'ları durdur
	for _, limiter := range limiters {
		limiter.Stop()
	}
}

// refillTokens, bucket'taki token'ları zamanla yeniler.
func (rl *RateLimiter) refillTokens(bucket *rateLimitBucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefillTime).Seconds()
	newTokens := elapsed * rl.refillRate
	bucket.tokens = min(bucket.tokens+newTokens, float64(rl.maxRequests))
	bucket.lastRefillTime = now
}

// Allow, belirtilen key için bir isteğin izin verilip verilmeyeceğini kontrol eder.
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
		bucket.tokens -= 1.0
		return true, int(bucket.tokens), 0
	}

	// Token yok, retry-after süresini hesapla
	retryAfter := time.Duration(1.0/rl.refillRate) * time.Second
	return false, 0, retryAfter
}

// RateLimit, rate limiting middleware'ini döndürür.
func RateLimit(maxRequests int, windowInSeconds int) Middleware {
	limiter := NewRateLimiter(maxRequests, windowInSeconds)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Rate limiting key'ini belirle
			key := r.RemoteAddr

			// TODO: Authentication eklendikten sonra user ID'yi kullan
			// if userID := auth.GetUserID(r); userID != "" {
			//     key = "user:" + userID
			// }

			// İsteğe izin ver
			allowed, remaining, retryAfter := limiter.Allow(key)

			// Rate limit header'larını ekle
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(windowInSeconds)*time.Second).Unix()))

			if !allowed {
				// Limit aşıldı
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
				response.Error(w, http.StatusTooManyRequests, fmt.Sprintf("Rate limit aşıldı. %d saniye sonra tekrar deneyin.", int(retryAfter.Seconds())))
				return
			}

			// İzin verildi
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