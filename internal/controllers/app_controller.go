package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/models"
	"github.com/biyonik/conduit-go/pkg/cache"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
)

// AppController, temel uygulama endpoint'lerini yönetir.
type AppController struct {
	Logger  *log.Logger
	DB      *sql.DB
	Grammar database.Grammar
	Config  *config.Config
	Cache   cache.Cache // Phase 3
	AppName string
}

// NewDB, yeni bir QueryBuilder başlatır.
func (ac *AppController) NewDB() *database.QueryBuilder {
	return database.NewBuilder(ac.DB, ac.Grammar)
}

// NewAppController, DI Container için fabrika fonksiyonu.
func NewAppController(c *container.Container) (*AppController, error) {
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	grammarType := reflect.TypeOf((*database.Grammar)(nil)).Elem()
	grammar := c.MustGet(grammarType).(database.Grammar)
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
	cacheDriver := c.MustGet(reflect.TypeOf((*cache.Cache)(nil)).Elem()).(cache.Cache)

	return &AppController{
		Logger:  logger,
		DB:      db,
		Grammar: grammar,
		Config:  cfg,
		Cache:   cacheDriver,
		AppName: "Conduit Go",
	}, nil
}

// HomeHandler, ana sayfa handler'ı.
func (ac *AppController) HomeHandler(w http.ResponseWriter, r *conduitReq.Request) {
	if r.IsJSON() {
		conduitRes.Success(w, 200, "JSON istediniz, JSON geldi!", nil)
		return
	}
	fmt.Fprintf(w, "Merhaba! Burası %s, Adres: %s", ac.AppName, r.URL.Path)
}

// HealthHandler, sistem sağlık kontrolü endpoint'i.
func (ac *AppController) HealthHandler(w http.ResponseWriter, r *conduitReq.Request) {
	healthData := map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0-phase3",
		"env":     ac.Config.App.Env,
	}

	// Database check
	if err := ac.DB.Ping(); err != nil {
		ac.Logger.Printf("Health check FAILED: Database ping error: %v", err)
		healthData["database"] = "error"
		conduitRes.Error(w, http.StatusServiceUnavailable, "Database bağlantısı kurulamadı")
		return
	}
	healthData["database"] = "connected"

	// Cache check
	healthData["cache_driver"] = ac.Config.Cache.Driver
	testKey := "health:check:" + time.Now().Format("20060102150405")
	if err := ac.Cache.Set(testKey, "ok", 1*time.Minute); err != nil {
		healthData["cache"] = "error"
	} else {
		healthData["cache"] = "ok"
		ac.Cache.Delete(testKey)
	}

	conduitRes.Success(w, 200, healthData, nil)
}

// CheckHandler, Bearer token kontrolü yapar.
func (ac *AppController) CheckHandler(w http.ResponseWriter, r *conduitReq.Request) {
	token := r.BearerToken()
	if token == "" {
		conduitRes.Error(w, 401, "Kimliksiz gezgin! Bearer token nerede?")
		return
	}
	conduitRes.Success(
		w,
		200,
		fmt.Sprintf("Giriş izni verildi. Token: %s", token),
		map[string]string{"timestamp": time.Now().Format(time.RFC3339)},
	)
}

// TestQueryHandler, ORM .Get() metodunu test eder.
func (ac *AppController) TestQueryHandler(w http.ResponseWriter, r *conduitReq.Request) {
	ac.Logger.Println("Query Builder (ORM .Get()) testi başladı...")

	type User struct {
		models.BaseModel
		Name  string `json:"name" db:"name"`
		Email string `json:"email" db:"email"`
	}

	var users []User

	err := ac.NewDB().
		Table("users").
		Select("id", "name", "email", "created_at", "updated_at").
		Where("status", "=", "active").
		Get(&users)

	if err != nil {
		ac.Logger.Printf("Sorgu hatası: %v", err)
		conduitRes.Error(w, 500, fmt.Sprintf("Sorgu hatası: %v", err))
		return
	}

	_ = conduitRes.Success(w, 200, users, map[string]int{"count": len(users)})
}
