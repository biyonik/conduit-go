package cache

import (
	"context"
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

// -----------------------------------------------------------------------------
// File Cache Driver (MEMORY LEAK + RACE CONDITION FIXED)
// -----------------------------------------------------------------------------
// FIXED ISSUES:
// ✅ Goroutine leak - GC artık gracefully durdurulabiliyor
// ✅ Race condition - Get() içinde lock upgrade pattern kullanılıyor
// ✅ Context-based shutdown mekanizması
// -----------------------------------------------------------------------------

type FileCacheEntry struct {
	Value     interface{} `json:"value"`
	ExpiresAt int64       `json:"expires_at"`
}

type FileCache struct {
	dir    string
	logger *log.Logger
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewFileCache, yeni bir File cache instance oluşturur.
func NewFileCache(dir string, logger *log.Logger) (*FileCache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Printf("❌ Cache dizini oluşturma hatası [%s]: %v", dir, err)
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	logger.Printf("✅ File cache başlatıldı: %s", dir)

	ctx, cancel := context.WithCancel(context.Background())

	fc := &FileCache{
		dir:    dir,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Garbage collection başlat
	fc.startGarbageCollection()

	return fc, nil
}

// startGarbageCollection, GC goroutine'ini başlatır.
func (f *FileCache) startGarbageCollection() {
	f.wg.Add(1)
	go f.garbageCollectionLoop()
}

// garbageCollectionLoop, periyodik olarak expired dosyaları temizler.
func (f *FileCache) garbageCollectionLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.cleanExpiredFiles()
		case <-f.ctx.Done():
			// Graceful shutdown
			f.logger.Println("🛑 File cache garbage collector durduruluyor...")
			return
		}
	}
}

// Stop, file cache'i gracefully durdurur.
func (f *FileCache) Stop() {
	f.cancel()
	f.wg.Wait()
}

// hashKey, key'den güvenli dosya adı oluşturur.
func (f *FileCache) hashKey(key string) (string, string) {
	hash := md5.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	subdir := hashStr[:2]
	filename := hashStr
	return subdir, filename
}

// filePath, key için dosya yolunu döndürür.
func (f *FileCache) filePath(key string) string {
	subdir, filename := f.hashKey(key)
	dirPath := filepath.Join(f.dir, subdir)
	os.MkdirAll(dirPath, 0755)
	return filepath.Join(dirPath, filename)
}

// Get, cache'den veri okur.
// RACE CONDITION FIX: Lock upgrade pattern kullanılıyor
func (f *FileCache) Get(key string) (interface{}, error) {
	path := f.filePath(key)

	// 1. Read lock ile dosyayı oku
	f.mu.RLock()

	// Dosya var mı kontrol et
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f.mu.RUnlock()
		return nil, nil // Cache miss
	}

	// Dosyayı oku
	data, err := os.ReadFile(path)
	if err != nil {
		f.mu.RUnlock()
		f.logger.Printf("❌ File cache okuma hatası [%s]: %v", key, err)
		return nil, fmt.Errorf("file cache read failed: %w", err)
	}

	// JSON decode
	var entry FileCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		f.mu.RUnlock()
		f.logger.Printf("❌ JSON decode hatası [%s]: %v", key, err)

		// Corrupt file - write lock ile sil
		f.mu.Lock()
		os.Remove(path)
		f.mu.Unlock()
		return nil, nil
	}

	// TTL kontrolü (hala read lock içindeyiz)
	if entry.ExpiresAt > 0 && time.Now().Unix() > entry.ExpiresAt {
		// Read lock'u bırak
		f.mu.RUnlock()

		// Write lock al ve dosyayı sil
		f.mu.Lock()
		// Double-check: başka goroutine silmiş olabilir
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			os.Remove(path)
		}
		f.mu.Unlock()

		return nil, nil
	}

	// Read lock'u bırak
	f.mu.RUnlock()

	return entry.Value, nil
}

// Set, cache'e veri yazar.
func (f *FileCache) Set(key string, value interface{}, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var expiresAt int64 = 0
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).Unix()
	}

	entry := FileCacheEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		f.logger.Printf("❌ JSON encode hatası [%s]: %v", key, err)
		return fmt.Errorf("json encode failed: %w", err)
	}

	path := f.filePath(key)

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
	val, err := f.Get(key)
	if err != nil {
		return nil, err
	}

	if val != nil {
		return val, nil
	}

	result, err := callback()
	if err != nil {
		return nil, err
	}

	if err := f.Set(key, result, ttl); err != nil {
		f.logger.Printf("⚠️  Remember cache yazma hatası [%s]: %v", key, err)
	}

	return result, nil
}

// Increment, sayısal değeri artırır.
func (f *FileCache) Increment(key string, value int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	currentVal, err := f.Get(key)
	if err != nil {
		return 0, err
	}

	var current int64 = 0
	if currentVal != nil {
		if floatVal, ok := currentVal.(float64); ok {
			current = int64(floatVal)
		}
	}

	newVal := current + value

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

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var entry FileCacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			// Corrupt file - sil
			if err := os.Remove(path); err == nil {
				cleaned++
			}
			return nil
		}

		// TTL kontrolü
		if entry.ExpiresAt > 0 && now > entry.ExpiresAt {
			if err := os.Remove(path); err == nil {
				cleaned++
			}
		}

		return nil
	})

	if err == nil && cleaned > 0 {
		f.logger.Printf("🧹 Garbage collection: %d expired file silindi", cleaned)
	}
}
