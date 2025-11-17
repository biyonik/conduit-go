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
// Application Entry Point (Phase 2: Authentication & Authorization)
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın başlangıç noktasıdır. Phase 2'de authentication ve
// authorization özellikleri eklenmiştir.
//
// YENİ ÖZELLİKLER:
// - JWT-based authentication
// - User registration & login
// - Password reset
// - Role-based authorization
// - Protected routes
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

	// Controller'lar
	c.Register(controllers.NewAppController)
	c.Register(controllers.NewAuthController)     // YENİ: Auth Controller
	c.Register(controllers.NewPasswordController) // YENİ: Password Controller

	// =========================================================================
	// 3. GEREKLI SERVİSLERİ RESOLVE ET
	// =========================================================================
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
	appController := c.MustGet(reflect.TypeOf((*controllers.AppController)(nil))).(*controllers.AppController)
	authController := c.MustGet(reflect.TypeOf((*controllers.AuthController)(nil))).(*controllers.AuthController)
	passwordController := c.MustGet(reflect.TypeOf((*controllers.PasswordController)(nil))).(*controllers.PasswordController)

	// =========================================================================
	// 4. ROUTER'I OLUŞTUR VE MIDDLEWARE'LERI KAYDET
	// =========================================================================
	r := router.New()

	// Global Middleware'ler (Sıralama önemli!)
	r.Use(middleware.PanicRecovery(logger)) // 1. Panic yakalama
	r.Use(middleware.Logging)               // 2. Request logging
	r.Use(middleware.CORSMiddleware("*"))   // 3. CORS
	r.Use(middleware.RateLimit(100, 60))    // 4. Rate limiting: 100 req/min

	// =========================================================================
	// 5. PUBLIC ROTALARI TANIMLA
	// =========================================================================

	// Genel endpoint'ler
	r.GET("/", appController.HomeHandler)
	r.GET("/health", appController.HealthHandler)

	// =========================================================================
	// 6. AUTH ROTALARI (PUBLIC - Authentication gerektirmez)
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
	// 7. PROTECTED ROTALARI TANIMLA (Authentication gerekir)
	// =========================================================================

	// Authenticated user endpoint'leri
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
	// 8. API V1 ROUTES (Authenticated + Stricter Limits)
	// =========================================================================
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.Auth())            // Tüm API endpoint'leri protected
	apiV1.Use(middleware.RateLimit(50, 60)) // API için daha sıkı limit: 50 req/min

	// Test endpoint (authenticated)
	apiV1.GET("/check", appController.CheckHandler)
	apiV1.GET("/testquery", appController.TestQueryHandler)

	// =========================================================================
	// 9. ADMIN ROTALARI (Sadece admin'ler erişebilir)
	// =========================================================================
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(middleware.Auth())            // Authentication gerekli
	adminGroup.Use(middleware.Admin())           // Admin role gerekli
	adminGroup.Use(middleware.RateLimit(30, 60)) // Admin için limit: 30 req/min

	// Admin endpoint'leri (Phase 3'te eklenecek)
	// adminGroup.GET("/users", adminController.ListUsers)
	// adminGroup.DELETE("/users/{id}", adminController.DeleteUser)

	// =========================================================================
	// 10. HTTP SUNUCUSUNU YAPILANDΙR
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
	// 11. SUNUCUYU GOROUTINE'DE BAŞLAT
	// =========================================================================
	go func() {
		logger.Printf("🚀 Conduit Go çalışıyor (Port: %s, Ortam: %s)...", cfg.Server.Port, cfg.App.Env)
		logger.Printf("📍 Health Check: http://localhost:%s/health", cfg.Server.Port)
		logger.Printf("🔐 Auth Endpoints:")
		logger.Printf("   - POST /api/auth/register")
		logger.Printf("   - POST /api/auth/login")
		logger.Printf("   - POST /api/auth/logout (protected)")
		logger.Printf("   - GET  /api/auth/profile (protected)")
		logger.Printf("   - POST /api/auth/forgot-password")
		logger.Printf("   - POST /api/auth/reset-password")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("❌ Sunucu başlatılamadı: %v", err)
		}
	}()

	// =========================================================================
	// 12. GRACEFUL SHUTDOWN
	// =========================================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Println("🛑 Kapanma sinyali alındı, graceful shutdown başlatılıyor...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Println("⏳ HTTP sunucusu kapatılıyor...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("⚠️  HTTP sunucusu zorla kapatıldı: %v", err)
	} else {
		logger.Println("✅ HTTP sunucusu gracefully kapatıldı")
	}

	logger.Println("⏳ Database bağlantıları kapatılıyor...")
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	if err := db.Close(); err != nil {
		logger.Printf("⚠️  Database kapatılamadı: %v", err)
	} else {
		logger.Println("✅ Database bağlantıları kapatıldı")
	}

	logger.Println("👋 Uygulama temiz bir şekilde kapatıldı. Hoşça kal!")
}
