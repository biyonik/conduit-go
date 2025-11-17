// -----------------------------------------------------------------------------
// Memory Cache Driver
// -----------------------------------------------------------------------------
// In-memory cache implementation (non-persistent).
//
// Testing ve geçici cache için idealdir.
// Request-level cache, unit test, development ortamlarında kullanılır.
//
// Özellikler:
// - Ultra-fast (direct memory access)
// - Thread-safe (sync.RWMutex)
// - TTL support (automatic cleanup)
// - No serialization overhead
// - No external dependencies
//
// Sınırlamalar:
// - Non-persistent (restart'ta kaybolur)
// - Single-server only (distributed değil)
// - Memory leak riski (dikkatli kullan!)
// -----------------------------------------------------------------------------

package cache

import (
	"log"
	"sync"
	"time"
)

// MemoryCacheEntry, memory'de saklanan veri yapısı.
type MemoryCacheEntry struct {
	Value     interface{} // Gerçek değer (pointer)
	ExpiresAt time.Time   // Expire zamanı (zero value = süresiz)
}

// IsExpired, entry'nin expire olup olmadığını kontrol eder.
func (e *MemoryCacheEntry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false // Süresiz
	}
	return time.Now().After(e.ExpiresAt)
}

// MemoryCache, in-memory cache implementation.
type MemoryCache struct {
	store  map[string]*MemoryCacheEntry
	mu     sync.RWMutex
	logger *log.Logger
}

// NewMemoryCache, yeni bir Memory cache instance oluşturur.
//
// Parametreler:
//   - logger: Log instance
//
// Döndürür:
//   - *MemoryCache: Cache instance
//
// Örnek:
//
//	cache := NewMemoryCache(logger)
//	cache.Set("user:123", user, 10*time.Minute)
//
// Performans Notu:
// - En hızlı cache driver (Redis'ten 10x hızlı)
// - Serialization overhead yok
// - Direct memory access
func NewMemoryCache(logger *log.Logger) *MemoryCache {
	mc := &MemoryCache{
		store:  make(map[string]*MemoryCacheEntry),
		logger: logger,
	}

	// Garbage collection başlat
	go mc.startGarbageCollection()

	logger.Println("✅ Memory cache başlatıldı")

	return mc
}

// Get, cache'den veri okur.
func (m *MemoryCache) Get(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.store[key]
	if !exists {
		return nil, nil // Cache miss
	}

	// TTL kontrolü
	if entry.IsExpired() {
		// Expired - silinecek (GC tarafından)
		return nil, nil
	}

	return entry.Value, nil
}

// Set, cache'e veri yazar.
func (m *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Expire zamanını hesapla
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.store[key] = &MemoryCacheEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	return nil
}

// Delete, cache'den veri siler.
func (m *MemoryCache) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.store, key)
	return nil
}

// Has, key'in varlığını kontrol eder.
func (m *MemoryCache) Has(key string) (bool, error) {
	val, err := m.Get(key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

// Remember, cache'den okur veya callback'i çalıştırıp cache'ler.
func (m *MemoryCache) Remember(key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	// Önce cache'i kontrol et
	val, err := m.Get(key)
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
	if err := m.Set(key, result, ttl); err != nil {
		m.logger.Printf("⚠️  Remember cache yazma hatası [%s]: %v", key, err)
	}

	return result, nil
}

// Increment, sayısal değeri artırır (thread-safe).
func (m *MemoryCache) Increment(key string, value int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.store[key]

	var current int64 = 0
	if exists && !entry.IsExpired() {
		// Type assertion
		if intVal, ok := entry.Value.(int64); ok {
			current = intVal
		}
	}

	// Artır
	newVal := current + value

	// Kaydet (TTL koru)
	var expiresAt time.Time
	if exists {
		expiresAt = entry.ExpiresAt
	}

	m.store[key] = &MemoryCacheEntry{
		Value:     newVal,
		ExpiresAt: expiresAt,
	}

	return newVal, nil
}

// Decrement, sayısal değeri azaltır.
func (m *MemoryCache) Decrement(key string, value int64) (int64, error) {
	return m.Increment(key, -value)
}

// Flush, tüm cache'i temizler.
func (m *MemoryCache) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store = make(map[string]*MemoryCacheEntry)
	m.logger.Println("⚠️  Memory cache tamamen temizlendi")

	return nil
}

// GetMultiple, birden fazla key'i okur.
func (m *MemoryCache) GetMultiple(keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, key := range keys {
		val, err := m.Get(key)
		if err != nil {
			result[key] = nil
			continue
		}
		result[key] = val
	}

	return result, nil
}

// SetMultiple, birden fazla key-value'yi yazar.
func (m *MemoryCache) SetMultiple(values map[string]interface{}, ttl time.Duration) error {
	for key, value := range values {
		if err := m.Set(key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMultiple, birden fazla key'i siler.
func (m *MemoryCache) DeleteMultiple(keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.store, key)
	}
	return nil
}

// Stats, memory cache istatistiklerini döndürür.
func (m *MemoryCache) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Expired olmayan entry sayısı
	validCount := 0
	for _, entry := range m.store {
		if !entry.IsExpired() {
			validCount++
		}
	}

	return map[string]interface{}{
		"driver":       "memory",
		"total_keys":   len(m.store),
		"valid_keys":   validCount,
		"expired_keys": len(m.store) - validCount,
	}
}

// startGarbageCollection, expired entry'leri periyodik olarak temizler.
//
// Her 5 dakikada bir çalışır, memory leak önler.
func (m *MemoryCache) startGarbageCollection() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanExpiredEntries()
	}
}

// cleanExpiredEntries, expired entry'leri temizler.
func (m *MemoryCache) cleanExpiredEntries() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleaned := 0
	for key, entry := range m.store {
		if entry.IsExpired() {
			delete(m.store, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		m.logger.Printf("🧹 Memory cache garbage collection: %d expired entry silindi", cleaned)
	}
}

// Size, cache'deki toplam entry sayısını döndürür.
//
// Debug ve monitoring için kullanılır.
func (m *MemoryCache) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.store)
}

// Clear, tüm entry'leri siler (Flush'ın alias'ı).
func (m *MemoryCache) Clear() error {
	return m.Flush()
}
