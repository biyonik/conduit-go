// internal/controllers/app_controller.go
package controllers

import (
	"database/sql" // <-- EKLENDİ
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

// AppController, artık DB ve Grammar'ı tutuyor.
type AppController struct {
	Logger  *log.Logger
	DB      *sql.DB // <-- EKLENDİ (QueryExecutor olarak)
	Grammar database.Grammar
	AppName string
}

// NewDB: *sql.DB'yi ve Grammar'ı kullanarak
// yeni bir QueryBuilder başlatan bir helper fonksiyon.
func (ac *AppController) NewDB() *database.QueryBuilder {
	// 'DB' (QueryExecutor) ve 'Grammar'ı kullanarak builder'ı oluşturur
	return database.NewBuilder(ac.DB, ac.Grammar)
}

// NewAppController, DI Konteyneri için fabrika fonksiyonu.
func NewAppController(c *container.Container) (*AppController, error) {
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	grammarType := reflect.TypeOf((*database.Grammar)(nil)).Elem()
	grammar := c.MustGet(grammarType).(database.Grammar)

	// --- YENİ EKLENEN ---
	// *sql.DB havuzunu (pool) konteynerden çöz
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	// --- YENİ EKLENEN SONU ---

	return &AppController{
		Logger:  logger,
		DB:      db, // <-- EKLENDİ
		Grammar: grammar,
		AppName: "Conduit Go (DI)",
	}, nil
}

func (ac *AppController) HomeHandler(w http.ResponseWriter, r *conduitReq.Request) {
	if r.IsJSON() {
		conduitRes.Success(w, 200, "JSON istediniz, JSON geldi!", nil)
		return
	}
	fmt.Fprintf(w, "Merhaba! Burası %s, Adres: %s", ac.AppName, r.URL.Path)
}

// CheckHandler, 'main.go'dan taşındı.
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

// TestQueryHandler, ARTIK GERÇEK BİR SORGULAMA YAPABİLİR.
// (Şimdilik .ToSQL() ile devam ediyoruz, .Get() metodunu ekleyince burayı tekrar güncelleyeceğiz)
func (ac *AppController) TestQueryHandler(w http.ResponseWriter, r *conduitReq.Request) {
	ac.Logger.Println("Query Builder (ORM .Get()) testi başladı...")

	// Test için geçici bir 'User' struct'ı tanımlayalım
	// (Normalde bu 'internal/models/user.go'da olurdu)
	type User struct {
		models.BaseModel        // Gömülü struct
		Name             string `json:"name" db:"name"`
		Email            string `json:"email" db:"email"`
		// Not: 'status' alanı struct'ta yok, 'db' tag'i yoksa
		// scanner.go'daki mantık onu atlayacak.
	}

	// Sonuçları dolduracağımız slice'ı hazırla
	var users []User

	// Yeni ORM metodunu çağır!
	err := ac.NewDB().
		Table("users").
		Select("id", "name", "email", "created_at", "updated_at"). // 'db' tag'leri ile eşleşmeli
		Where("status", "=", "active").
		Get(&users) // Adresini (&) yolla

	if err != nil {
		ac.Logger.Printf("Sorgu hatası: %v", err)
		conduitRes.Error(w, 500, fmt.Sprintf("Sorgu hatası: %v", err))
		return
	}

	_ = conduitRes.Success(w, 200, users, map[string]int{"count": len(users)})
}
