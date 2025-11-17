package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/models"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
)

// AppController, temel uygulama endpoint'lerini yönetir.
type AppController struct {
	Logger  *log.Logger
	DB      *sql.DB
	Grammar database.Grammar
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

	return &AppController{
		Logger:  logger,
		DB:      db,
		Grammar: grammar,
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
//
// Bu endpoint kubernetes liveness/readiness probe'ları için kullanılır.
// Database bağlantısını kontrol eder ve sistem durumunu döndürür.
//
// Response Format:
//
//	{
//	  "success": true,
//	  "data": {
//	    "status": "healthy",
//	    "version": "1.0.0",
//	    "uptime": "5m30s",
//	    "database": "connected"
//	  }
//	}
//
// Status Codes:
// - 200: Sistem sağlıklı
// - 503: Database bağlantısı yok (service unavailable)
func (ac *AppController) HealthHandler(w http.ResponseWriter, r *conduitReq.Request) {
	// Database bağlantısını kontrol et
	if err := ac.DB.Ping(); err != nil {
		ac.Logger.Printf("Health check FAILED: Database ping error: %v", err)
		conduitRes.Error(w, http.StatusServiceUnavailable, "Database bağlantısı kurulamadı")
		return
	}

	// Sistem bilgilerini topla
	healthData := map[string]string{
		"status":   "healthy",
		"version":  "1.0.0",
		"database": "connected",
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

	// Test için geçici User struct'ı
	type User struct {
		models.BaseModel
		Name  string `json:"name" db:"name"`
		Email string `json:"email" db:"email"`
	}

	var users []User

	// ORM sorgusu
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
