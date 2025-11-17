// cmd/api/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/internal/controllers"
	"github.com/biyonik/conduit-go/internal/middleware"
	"github.com/biyonik/conduit-go/internal/router"
	"github.com/biyonik/conduit-go/pkg/cache"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
)

// -----------------------------------------------------------------------------
// Application Entry Point (Phase 3: Cache System)
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın başlangıç noktasıdır.
//
// PHASE 1: Security & Stability
// - SQL Injection Protection
// - CSRF Protection
// - Rate Limiting
// - Graceful Shutdown
//
// PHASE 2: Authentication & Authorization
// - JWT-based authentication
// - User registration & login
// - Password reset
// - Role-based authorization
//
// PHASE 3: Advanced Features
// - Redis Cache System
// - File Cache (fallback)
// - Memory Cache (testing)
// - Laravel-style cache interface
// -----------------------------------------------------------------------------

func main() {
	// =========================================================================
	// 1. DEPENDENCY INJECTION CONTAINER'I BAŞLAT
	// =========================================================================
	c := container.New()

	// =========================================================================
	// 2. SERVİSLERİ KONTEYNERE KAYDET
	// =========================================================================

	// Config servisi
	c.Register(func(c *container.Container) (*config.Config, error) {
		return config.Load(), nil
	})

	// Logger servisi
	c.Register(func(c *container.Container) (*log.Logger, error) {
		return log.New(os.Stdout, "[Conduit-Go] ", log.Ldate|log.Ltime|log.Lshortfile), nil
	})

	// Veritabanı Bağlantısı
	c.Register(func(c *container.Container) (*sql.DB, error) {
		cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
		db, err := database.Connect(cfg.DB.DSN)
		if err != nil {
			return nil, err
		}
		return db, nil
	})

	// SQL Grammar
	c.Register(func(c *container.Container) (database.Grammar, error) {
		return database.NewMySQLGrammar(), nil
	})

	// =========================================================================
	// 3. PHASE 3: CACHE SYSTEM INITIALIZATION
	// =========================================================================

	// Cache servisi - driver'a göre oluştur
	c.Register(func(c *container.Container) (cache.Cache, error) {
		cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
		logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)

		switch cfg.Cache.Driver {
		case "redis":
			// Redis Cache
			logger.Println("🔄 Redis cache başlatılıyor...")

			redisConfig := &database.RedisConfig{
				Host:         cfg.Redis.Host,
				Port:         cfg.Redis.Port,
				Password:     cfg.Redis.Password,
				DB:           cfg.Redis.DB,
				PoolSize:     10,
				MinIdleConns: 2,
				MaxRetries:   3,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			}

			redisClient, err := database.NewRedisClient(redisConfig, logger)
			if err != nil {
				logger.Printf("⚠️  Redis bağlantısı başarısız, file cache'e geçiliyor: %v", err)
				// Fallback to file cache
				return cache.NewFileCache(cfg.Cache.FileDir, logger)
			}

			// Redis client'ı container'a kaydet (shutdown için gerekli)
			c.Register(func(c *container.Container) (*database.RedisClient, error) {
				return redisClient, nil
			})

			logger.Printf("✅ Redis cache başlatıldı (prefix: %s)", cfg.Cache.Prefix)
			return cache.NewRedisCache(redisClient.Client(), logger, cfg.Cache.Prefix), nil

		case "file":
			// File Cache
			logger.Println("🔄 File cache başlatılıyor...")
			fileCache, err := cache.NewFileCache(cfg.Cache.FileDir, logger)
			if err != nil {
				return nil, fmt.Errorf("file cache oluşturulamadı: %w", err)
			}
			logger.Printf("✅ File cache başlatıldı (dir: %s)", cfg.Cache.FileDir)
			return fileCache, nil

		case "memory":
			// Memory Cache
			logger.Println("🔄 Memory cache başlatılıyor...")
			if cfg.IsProduction() {
				logger.Println("⚠️  UYARI: Memory cache production ortamı için önerilmez!")
			}
			logger.Println("✅ Memory cache başlatıldı")
			return cache.NewMemoryCache(logger), nil

		default:
			return nil, fmt.Errorf("geçersiz cache driver: %s", cfg.Cache.Driver)
		}
	})

	// Controller'lar
	c.Register(controllers.NewAppController)
	c.Register(controllers.NewAuthController)
	c.Register(controllers.NewPasswordController)

	// =========================================================================
	// 4. GEREKLI SERVİSLERİ RESOLVE ET
	// =========================================================================
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
	cacheDriver := c.MustGet(reflect.TypeOf((*cache.Cache)(nil)).Elem()).(cache.Cache)
	appController := c.MustGet(reflect.TypeOf((*controllers.AppController)(nil))).(*controllers.AppController)
	authController := c.MustGet(reflect.TypeOf((*controllers.AuthController)(nil))).(*controllers.AuthController)
	passwordController := c.MustGet(reflect.TypeOf((*controllers.PasswordController)(nil))).(*controllers.PasswordController)

	// =========================================================================
	// 5. CACHE DEMO (Opsiyonel - Development için)
	// =========================================================================
	if cfg.IsDevelopment() {
		logger.Println("\n📝 Cache System Demo:")

		// Set example
		err := cacheDriver.Set("app:version", "1.0.0-phase3", 10*time.Minute)
		if err != nil {
			logger.Printf("⚠️  Cache set hatası: %v", err)
		} else {
			logger.Println("✅ Cache set: app:version = 1.0.0-phase3")
		}

		// Get example
		version, err := cacheDriver.Get("app:version")
		if err != nil {
			logger.Printf("⚠️  Cache get hatası: %v", err)
		} else if version != nil {
			logger.Printf("✅ Cache get: app:version = %v", version)
		}

		// Remember pattern example
		startTime := time.Now()
		data, err := cacheDriver.Remember("demo:expensive", 5*time.Minute, func() (interface{}, error) {
			logger.Println("   🔄 Expensive operation simulating...")
			time.Sleep(100 * time.Millisecond)
			return map[string]string{"result": "computed"}, nil
		})
		elapsed := time.Since(startTime)
		if err != nil {
			logger.Printf("⚠️  Remember hatası: %v", err)
		} else {
			logger.Printf("✅ Remember: %v (took: %v)", data, elapsed)
		}

		// Second call (should be cached)
		startTime = time.Now()
		data2, _ := cacheDriver.Remember("demo:expensive", 5*time.Minute, func() (interface{}, error) {
			logger.Println("   ❌ Bu mesaj görünmemeli!")
			return nil, nil
		})
		elapsed2 := time.Since(startTime)
		logger.Printf("✅ Remember (cached): %v (took: %v)\n", data2, elapsed2)
	}

	// =========================================================================
	// 6. ROUTER'I OLUŞTUR VE MIDDLEWARE'LERI KAYDET
	// =========================================================================
	r := router.New()

	// Global Middleware'ler (Sıralama önemli!)
	r.Use(middleware.PanicRecovery(logger)) // 1. Panic yakalama
	r.Use(middleware.Logging)               // 2. Request logging
	r.Use(middleware.CORSMiddleware("*"))   // 3. CORS
	r.Use(middleware.RateLimit(100, 60))    // 4. Rate limiting: 100 req/min

	// =========================================================================
	// 7. PUBLIC ROTALARI TANIMLA
	// =========================================================================

	// Genel endpoint'ler
	r.GET("/", appController.HomeHandler)

	// Health check endpoint - Cache status dahil
	r.GET("/health", appController.HealthHandler)

	// =========================================================================
	// 8. AUTH ROTALARI (PUBLIC - Authentication gerektirmez)
	// =========================================================================
	authGroup := r.Group("/api/auth")

	// CSRF koruması ekle (POST/PUT/DELETE için)
	authGroup.Use(middleware.CSRFProtection())

	// Daha sıkı rate limit (brute force koruması)
	authGroup.Use(middleware.RateLimit(10, 60)) // 10 req/min

	// Authentication endpoint'leri
	authGroup.POST("/register", authController.Register)
	authGroup.POST("/login", authController.Login)
	authGroup.POST("/refresh", authController.RefreshToken)

	// Password reset endpoint'leri
	authGroup.POST("/forgot-password", passwordController.ForgotPassword)
	authGroup.POST("/reset-password", passwordController.ResetPassword)

	// =========================================================================
	// 9. PROTECTED ROTALARI TANIMLA (Authentication gerekir)
	// =========================================================================

	// Authenticated user endpoint'leri
	r.POST("/api/auth/logout", authController.Logout).
		Middleware(middleware.Auth())

	r.GET("/api/auth/profile", authController.Profile).
		Middleware(middleware.Auth())

	r.PUT("/api/auth/profile", authController.UpdateProfile).
		Middleware(middleware.Auth()).
		Middleware(middleware.CSRFProtection())

	r.PUT("/api/auth/password", authController.ChangePassword).
		Middleware(middleware.Auth()).
		Middleware(middleware.CSRFProtection())

	// =========================================================================
	// 10. API V1 ROUTES (Authenticated + Stricter Limits)
	// =========================================================================
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.Auth())            // Tüm API endpoint'leri protected
	apiV1.Use(middleware.RateLimit(50, 60)) // API için daha sıkı limit: 50 req/min

	// Test endpoint (authenticated)
	apiV1.GET("/check", appController.CheckHandler)
	apiV1.GET("/testquery", appController.TestQueryHandler)

	// =========================================================================
	// 11. ADMIN ROTALARI (Sadece admin'ler erişebilir)
	// =========================================================================
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(middleware.Auth())            // Authentication gerekli
	adminGroup.Use(middleware.Admin())           // Admin role gerekli
	adminGroup.Use(middleware.RateLimit(30, 60)) // Admin için limit: 30 req/min

	// Admin endpoint'leri (Phase 3'te eklenecek)
	// adminGroup.GET("/users", adminController.ListUsers)
	// adminGroup.DELETE("/users/{id}", adminController.DeleteUser)

	// =========================================================================
	// 12. HTTP SUNUCUSUNU YAPILANDIR
	// =========================================================================
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// =========================================================================
	// 13. SUNUCUYU GOROUTINE'DE BAŞLAT
	// =========================================================================
	go func() {
		logger.Println("\n" + strings.Repeat("=", 70))
		logger.Printf("🚀 Conduit-Go Framework v1.0.0 (Phase 3)")
		logger.Println(strings.Repeat("=", 70))
		logger.Printf("📍 Server: http://localhost:%s", cfg.Server.Port)
		logger.Printf("🌐 Environment: %s", cfg.App.Env)
		logger.Printf("💾 Cache Driver: %s", cfg.Cache.Driver)
		if cfg.Cache.Driver == "redis" {
			logger.Printf("🔗 Redis: %s:%d (DB: %d)", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)
		}
		logger.Println(strings.Repeat("-", 70))
		logger.Println("📡 Available Endpoints:")
		logger.Println("   PUBLIC:")
		logger.Printf("   - GET  /health")
		logger.Println("   AUTH:")
		logger.Printf("   - POST /api/auth/register")
		logger.Printf("   - POST /api/auth/login")
		logger.Printf("   - POST /api/auth/refresh")
		logger.Printf("   - POST /api/auth/forgot-password")
		logger.Printf("   - POST /api/auth/reset-password")
		logger.Println("   PROTECTED:")
		logger.Printf("   - POST /api/auth/logout")
		logger.Printf("   - GET  /api/auth/profile")
		logger.Printf("   - PUT  /api/auth/profile")
		logger.Printf("   - PUT  /api/auth/password")
		logger.Println("   API:")
		logger.Printf("   - GET  /api/v1/check")
		logger.Printf("   - GET  /api/v1/testquery")
		logger.Println(strings.Repeat("=", 70))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("❌ Sunucu başlatılamadı: %v", err)
		}
	}()

	// =========================================================================
	// 14. GRACEFUL SHUTDOWN
	// =========================================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Println("\n🛑 Kapanma sinyali alındı, graceful shutdown başlatılıyor...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// HTTP sunucusu kapat
	logger.Println("⏳ HTTP sunucusu kapatılıyor...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("⚠️  HTTP sunucusu zorla kapatıldı: %v", err)
	} else {
		logger.Println("✅ HTTP sunucusu gracefully kapatıldı")
	}

	// Redis client kapat (varsa)
	if cfg.Cache.Driver == "redis" {
		logger.Println("⏳ Redis bağlantısı kapatılıyor...")
		if redisClient, _ := c.Get(reflect.TypeOf((*database.RedisClient)(nil))); redisClient != nil {
			if rc, e := redisClient.(*database.RedisClient); e {
				if err := rc.Close(); err != nil {
					logger.Printf("⚠️  Redis kapatılamadı: %v", err)
				} else {
					logger.Println("✅ Redis bağlantısı kapatıldı")
				}
			}
		}
	}

	// Database bağlantıları kapat
	logger.Println("⏳ Database bağlantıları kapatılıyor...")
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	if err := db.Close(); err != nil {
		logger.Printf("⚠️  Database kapatılamadı: %v", err)
	} else {
		logger.Println("✅ Database bağlantıları kapatıldı")
	}

	logger.Println("👋 Uygulama temiz bir şekilde kapatıldı. Hoşça kal!")
}
