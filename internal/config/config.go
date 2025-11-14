// -----------------------------------------------------------------------------
// Config Package
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın merkezi konfigürasyon yönetimini sağlar. Laravel veya
// Symfony gibi frameworklerdeki .env ve config yapısına benzer bir şekilde,
// ortam değişkenlerini okuyarak uygulama, veritabanı ve sunucu ayarlarını
// merkezi olarak yönetir.
//
// Config yapısı, uygulamanın tüm kritik parametrelerini tip güvenli bir şekilde
// taşır ve varsayılan değerler ile birlikte çalışır. Eksik ortam değişkenleri
// olduğunda log üzerinden uyarı verir ve default değerleri kullanır.
// -----------------------------------------------------------------------------

package config

import (
	"log"
	"os"
)

// Config, uygulamanın merkezi yapılandırma nesnesidir. Bu nesne, uygulama,
// sunucu ve veritabanı ayarlarını kapsar.
//
// Alanlar:
//   - App: Uygulama ile ilgili ortam ve URL bilgisi
//   - DB: Veritabanı bağlantı bilgisi (DSN)
//   - Server: Sunucu portu
//
// Kullanım: config.Load() fonksiyonu ile ortam değişkenlerini okuyup
// yapılandırmayı alabilirsiniz.
type Config struct {
	App struct {
		Env string // Ortam (development, production, test vs.)
		URL string // Uygulama URL'si
	}
	DB struct {
		DSN string // Veritabanı bağlantı string'i
	}
	Server struct {
		Port string // Sunucunun çalışacağı port
	}
}

// Load, ortam değişkenlerini okuyarak Config nesnesini döndürür.
// Eksik değişkenlerde varsayılan değerleri kullanır ve log mesajı üretir.
func Load() *Config {
	cfg := &Config{}

	getEnv := func(key, defaultValue string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		log.Printf("Uyarı: %s ortam değişkeni bulunamadı, varsayılan (%s) kullanılıyor.", key, defaultValue)
		return defaultValue
	}

	cfg.App.Env = getEnv("APP_ENV", "development")
	cfg.App.URL = getEnv("APP_URL", "http://localhost:8000")
	cfg.Server.Port = getEnv("PORT", "8000")

	cfg.DB.DSN = getEnv("DB_DSN", "root:password@tcp(127.0.0.1:3306)/conduit_go?parseTime=true")

	return cfg
}
