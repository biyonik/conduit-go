// cmd/api/main.go
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/internal/controllers"
	"github.com/biyonik/conduit-go/internal/middleware"
	"github.com/biyonik/conduit-go/internal/router"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
)

func main() {
	// 1. Konteyneri Başlat
	c := container.New()

	// 2. Servisleri Konteynere Kaydet (Provider'ları kullanarak)

	// Config
	c.Register(func(c *container.Container) (*config.Config, error) {
		return config.Load(), nil
	})

	// Logger
	c.Register(func(c *container.Container) (*log.Logger, error) {
		return log.New(os.Stdout, "", log.Ldate|log.Ltime), nil
	})

	// Veritabanı Bağlantısı (*sql.DB)
	c.Register(func(c *container.Container) (*sql.DB, error) {
		cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
		db, err := database.Connect(cfg.DB.DSN)
		if err != nil {
			return nil, err
		}
		// TODO: main.go'da defer db.Close() vardı,
		// uygulamanın kapanışını (graceful shutdown) dinleyip orada kapatmak lazım.
		return db, nil
	})

	// SQL Grammar
	c.Register(func(c *container.Container) (database.Grammar, error) {
		// Şimdilik sadece MySQL
		// İleride config'den okunabilir.
		return database.NewMySQLGrammar(), nil
	})

	// Controller'lar
	c.Register(controllers.NewAppController)

	// 3. Rota ve Sunucuyu Başlat

	// Gerekli servisleri önceden çöz (resolve)
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
	appController := c.MustGet(reflect.TypeOf((*controllers.AppController)(nil))).(*controllers.AppController)

	// Router'ı oluştur
	r := router.New()

	// Middleware'leri kaydet
	r.Use(middleware.PanicRecovery(logger)) // Adım 1.C'den
	r.Use(middleware.CORSMiddleware("*"))
	r.Use(middleware.Logging)

	// Rotaları tanımla (Artık Controller metotlarını bağlıyoruz)
	r.GET("/", appController.HomeHandler)
	r.GET("/api/check", appController.CheckHandler)
	r.GET("/api/testquery", appController.TestQueryHandler)

	// (Yeni controller'lar eklendikçe buraya eklenecek)
	// Örn:
	// c.Register(controllers.NewUserController)
	// userController := c.MustGet(reflect.TypeOf((*controllers.UserController)(nil))).(*controllers.UserController)
	// r.GET("/api/users", userController.ListUsers)
	// r.GET("/api/users/{id}", userController.GetUser)

	// Sunucuyu başlat
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	logger.Printf("🚀 Conduit Go (DI) çalışıyor (Port: %s, Ortam: %s)...",
		cfg.Server.Port, cfg.App.Env)

	log.Fatal(srv.ListenAndServe())
}
