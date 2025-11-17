// -----------------------------------------------------------------------------
// Redis Connection Pool
// -----------------------------------------------------------------------------
// Redis bağlantı havuzu ve connection yönetimi.
//
// Bu dosya Redis sunucusuna bağlantı kurar ve connection pool yönetir.
// Context-based timeout ve error handling içerir.
//
// Özellikler:
// - Connection pooling
// - Health check
// - Graceful shutdown
// - Context timeout support
// -----------------------------------------------------------------------------

package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig, Redis bağlantı yapılandırması.
type RedisConfig struct {
	Host         string        // Redis sunucu adresi
	Port         int           // Redis port
	Password     string        // Redis şifresi (opsiyonel)
	DB           int           // Database numarası (0-15)
	PoolSize     int           // Connection pool boyutu
	MinIdleConns int           // Minimum idle connection sayısı
	MaxRetries   int           // Maksimum retry sayısı
	DialTimeout  time.Duration // Bağlantı timeout süresi
	ReadTimeout  time.Duration // Okuma timeout süresi
	WriteTimeout time.Duration // Yazma timeout süresi
}

// DefaultRedisConfig, varsayılan Redis yapılandırması.
//
// Production ortamı için önerilen değerler:
// - PoolSize: CPU core sayısının 2-4 katı
// - MinIdleConns: PoolSize'ın %25'i
// - Timeout'lar: Network latency'ye göre ayarlanmalı
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:         "127.0.0.1",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// RedisClient, Redis client wrapper.
//
// Bu struct redis.Client'ı wrap eder ve ek fonksiyonlar sağlar.
type RedisClient struct {
	client *redis.Client
	logger *log.Logger
}

// NewRedisClient, yeni bir Redis client oluşturur.
//
// Connection pool'u başlatır ve bağlantıyı test eder.
//
// Parametreler:
//   - config: Redis yapılandırması
//   - logger: Log instance
//
// Döndürür:
//   - *RedisClient: Redis client instance
//   - error: Bağlantı hatası
//
// Örnek:
//
//	config := DefaultRedisConfig()
//	client, err := NewRedisClient(config, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
// Güvenlik Notu:
// - Redis şifresi environment variable'dan okunmalı
// - Production'da TLS kullanılmalı (redis.Options.TLSConfig)
func NewRedisClient(config *RedisConfig, logger *log.Logger) (*RedisClient, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	// Redis client options
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	// Redis client oluştur
	client := redis.NewClient(opts)

	// Bağlantıyı test et
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Printf("❌ Redis bağlantı hatası: %v", err)
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	logger.Printf("✅ Redis bağlantısı başarılı: %s:%d (DB: %d)", config.Host, config.Port, config.DB)

	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

// Client, raw redis.Client instance döndürür.
//
// Cache implementation'ları için gerekli.
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// Ping, Redis sunucusunun erişilebilir olup olmadığını kontrol eder.
//
// Health check endpoint'lerinde kullanılabilir.
//
// Döndürür:
//   - error: Bağlantı hatası varsa
//
// Örnek:
//
//	if err := redisClient.Ping(); err != nil {
//	    return response.Error(w, 503, "Redis unavailable")
//	}
func (r *RedisClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return r.client.Ping(ctx).Err()
}

// Stats, Redis connection pool istatistiklerini döndürür.
//
// Monitoring ve debugging için kullanılır.
//
// Döndürür:
//   - map[string]interface{}: Pool istatistikleri
//
// Örnek:
//
//	stats := redisClient.Stats()
//	log.Printf("Redis Stats: %+v", stats)
func (r *RedisClient) Stats() map[string]interface{} {
	poolStats := r.client.PoolStats()

	return map[string]interface{}{
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"timeouts":    poolStats.Timeouts,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}
}

// FlushDB, mevcut database'i temizler.
//
// UYARI: Tüm cache verilerini siler!
// Sadece test ve development ortamlarında kullanılmalı.
//
// Döndürür:
//   - error: İşlem hatası
//
// Güvenlik Notu:
// - Production'da bu fonksiyon devre dışı bırakılmalı
// - Environment variable ile kontrol edilmeli (APP_ENV != production)
func (r *RedisClient) FlushDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.FlushDB(ctx).Err(); err != nil {
		r.logger.Printf("❌ Redis FlushDB hatası: %v", err)
		return fmt.Errorf("redis flush failed: %w", err)
	}

	r.logger.Println("⚠️  Redis database temizlendi (FlushDB)")
	return nil
}

// Close, Redis bağlantısını kapatır.
//
// Graceful shutdown sırasında çağrılmalı.
//
// Döndürür:
//   - error: Kapatma hatası
//
// Örnek:
//
//	defer redisClient.Close()
func (r *RedisClient) Close() error {
	if err := r.client.Close(); err != nil {
		r.logger.Printf("❌ Redis kapatma hatası: %v", err)
		return err
	}

	r.logger.Println("✅ Redis bağlantısı kapatıldı")
	return nil
}
