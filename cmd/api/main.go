// cmd/api/main.go
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/internal/controllers"
	"github.com/biyonik/conduit-go/internal/middleware"
	"github.com/biyonik/conduit-go/internal/router"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
)

// -----------------------------------------------------------------------------
// Application Entry Point
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın başlangıç noktasıdır. Dependency Injection container'ı
// başlatır, servisleri kaydeder, router'ı yapılandırır ve HTTP sunucusunu başlatır.
//
// YENİ: GRACEFUL SHUTDOWN
// Uygulama artık SIGINT (Ctrl+C) ve SIGTERM sinyallerini yakalar ve
// graceful shutdown yapar. Bu sayede:
// - Aktif istekler tamamlanır
// - Database bağlantıları düzgün kapatılır
// - Kaynak sızıntıları (resource leak) önlenir
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

	// Veritabanı Bağlantısı (*sql.DB)
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

	// Controller'lar
	c.Register(controllers.NewAppController)

	// =========================================================================
	// 3. GEREKLI SERVİSLERİ RESOLVE ET
	// =========================================================================
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
	appController := c.MustGet(reflect.TypeOf((*controllers.AppController)(nil))).(*controllers.AppController)

	// =========================================================================
	// 4. ROUTER'I OLUŞTUR VE MIDDLEWARE'LERI KAYDET
	// =========================================================================
	r := router.New()

	// Global Middleware'ler (Sıralama önemli!)
	r.Use(middleware.PanicRecovery(logger)) // 1. Panic yakalama (en dışta olmalı)
	r.Use(middleware.Logging)               // 2. Request logging
	r.Use(middleware.CORSMiddleware("*"))   // 3. CORS
	r.Use(middleware.CSRFProtection())      // 4. CSRF protection (YENİ!)
	r.Use(middleware.RateLimit(100, 60))    // 5. Rate limiting: 100 req/min (YENİ!)

	// =========================================================================
	// 5. ROTALARI TANIMLA
	// =========================================================================
	r.GET("/", appController.HomeHandler)
	r.GET("/health", appController.HealthHandler) // Health check endpoint (YENİ!)
	r.GET("/api/check", appController.CheckHandler)
	r.GET("/api/testquery", appController.TestQueryHandler)

	// API Group (daha sıkı rate limit)
	apiGroup := r.Group("/api/v1")
	apiGroup.Use(middleware.RateLimit(50, 60)) // API için 50 req/min

	// TODO: İleride eklenecek rotalar:
	// apiGroup.POST("/register", userController.Register)
	// apiGroup.POST("/login", userController.Login)
	// apiGroup.GET("/profile", userController.Profile).Middleware(middleware.Auth("jwt"))

	// =========================================================================
	// 6. HTTP SUNUCUSUNU YAPΙLANDΙR
	// =========================================================================
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        r,
		ReadTimeout:    15 * time.Second, // İstek okuma timeout'u
		WriteTimeout:   15 * time.Second, // Response yazma timeout'u
		IdleTimeout:    60 * time.Second, // Keep-alive connection timeout'u
		MaxHeaderBytes: 1 << 20,          // 1 MB (büyük header saldırılarına karşı)
	}

	// =========================================================================
	// 7. SUNUCUYU GOROUTINE'DE BAŞLAT (NON-BLOCKING)
	// =========================================================================
	go func() {
		logger.Printf("🚀 Conduit Go çalışıyor (Port: %s, Ortam: %s)...", cfg.Server.Port, cfg.App.Env)
		logger.Printf("📍 Health Check: http://localhost:%s/health", cfg.Server.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("❌ Sunucu başlatılamadı: %v", err)
		}
	}()

	// =========================================================================
	// 8. GRACEFUL SHUTDOWN İÇİN SİNYAL DİNLEYİCİSİ
	// =========================================================================
	// OS sinyallerini dinlemek için bir channel oluştur
	quit := make(chan os.Signal, 1)

	// SIGINT (Ctrl+C) ve SIGTERM sinyallerini yakala
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Bloklanır ve sinyal gelene kadar bekler
	<-quit
	logger.Println("🛑 Kapanma sinyali alındı, graceful shutdown başlatılıyor...")

	// =========================================================================
	// 9. GRACEFUL SHUTDOWN PROSEDÜRÜ
	// =========================================================================

	// Shutdown için timeout context'i oluştur (maksimum 30 saniye)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// HTTP sunucusunu gracefully kapat
	// Bu, yeni bağlantıları kabul etmeyi durdurur ve mevcut isteklerin
	// tamamlanmasını bekler (timeout'a kadar)
	logger.Println("⏳ HTTP sunucusu kapatılıyor (aktif istekler tamamlanıyor)...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("⚠️  HTTP sunucusu zorla kapatıldı: %v", err)
	} else {
		logger.Println("✅ HTTP sunucusu gracefully kapatıldı")
	}

	// Database bağlantılarını kapat
	logger.Println("⏳ Database bağlantıları kapatılıyor...")
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	if err := db.Close(); err != nil {
		logger.Printf("⚠️  Database kapatılamadı: %v", err)
	} else {
		logger.Println("✅ Database bağlantıları kapatıldı")
	}

	// TODO: İleride eklenecek cleanup işlemleri:
	// - Redis bağlantılarını kapat
	// - Queue worker'ları durdur
	// - Cache'i flush et
	// - Metrics'leri kaydet

	logger.Println("👋 Uygulama temiz bir şekilde kapatıldı. Hoşça kal!")
}
