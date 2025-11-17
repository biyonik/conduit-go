// -----------------------------------------------------------------------------
// Cache Interface
// -----------------------------------------------------------------------------
// Laravel-style cache interface tanımı.
//
// Bu dosya tüm cache driver'ların implement etmesi gereken interface'i tanımlar.
// Driver'lar: Redis, File, Memory
//
// Özellikler:
// - Get/Set/Delete operations
// - TTL (Time To Live) support
// - Remember pattern (cache or execute)
// - Increment/Decrement for counters
// - Flush (clear all)
// -----------------------------------------------------------------------------

package cache

import (
	"time"
)

// Cache, tüm cache driver'ların implement etmesi gereken interface.
//
// Bu interface Laravel Cache facade pattern'ini takip eder.
// Her driver (Redis, File, Memory) bu interface'i implement eder.
//
// Örnek kullanım:
//
//	var cache Cache = NewRedisCache(redisClient, logger)
//	cache.Set("user:123", user, 10*time.Minute)
type Cache interface {
	// Get, cache'den veri okur.
	//
	// Key bulunamazsa nil döner, hata vermez.
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//
	// Döndürür:
	//   - interface{}: Cache'deki değer (JSON decode edilmiş)
	//   - error: Okuma hatası
	//
	// Örnek:
	//   value, err := cache.Get("user:123")
	//   if err != nil {
	//       return err
	//   }
	//   if value == nil {
	//       // Cache miss
	//   }
	Get(key string) (interface{}, error)

	// Set, cache'e veri yazar.
	//
	// TTL (Time To Live) belirtilirse, süre sonunda otomatik silinir.
	// TTL = 0 ise süresiz saklanır (dikkatli kullan!).
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//   - value: Saklanacak değer (JSON encode edilir)
	//   - ttl: Geçerlilik süresi (0 = süresiz)
	//
	// Döndürür:
	//   - error: Yazma hatası
	//
	// Örnek:
	//   err := cache.Set("user:123", user, 10*time.Minute)
	//
	// Güvenlik Notu:
	// - Sensitive data cache'lemeden önce encrypt edilmeli
	// - TTL mutlaka belirlenmeli (memory leak önlemek için)
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete, cache'den veri siler.
	//
	// Key bulunamazsa hata vermez, sessizce geçer.
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//
	// Döndürür:
	//   - error: Silme hatası
	//
	// Örnek:
	//   err := cache.Delete("user:123")
	Delete(key string) error

	// Has, key'in cache'de olup olmadığını kontrol eder.
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//
	// Döndürür:
	//   - bool: Key varsa true
	//   - error: Kontrol hatası
	//
	// Örnek:
	//   exists, err := cache.Has("user:123")
	//   if exists {
	//       // Cache hit
	//   }
	Has(key string) (bool, error)

	// Remember, cache'den okur, bulamazsa fonksiyonu çalıştırıp cache'ler.
	//
	// Bu Laravel'in en popüler pattern'lerinden biri:
	// "Cache'de varsa al, yoksa hesapla ve cache'le"
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//   - ttl: Geçerlilik süresi
	//   - callback: Cache miss durumunda çalışacak fonksiyon
	//
	// Döndürür:
	//   - interface{}: Cache'deki veya yeni hesaplanan değer
	//   - error: İşlem hatası
	//
	// Örnek:
	//   users, err := cache.Remember("users:all", 10*time.Minute, func() (interface{}, error) {
	//       return userRepo.GetAll()
	//   })
	//
	// Güvenlik Notu:
	// - Callback fonksiyonu thread-safe olmalı
	// - Race condition'a karşı dikkatli olunmalı
	Remember(key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error)

	// Increment, sayısal değeri artırır.
	//
	// Counter, rate limiting gibi use case'ler için kullanılır.
	// Key yoksa 0'dan başlatır.
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//   - value: Artırılacak miktar (varsayılan: 1)
	//
	// Döndürür:
	//   - int64: Yeni değer
	//   - error: İşlem hatası
	//
	// Örnek:
	//   count, err := cache.Increment("page:views", 1)
	Increment(key string, value int64) (int64, error)

	// Decrement, sayısal değeri azaltır.
	//
	// Parametreler:
	//   - key: Cache anahtarı
	//   - value: Azaltılacak miktar (varsayılan: 1)
	//
	// Döndürür:
	//   - int64: Yeni değer
	//   - error: İşlem hatası
	//
	// Örnek:
	//   remaining, err := cache.Decrement("api:quota:user123", 1)
	Decrement(key string, value int64) (int64, error)

	// Flush, tüm cache'i temizler.
	//
	// UYARI: Bu operasyon geri alınamaz!
	// Production'da dikkatli kullanılmalı.
	//
	// Döndürür:
	//   - error: Temizleme hatası
	//
	// Örnek:
	//   if err := cache.Flush(); err != nil {
	//       log.Printf("Cache flush hatası: %v", err)
	//   }
	//
	// Güvenlik Notu:
	// - Admin yetkisi gerektirebilir
	// - Audit log tutulmalı
	Flush() error

	// GetMultiple, birden fazla key'i tek seferde okur.
	//
	// Network round-trip azaltmak için kullanılır.
	//
	// Parametreler:
	//   - keys: Cache anahtarları
	//
	// Döndürür:
	//   - map[string]interface{}: Key-value map (bulunamayanlar nil)
	//   - error: Okuma hatası
	//
	// Örnek:
	//   values, err := cache.GetMultiple([]string{"user:1", "user:2", "user:3"})
	GetMultiple(keys []string) (map[string]interface{}, error)

	// SetMultiple, birden fazla key-value'yi tek seferde yazar.
	//
	// Parametreler:
	//   - values: Key-value map
	//   - ttl: Geçerlilik süresi (tüm değerler için aynı)
	//
	// Döndürür:
	//   - error: Yazma hatası
	//
	// Örnek:
	//   values := map[string]interface{}{
	//       "user:1": user1,
	//       "user:2": user2,
	//   }
	//   err := cache.SetMultiple(values, 10*time.Minute)
	SetMultiple(values map[string]interface{}, ttl time.Duration) error

	// DeleteMultiple, birden fazla key'i tek seferde siler.
	//
	// Parametreler:
	//   - keys: Cache anahtarları
	//
	// Döndürür:
	//   - error: Silme hatası
	//
	// Örnek:
	//   err := cache.DeleteMultiple([]string{"user:1", "user:2"})
	DeleteMultiple(keys []string) error
}

// Stats, cache istatistikleri interface.
//
// Monitoring ve debugging için kullanılır.
// Tüm driver'lar optional olarak implement edebilir.
type Stats interface {
	// Stats, cache istatistiklerini döndürür.
	//
	// Döndürür:
	//   - map[string]interface{}: İstatistik verileri
	//
	// Örnek:
	//   if s, ok := cache.(Stats); ok {
	//       stats := s.Stats()
	//       log.Printf("Cache stats: %+v", stats)
	//   }
	Stats() map[string]interface{}
}
