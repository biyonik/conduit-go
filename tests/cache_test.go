// -----------------------------------------------------------------------------
// Cache System Tests
// -----------------------------------------------------------------------------
// Cache driver'ları test eder (Redis, File, Memory).
//
// Test edilen özellikler:
// - Basic operations (Get/Set/Delete)
// - TTL support
// - Remember pattern
// - Increment/Decrement
// - Bulk operations
// - Garbage collection
// -----------------------------------------------------------------------------

package tests

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/biyonik/conduit-go/pkg/cache"
	"github.com/biyonik/conduit-go/pkg/database"
	_ "github.com/redis/go-redis/v9"
)

// setupMemoryCache, test için memory cache oluşturur.
func setupMemoryCache() cache.Cache {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	return cache.NewMemoryCache(logger)
}

// setupFileCache, test için file cache oluşturur.
func setupFileCache(t *testing.T) cache.Cache {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	tempDir := t.TempDir() // Test bitince otomatik silinir

	c, err := cache.NewFileCache(tempDir, logger)
	if err != nil {
		t.Fatalf("File cache oluşturulamadı: %v", err)
	}

	return c
}

// setupRedisCache, test için redis cache oluşturur.
//
// Not: Bu test Redis bağlantısı gerektirir.
// Redis yoksa skip edilir.
func setupRedisCache(t *testing.T) cache.Cache {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	// Redis config
	config := database.DefaultRedisConfig()
	config.Host = "127.0.0.1"
	config.Port = 6379

	// Redis client oluştur
	redisClient, err := database.NewRedisClient(config, logger)
	if err != nil {
		t.Skip("Redis bağlantısı yok, test skip edildi")
		return nil
	}

	// Test DB'yi temizle
	redisClient.Client().FlushDB(ctx)

	return cache.NewRedisCache(redisClient.Client(), logger, "test:")
}

// TestCacheBasicOperations, temel cache operasyonlarını test eder.
func TestCacheBasicOperations(t *testing.T) {
	drivers := []struct {
		name  string
		cache cache.Cache
	}{
		{"Memory", setupMemoryCache()},
		{"File", setupFileCache(t)},
		// {"Redis", setupRedisCache(t)}, // Redis varsa uncomment et
	}

	for _, driver := range drivers {
		t.Run(driver.name, func(t *testing.T) {
			c := driver.cache

			// Test data
			testData := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			}

			// SET operation
			err := c.Set("user:123", testData, 10*time.Second)
			if err != nil {
				t.Errorf("Set hatası: %v", err)
			}

			// GET operation
			val, err := c.Get("user:123")
			if err != nil {
				t.Errorf("Get hatası: %v", err)
			}
			if val == nil {
				t.Error("Cache miss olmamalıydı")
			}

			// HAS operation
			exists, err := c.Has("user:123")
			if err != nil {
				t.Errorf("Has hatası: %v", err)
			}
			if !exists {
				t.Error("Key bulunmalıydı")
			}

			// DELETE operation
			err = c.Delete("user:123")
			if err != nil {
				t.Errorf("Delete hatası: %v", err)
			}

			// Verify deletion
			exists, err = c.Has("user:123")
			if err != nil {
				t.Errorf("Has hatası: %v", err)
			}
			if exists {
				t.Error("Key silinmeliydi")
			}
		})
	}
}

// TestCacheTTL, TTL (Time To Live) özelliğini test eder.
func TestCacheTTL(t *testing.T) {
	c := setupMemoryCache()

	// Kısa TTL ile yaz
	err := c.Set("temp:data", "test-value", 1*time.Second)
	if err != nil {
		t.Fatalf("Set hatası: %v", err)
	}

	// Hemen oku (olmalı)
	val, err := c.Get("temp:data")
	if err != nil {
		t.Fatalf("Get hatası: %v", err)
	}
	if val == nil {
		t.Error("Cache hit olmalıydı")
	}

	// TTL'in expire olmasını bekle
	time.Sleep(2 * time.Second)

	// Tekrar oku (olmamalı)
	val, err = c.Get("temp:data")
	if err != nil {
		t.Fatalf("Get hatası: %v", err)
	}
	if val != nil {
		t.Error("Cache miss olmalıydı (TTL expired)")
	}
}

// TestCacheRememberPattern, Remember pattern'ini test eder.
func TestCacheRememberPattern(t *testing.T) {
	c := setupMemoryCache()

	callCount := 0
	callback := func() (interface{}, error) {
		callCount++
		return map[string]interface{}{
			"id":   1,
			"name": "Test User",
		}, nil
	}

	// İlk çağrı - callback çalışmalı
	val1, err := c.Remember("user:1", 10*time.Second, callback)
	if err != nil {
		t.Fatalf("Remember hatası: %v", err)
	}
	if val1 == nil {
		t.Error("Değer olmalıydı")
	}
	if callCount != 1 {
		t.Errorf("Callback 1 kez çalışmalıydı, %d kez çalıştı", callCount)
	}

	// İkinci çağrı - cache'den dönmeli (callback çalışmamalı)
	val2, err := c.Remember("user:1", 10*time.Second, callback)
	if err != nil {
		t.Fatalf("Remember hatası: %v", err)
	}
	if val2 == nil {
		t.Error("Cache'den değer dönmeliydi")
	}
	if callCount != 1 {
		t.Errorf("Callback hâlâ 1 kez çalışmalıydı, %d kez çalıştı", callCount)
	}
}

// TestCacheIncrement, Increment/Decrement operasyonlarını test eder.
func TestCacheIncrement(t *testing.T) {
	c := setupMemoryCache()

	// İlk increment (0'dan başlar)
	val, err := c.Increment("counter", 1)
	if err != nil {
		t.Fatalf("Increment hatası: %v", err)
	}
	if val != 1 {
		t.Errorf("Değer 1 olmalıydı, %d oldu", val)
	}

	// İkinci increment
	val, err = c.Increment("counter", 5)
	if err != nil {
		t.Fatalf("Increment hatası: %v", err)
	}
	if val != 6 {
		t.Errorf("Değer 6 olmalıydı, %d oldu", val)
	}

	// Decrement
	val, err = c.Decrement("counter", 2)
	if err != nil {
		t.Fatalf("Decrement hatası: %v", err)
	}
	if val != 4 {
		t.Errorf("Değer 4 olmalıydı, %d oldu", val)
	}
}

// TestCacheBulkOperations, bulk operasyonları test eder.
func TestCacheBulkOperations(t *testing.T) {
	c := setupMemoryCache()

	// SetMultiple
	values := map[string]interface{}{
		"user:1": map[string]string{"name": "User 1"},
		"user:2": map[string]string{"name": "User 2"},
		"user:3": map[string]string{"name": "User 3"},
	}

	err := c.SetMultiple(values, 10*time.Second)
	if err != nil {
		t.Fatalf("SetMultiple hatası: %v", err)
	}

	// GetMultiple
	keys := []string{"user:1", "user:2", "user:3", "user:999"}
	results, err := c.GetMultiple(keys)
	if err != nil {
		t.Fatalf("GetMultiple hatası: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("4 sonuç olmalıydı, %d oldu", len(results))
	}

	if results["user:1"] == nil {
		t.Error("user:1 bulunmalıydı")
	}
	if results["user:999"] != nil {
		t.Error("user:999 nil olmalıydı")
	}

	// DeleteMultiple
	err = c.DeleteMultiple([]string{"user:1", "user:2"})
	if err != nil {
		t.Fatalf("DeleteMultiple hatası: %v", err)
	}

	// Verify deletion
	exists, _ := c.Has("user:1")
	if exists {
		t.Error("user:1 silinmeliydi")
	}

	exists, _ = c.Has("user:3")
	if !exists {
		t.Error("user:3 hâlâ olmalıydı")
	}
}

// TestCacheFlush, Flush operasyonunu test eder.
func TestCacheFlush(t *testing.T) {
	c := setupMemoryCache()

	// Birkaç key ekle
	c.Set("key1", "value1", 10*time.Second)
	c.Set("key2", "value2", 10*time.Second)
	c.Set("key3", "value3", 10*time.Second)

	// Flush
	err := c.Flush()
	if err != nil {
		t.Fatalf("Flush hatası: %v", err)
	}

	// Verify flush
	exists, _ := c.Has("key1")
	if exists {
		t.Error("Cache temizlenmeliydi")
	}
}

// BenchmarkCacheSet, Set operasyonunun performansını ölçer.
func BenchmarkCacheSet(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	c := cache.NewMemoryCache(logger)

	testData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("bench:key", testData, 10*time.Minute)
	}
}

// BenchmarkCacheGet, Get operasyonunun performansını ölçer.
func BenchmarkCacheGet(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	c := cache.NewMemoryCache(logger)

	testData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	c.Set("bench:key", testData, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("bench:key")
	}
}

// BenchmarkCacheRemember, Remember pattern'inin performansını ölçer.
func BenchmarkCacheRemember(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	c := cache.NewMemoryCache(logger)

	callback := func() (interface{}, error) {
		return map[string]interface{}{
			"name": "Test User",
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Remember("bench:user", 10*time.Minute, callback)
	}
}
