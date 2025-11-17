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
//
// Phase 2: JWT, Authentication yapılandırması eklendi
// Phase 3: Redis, Cache, Queue, Mail yapılandırması eklendi
// -----------------------------------------------------------------------------

package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Config, uygulamanın merkezi yapılandırma nesnesidir.
//
// Nested struct yapısı kullanılarak ilgili ayarlar gruplandırılmıştır:
//   - App: Uygulama genel ayarları
//   - Server: Sunucu ayarları
//   - DB: Veritabanı ayarları
//   - JWT: Authentication token ayarları (Phase 2)
//   - Redis: Redis bağlantı ayarları (Phase 3)
//   - Cache: Cache sistem ayarları (Phase 3)
//   - RateLimit: Rate limiting ayarları
//   - Mail: Mail gönderim ayarları (Phase 3)
type Config struct {
	App struct {
		Name string // Uygulama adı
		Env  string // Ortam (development, production, test)
		URL  string // Uygulama URL'si
	}

	Server struct {
		Port string // Sunucunun çalışacağı port
	}

	DB struct {
		DSN             string        // Veritabanı bağlantı string'i
		MaxOpenConns    int           // Maksimum açık bağlantı sayısı
		MaxIdleConns    int           // Maksimum boşta bekleyen bağlantı sayısı
		ConnMaxLifetime time.Duration // Bağlantı maksimum ömrü
	}

	// Phase 2: JWT Authentication
	JWT struct {
		Secret            string        // JWT secret key
		Expiration        time.Duration // Access token süresi
		RefreshExpiration time.Duration // Refresh token süresi
	}

	// Phase 3: Redis Configuration
	Redis struct {
		Host     string // Redis host adresi
		Port     int    // Redis port
		Password string // Redis şifresi (opsiyonel)
		DB       int    // Database numarası (0-15)
	}

	// Phase 3: Cache Configuration
	Cache struct {
		Driver  string // Cache driver: redis, file, memory
		Prefix  string // Cache key prefix (namespace)
		FileDir string // File cache dizini (file driver için)
	}

	// Rate Limiting
	RateLimit struct {
		Enabled       bool // Rate limiting aktif mi?
		MaxRequests   int  // Maksimum istek sayısı
		WindowSeconds int  // Zaman penceresi (saniye)
	}

	// Phase 3: Mail Configuration
	Mail struct {
		Driver      string // Mail driver: smtp, sendmail
		Host        string // SMTP host
		Port        int    // SMTP port
		FromAddress string // Gönderici email adresi
	}
}

// Load, ortam değişkenlerini okuyarak Config nesnesini döndürür.
//
// Eksik değişkenlerde varsayılan değerleri kullanır ve log mesajı üretir.
// Tüm ayarlar environment variable'lardan okunur (.env dosyası veya sistem).
//
// Döndürür:
//   - *Config: Yapılandırma nesnesi
//
// Örnek kullanım:
//
//	cfg := config.Load()
//	log.Printf("Environment: %s", cfg.App.Env)
//	log.Printf("Cache Driver: %s", cfg.Cache.Driver)
func Load() *Config {
	cfg := &Config{}

	// Helper function: Ortam değişkenini oku, yoksa default kullan
	getEnv := func(key, defaultValue string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		log.Printf("⚠️  Uyarı: %s ortam değişkeni bulunamadı, varsayılan (%s) kullanılıyor.", key, defaultValue)
		return defaultValue
	}

	// Helper function: Integer ortam değişkeni
	getEnvAsInt := func(key string, defaultValue int) int {
		valueStr := os.Getenv(key)
		if valueStr == "" {
			log.Printf("⚠️  Uyarı: %s ortam değişkeni bulunamadı, varsayılan (%d) kullanılıyor.", key, defaultValue)
			return defaultValue
		}

		value, err := strconv.Atoi(valueStr)
		if err != nil {
			log.Printf("⚠️  Uyarı: %s için geçersiz değer: %s, varsayılan (%d) kullanılıyor.", key, valueStr, defaultValue)
			return defaultValue
		}

		return value
	}

	// Helper function: Boolean ortam değişkeni
	getEnvAsBool := func(key string, defaultValue bool) bool {
		valueStr := os.Getenv(key)
		if valueStr == "" {
			return defaultValue
		}

		value, err := strconv.ParseBool(valueStr)
		if err != nil {
			log.Printf("⚠️  Uyarı: %s için geçersiz boolean değer: %s, varsayılan (%t) kullanılıyor.", key, valueStr, defaultValue)
			return defaultValue
		}

		return value
	}

	// Helper function: Duration ortam değişkeni (saniye cinsinden)
	getEnvAsDuration := func(key string, defaultSeconds int) time.Duration {
		seconds := getEnvAsInt(key, defaultSeconds)
		return time.Duration(seconds) * time.Second
	}

	// Application Configuration
	cfg.App.Name = getEnv("APP_NAME", "Conduit-Go")
	cfg.App.Env = getEnv("APP_ENV", "development")
	cfg.App.URL = getEnv("APP_URL", "http://localhost:8000")

	// Server Configuration
	cfg.Server.Port = getEnv("PORT", "8000")

	// Database Configuration
	cfg.DB.DSN = getEnv("DB_DSN", "root:password@tcp(127.0.0.1:3306)/conduit_go?parseTime=true")
	cfg.DB.MaxOpenConns = getEnvAsInt("DB_MAX_OPEN_CONNS", 25)
	cfg.DB.MaxIdleConns = getEnvAsInt("DB_MAX_IDLE_CONNS", 25)
	cfg.DB.ConnMaxLifetime = getEnvAsDuration("DB_CONN_MAX_LIFETIME", 300) // 5 dakika

	// JWT Configuration (Phase 2)
	cfg.JWT.Secret = getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production")
	cfg.JWT.Expiration = getEnvAsDuration("JWT_EXPIRATION", 3600)                  // 1 saat
	cfg.JWT.RefreshExpiration = getEnvAsDuration("JWT_REFRESH_EXPIRATION", 604800) // 7 gün

	// Redis Configuration (Phase 3)
	cfg.Redis.Host = getEnv("REDIS_HOST", "127.0.0.1")
	cfg.Redis.Port = getEnvAsInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)

	// Cache Configuration (Phase 3)
	cfg.Cache.Driver = getEnv("CACHE_DRIVER", "memory") // redis, file, memory
	cfg.Cache.Prefix = getEnv("CACHE_PREFIX", "conduit:")
	cfg.Cache.FileDir = getEnv("CACHE_FILE_DIR", "./storage/cache")

	// Rate Limiting Configuration
	cfg.RateLimit.Enabled = getEnvAsBool("RATE_LIMIT_ENABLED", true)
	cfg.RateLimit.MaxRequests = getEnvAsInt("RATE_LIMIT_MAX_REQUESTS", 100)
	cfg.RateLimit.WindowSeconds = getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60)

	// Mail Configuration (Phase 3)
	cfg.Mail.Driver = getEnv("MAIL_DRIVER", "smtp")
	cfg.Mail.Host = getEnv("MAIL_HOST", "localhost")
	cfg.Mail.Port = getEnvAsInt("MAIL_PORT", 1025)
	cfg.Mail.FromAddress = getEnv("MAIL_FROM_ADDRESS", "noreply@conduit-go.local")

	// Validation
	if err := cfg.Validate(); err != nil {
		log.Printf("❌ Config validation hatası: %v", err)
	}

	return cfg
}

// Validate, config değerlerinin geçerliliğini kontrol eder.
//
// Production ortamı için kritik kontroller yapar:
// - JWT secret uzunluğu (min 32 karakter)
// - Cache driver geçerliliği
// - Production'da default secret kontrolü
//
// Döndürür:
//   - error: Validation hatası (varsa)
func (c *Config) Validate() error {
	// JWT secret kontrolü (Production)
	if c.IsProduction() {
		if len(c.JWT.Secret) < 32 {
			return fmt.Errorf("JWT_SECRET production'da en az 32 karakter olmalı")
		}

		// Default secret kontrolü
		if c.JWT.Secret == "your-super-secret-jwt-key-change-this-in-production" {
			return fmt.Errorf("JWT_SECRET production'da değiştirilmelidir")
		}
	}

	// Cache driver kontrolü
	validDrivers := map[string]bool{
		"redis":  true,
		"file":   true,
		"memory": true,
	}
	if !validDrivers[c.Cache.Driver] {
		return fmt.Errorf("geçersiz CACHE_DRIVER: %s (redis, file veya memory olmalı)", c.Cache.Driver)
	}

	// Production uyarıları
	if c.IsProduction() {
		if c.Cache.Driver == "memory" {
			log.Println("⚠️  UYARI: Memory cache production ortamı için önerilmez!")
		}
	}

	return nil
}

// IsProduction, uygulamanın production ortamında çalışıp çalışmadığını kontrol eder.
//
// Döndürür:
//   - bool: Production ise true
//
// Örnek:
//
//	if cfg.IsProduction() {
//	    // Production-specific logic
//	}
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// IsDevelopment, uygulamanın development ortamında çalışıp çalışmadığını kontrol eder.
//
// Döndürür:
//   - bool: Development ise true
//
// Örnek:
//
//	if cfg.IsDevelopment() {
//	    // Development-specific logic (debug, verbose logging, etc.)
//	}
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsTest, uygulamanın test ortamında çalışıp çalışmadığını kontrol eder.
//
// Döndürür:
//   - bool: Test ortamı ise true
//
// Örnek:
//
//	if cfg.IsTest() {
//	    // Test-specific logic
//	}
func (c *Config) IsTest() bool {
	return c.App.Env == "test"
}

// LoadConfig, Load() fonksiyonunun alias'ıdır (backward compatibility).
//
// Bazı kod parçaları LoadConfig() kullanabilir, bu yüzden ikisini de destekliyoruz.
//
// Döndürür:
//   - *Config: Yapılandırma nesnesi
//   - error: Her zaman nil (Load() hata döndürmez)
func LoadConfig() (*Config, error) {
	return Load(), nil
}
