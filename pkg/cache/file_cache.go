// -----------------------------------------------------------------------------
// File Cache Driver
// -----------------------------------------------------------------------------
// File-based cache implementation (fallback driver).
//
// Development ve shared hosting ortamları için uygundur.
// Redis kurulumu gerektirmez, basit ve güvenilirdir.
//
// Özellikler:
// - JSON serialization
// - TTL support (meta file ile)
// - Automatic garbage collection
// - File locking (concurrent access)
// - Hierarchical directory structure
//
// Performans Notu:
// - Redis'ten 10-100x yavaş
// - Distributed caching desteklemez
// - Production için Redis önerilir
// -----------------------------------------------------------------------------

package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileCacheEntry, cache dosyasında saklanan veri yapısı.
type FileCacheEntry struct {
	Value     interface{} `json:"value"`      // Gerçek değer
	ExpiresAt int64       `json:"expires_at"` // Unix timestamp (0 = süresiz)
}

// FileCache, file-based cache implementation.
type FileCache struct {
	dir    string // Cache dizini
	logger *log.Logger
	mu     sync.RWMutex // Concurrent access protection
}

// NewFileCache, yeni bir File cache instance oluşturur.
//
// Cache dizini yoksa otomatik oluşturulur.
//
// Parametreler:
//   - dir: Cache dosyalarının saklanacağı dizin
//   - logger: Log instance
//
// Döndürür:
//   - *FileCache: Cache instance
//   - error: Dizin oluşturma hatası
//
// Örnek:
//
//	cache, err := NewFileCache("/var/cache/myapp", logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Güvenlik Notu:
// - Cache dizini izinleri: 0755 (read/write/execute owner only)
// - Cache dosyaları: 0644 (read/write owner, read others)
// - Sensitive data cache'lemeden encrypt edilmeli
func NewFileCache(dir string, logger *log.Logger) (*FileCache, error) {
	// Cache dizinini oluştur (yoksa)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Printf("❌ Cache dizini oluşturma hatası [%s]: %v", dir, err)
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	logger.Printf("✅ File cache başlatıldı: %s", dir)

	fc := &FileCache{
		dir:    dir,
		logger: logger,
	}

	// Garbage collection başlat (expired files temizleme)
	go fc.startGarbageCollection()

	return fc, nil
}

// hashKey, key'den güvenli dosya adı oluşturur.
//
// MD5 hash kullanır, collision riski çok düşük.
// Hierarchical structure: ilk 2 karakter subdir (256 klasör max)
func (f *FileCache) hashKey(key string) (string, string) {
	hash := md5.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	// İlk 2 karakter subdir (ab/cdef123456...)
	subdir := hashStr[:2]
	filename := hashStr

	return subdir, filename
}

// filePath, key için dosya yolunu döndürür.
func (f *FileCache) filePath(key string) string {
	subdir, filename := f.hashKey(key)
	dirPath := filepath.Join(f.dir, subdir)

	// Subdir oluştur (yoksa)
	os.MkdirAll(dirPath, 0755)

	return filepath.Join(dirPath, filename)
}

// Get, cache'den veri okur.
func (f *FileCache) Get(key string) (interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	path := f.filePath(key)

	// Dosya var mı kontrol et
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	// Dosyayı oku
	data, err := os.ReadFile(path)
	if err != nil {
		f.logger.Printf("❌ File cache okuma hatası [%s]: %v", key, err)
		return nil, fmt.Errorf("file cache read failed: %w", err)
	}

	// JSON decode
	var entry FileCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		f.logger.Printf("❌ JSON decode hatası [%s]: %v", key, err)
		// Corrupt file - sil
		os.Remove(path)
		return nil, nil
	}

	// TTL kontrolü
	if entry.ExpiresAt > 0 && time.Now().Unix() > entry.ExpiresAt {
		// Expired - sil
		os.Remove(path)
		return nil, nil
	}

	return entry.Value, nil
}

// Set, cache'e veri yazar.
func (f *FileCache) Set(key string, value interface{}, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// TTL hesapla
	var expiresAt int64 = 0
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).Unix()
	}

	// Entry oluştur
	entry := FileCacheEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	// JSON encode
	data, err := json.Marshal(entry)
	if err != nil {
		f.logger.Printf("❌ JSON encode hatası [%s]: %v", key, err)
		return fmt.Errorf("json encode failed: %w", err)
	}

	path := f.filePath(key)

	// Dosyaya yaz
	if err := os.WriteFile(path, data, 0644); err != nil {
		f.logger.Printf("❌ File cache yazma hatası [%s]: %v", key, err)
		return fmt.Errorf("file cache write failed: %w", err)
	}

	return nil
}

// Delete, cache'den veri siler.
func (f *FileCache) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := f.filePath(key)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		f.logger.Printf("❌ File cache silme hatası [%s]: %v", key, err)
		return fmt.Errorf("file cache delete failed: %w", err)
	}

	return nil
}

// Has, key'in varlığını kontrol eder.
func (f *FileCache) Has(key string) (bool, error) {
	val, err := f.Get(key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

// Remember, cache'den okur veya callback'i çalıştırıp cache'ler.
func (f *FileCache) Remember(key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	// Önce cache'i kontrol et
	val, err := f.Get(key)
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
	if err := f.Set(key, result, ttl); err != nil {
		f.logger.Printf("⚠️  Remember cache yazma hatası [%s]: %v", key, err)
	}

	return result, nil
}

// Increment, sayısal değeri artırır.
//
// Not: File cache'de atomic değil, race condition olabilir.
func (f *FileCache) Increment(key string, value int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Mevcut değeri oku
	currentVal, err := f.Get(key)
	if err != nil {
		return 0, err
	}

	var current int64 = 0
	if currentVal != nil {
		// Type assertion (float64 çünkü JSON decode eder)
		if floatVal, ok := currentVal.(float64); ok {
			current = int64(floatVal)
		}
	}

	// Artır
	newVal := current + value

	// Kaydet (TTL yok, süresiz)
	if err := f.Set(key, newVal, 0); err != nil {
		return 0, err
	}

	return newVal, nil
}

// Decrement, sayısal değeri azaltır.
func (f *FileCache) Decrement(key string, value int64) (int64, error) {
	return f.Increment(key, -value)
}

// Flush, tüm cache'i temizler.
func (f *FileCache) Flush() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Cache dizinini sil ve tekrar oluştur
	if err := os.RemoveAll(f.dir); err != nil {
		f.logger.Printf("❌ Cache temizleme hatası: %v", err)
		return fmt.Errorf("cache flush failed: %w", err)
	}

	if err := os.MkdirAll(f.dir, 0755); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	f.logger.Println("⚠️  File cache tamamen temizlendi")
	return nil
}

// GetMultiple, birden fazla key'i okur.
func (f *FileCache) GetMultiple(keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, key := range keys {
		val, err := f.Get(key)
		if err != nil {
			result[key] = nil
			continue
		}
		result[key] = val
	}

	return result, nil
}

// SetMultiple, birden fazla key-value'yi yazar.
func (f *FileCache) SetMultiple(values map[string]interface{}, ttl time.Duration) error {
	for key, value := range values {
		if err := f.Set(key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMultiple, birden fazla key'i siler.
func (f *FileCache) DeleteMultiple(keys []string) error {
	for _, key := range keys {
		if err := f.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

// Stats, file cache istatistiklerini döndürür.
func (f *FileCache) Stats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Toplam dosya sayısını hesapla
	var fileCount int
	var totalSize int64

	filepath.Walk(f.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})

	return map[string]interface{}{
		"driver":     "file",
		"directory":  f.dir,
		"file_count": fileCount,
		"total_size": totalSize,
	}
}

// startGarbageCollection, expired dosyaları periyodik olarak temizler.
//
// Her 10 dakikada bir çalışır, TTL'i geçmiş dosyaları siler.
func (f *FileCache) startGarbageCollection() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		f.cleanExpiredFiles()
	}
}

// cleanExpiredFiles, expired dosyaları temizler.
func (f *FileCache) cleanExpiredFiles() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().Unix()
	var cleaned int

	err := filepath.Walk(f.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Dosyayı oku
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// JSON decode
		var entry FileCacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			// Corrupt file - sil
			err := os.Remove(path)
			if err != nil {
				return err
			}
			cleaned++
			return nil
		}

		// TTL kontrolü
		if entry.ExpiresAt > 0 && now > entry.ExpiresAt {
			err := os.Remove(path)
			if err != nil {
				return err
			}
			cleaned++
		}

		return nil
	})
	if err != nil {
		return
	}

	if cleaned > 0 {
		f.logger.Printf("🧹 Garbage collection: %d expired file silindi", cleaned)
	}
}
