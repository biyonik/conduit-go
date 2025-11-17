// -----------------------------------------------------------------------------
// Redis Cache Driver
// -----------------------------------------------------------------------------
// Redis-based cache implementation.
//
// Production ortamı için önerilen cache driver.
// Distributed caching, high performance, persistence destekler.
//
// Özellikler:
// - JSON serialization
// - TTL support
// - Atomic operations (Increment/Decrement)
// - Pipeline support (GetMultiple/SetMultiple)
// - Connection pooling
// -----------------------------------------------------------------------------

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache, Redis-based cache implementation.
type RedisCache struct {
	client *redis.Client
	logger *log.Logger
	prefix string // Key prefix (namespace)
}

// NewRedisCache, yeni bir Redis cache instance oluşturur.
//
// Parametreler:
//   - client: Redis client
//   - logger: Log instance
//   - prefix: Cache key prefix (opsiyonel, örn: "myapp:")
//
// Döndürür:
//   - *RedisCache: Cache instance
//
// Örnek:
//
//	cache := NewRedisCache(redisClient, logger, "myapp:")
//	cache.Set("users:all", users, 10*time.Minute)
//	// Gerçek key: "myapp:users:all"
func NewRedisCache(client *redis.Client, logger *log.Logger, prefix string) *RedisCache {
	return &RedisCache{
		client: client,
		logger: logger,
		prefix: prefix,
	}
}

// prefixKey, key'e prefix ekler.
func (r *RedisCache) prefixKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + key
}

// Get, cache'den veri okur.
func (r *RedisCache) Get(key string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	prefixedKey := r.prefixKey(key)
	val, err := r.client.Get(ctx, prefixedKey).Result()

	// Key bulunamadı (cache miss)
	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		r.logger.Printf("❌ Redis Get hatası [%s]: %v", prefixedKey, err)
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	// JSON decode
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		r.logger.Printf("❌ JSON decode hatası [%s]: %v", prefixedKey, err)
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	return result, nil
}

// Set, cache'e veri yazar.
func (r *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// JSON encode
	data, err := json.Marshal(value)
	if err != nil {
		r.logger.Printf("❌ JSON encode hatası [%s]: %v", key, err)
		return fmt.Errorf("json encode failed: %w", err)
	}

	prefixedKey := r.prefixKey(key)

	// Redis'e yaz
	if err := r.client.Set(ctx, prefixedKey, data, ttl).Err(); err != nil {
		r.logger.Printf("❌ Redis Set hatası [%s]: %v", prefixedKey, err)
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Delete, cache'den veri siler.
func (r *RedisCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	prefixedKey := r.prefixKey(key)
	if err := r.client.Del(ctx, prefixedKey).Err(); err != nil {
		r.logger.Printf("❌ Redis Delete hatası [%s]: %v", prefixedKey, err)
		return fmt.Errorf("redis delete failed: %w", err)
	}

	return nil
}

// Has, key'in varlığını kontrol eder.
func (r *RedisCache) Has(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	prefixedKey := r.prefixKey(key)
	count, err := r.client.Exists(ctx, prefixedKey).Result()
	if err != nil {
		r.logger.Printf("❌ Redis Exists hatası [%s]: %v", prefixedKey, err)
		return false, fmt.Errorf("redis exists failed: %w", err)
	}

	return count > 0, nil
}

// Remember, cache'den okur veya callback'i çalıştırıp cache'ler.
//
// Thread-safe değil! Production'da lock mechanism eklenebilir.
func (r *RedisCache) Remember(key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	// Önce cache'i kontrol et
	val, err := r.Get(key)
	if err != nil {
		return nil, err
	}

	// Cache hit
	if val != nil {
		return val, nil
	}

	// Cache miss - callback çalıştır
	result, err := callback()
	if err != nil {
		return nil, err
	}

	// Cache'e yaz
	if err := r.Set(key, result, ttl); err != nil {
		// Cache yazma hatası - result'u döndür ama log tut
		r.logger.Printf("⚠️  Remember cache yazma hatası [%s]: %v", key, err)
	}

	return result, nil
}

// Increment, sayısal değeri artırır (atomic).
func (r *RedisCache) Increment(key string, value int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	prefixedKey := r.prefixKey(key)
	newVal, err := r.client.IncrBy(ctx, prefixedKey, value).Result()
	if err != nil {
		r.logger.Printf("❌ Redis Increment hatası [%s]: %v", prefixedKey, err)
		return 0, fmt.Errorf("redis increment failed: %w", err)
	}

	return newVal, nil
}

// Decrement, sayısal değeri azaltır (atomic).
func (r *RedisCache) Decrement(key string, value int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	prefixedKey := r.prefixKey(key)
	newVal, err := r.client.DecrBy(ctx, prefixedKey, value).Result()
	if err != nil {
		r.logger.Printf("❌ Redis Decrement hatası [%s]: %v", prefixedKey, err)
		return 0, fmt.Errorf("redis decrement failed: %w", err)
	}

	return newVal, nil
}

// Flush, tüm cache'i temizler.
//
// UYARI: Prefix varsa sadece o namespace temizlenir.
// Prefix yoksa TÜM Redis database temizlenir!
func (r *RedisCache) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prefix varsa sadece o namespace'i temizle
	if r.prefix != "" {
		pattern := r.prefix + "*"
		iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

		keys := []string{}
		for iter.Next(ctx) {
			keys = append(keys, iter.Val())
		}

		if err := iter.Err(); err != nil {
			r.logger.Printf("❌ Redis Scan hatası: %v", err)
			return fmt.Errorf("redis scan failed: %w", err)
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				r.logger.Printf("❌ Redis Flush hatası: %v", err)
				return fmt.Errorf("redis flush failed: %w", err)
			}
		}

		r.logger.Printf("⚠️  Redis cache temizlendi [prefix: %s, keys: %d]", r.prefix, len(keys))
		return nil
	}

	// Prefix yoksa tüm DB'yi temizle
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		r.logger.Printf("❌ Redis FlushDB hatası: %v", err)
		return fmt.Errorf("redis flushdb failed: %w", err)
	}

	r.logger.Println("⚠️  Redis database tamamen temizlendi (FlushDB)")
	return nil
}

// GetMultiple, birden fazla key'i pipeline ile okur.
func (r *RedisCache) GetMultiple(keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return map[string]interface{}{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prefix ekle
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = r.prefixKey(key)
	}

	// Pipeline kullan (tek network round-trip)
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(prefixedKeys))
	for i, key := range prefixedKeys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		r.logger.Printf("❌ Redis Pipeline hatası: %v", err)
		return nil, fmt.Errorf("redis pipeline failed: %w", err)
	}

	// Sonuçları topla
	result := make(map[string]interface{})
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			result[keys[i]] = nil
			continue
		}
		if err != nil {
			result[keys[i]] = nil
			continue
		}

		var decoded interface{}
		if err := json.Unmarshal([]byte(val), &decoded); err != nil {
			r.logger.Printf("⚠️  JSON decode hatası [%s]: %v", keys[i], err)
			result[keys[i]] = nil
			continue
		}

		result[keys[i]] = decoded
	}

	return result, nil
}

// SetMultiple, birden fazla key-value'yi pipeline ile yazar.
func (r *RedisCache) SetMultiple(values map[string]interface{}, ttl time.Duration) error {
	if len(values) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pipeline kullan
	pipe := r.client.Pipeline()
	for key, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			r.logger.Printf("❌ JSON encode hatası [%s]: %v", key, err)
			continue
		}

		prefixedKey := r.prefixKey(key)
		pipe.Set(ctx, prefixedKey, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Printf("❌ Redis Pipeline Set hatası: %v", err)
		return fmt.Errorf("redis pipeline set failed: %w", err)
	}

	return nil
}

// DeleteMultiple, birden fazla key'i pipeline ile siler.
func (r *RedisCache) DeleteMultiple(keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prefix ekle
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = r.prefixKey(key)
	}

	if err := r.client.Del(ctx, prefixedKeys...).Err(); err != nil {
		r.logger.Printf("❌ Redis Delete Multiple hatası: %v", err)
		return fmt.Errorf("redis delete multiple failed: %w", err)
	}

	return nil
}

// Stats, Redis cache istatistiklerini döndürür.
func (r *RedisCache) Stats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		r.logger.Printf("❌ Redis Info hatası: %v", err)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"driver": "redis",
		"prefix": r.prefix,
		"info":   info,
	}
}
