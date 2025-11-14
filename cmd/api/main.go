// Bu Go dosyası, Conduit Go adında küçük, modüler ve genişletilebilir bir
// HTTP sunucusunun giriş noktasını (entrypoint) temsil eder. Dosya genel
// yapısı itibarıyla Laravel ve Symfony gibi frameworklerde görülen
// "kernel + middleware + controller" mimarisinin sadeleştirilmiş bir
// Go uyarlamasını andırır.
//
// Amaç: Paket içi request/response katmanlarıyla çalışan, okunabilirliği yüksek,
// anlaşılır ve profesyonel bir servis mimarisi oluşturmaktır. Uygulama hem
// gelen istekleri işlemek hem de belirli yardımcı fonksiyonlarla (IsJSON,
// BearerToken vb.) daha düzenli bir API deneyimi sunmak için yapılandırılmıştır.
//
// Bu dosyada:
//   - Uygulamanın metadata bilgilerini tutan Application yapısı,
//   - HTTP handler'larına otomatik olarak geliştirilmiş Request modelini ileten
//     conduitHandler wrapper fonksiyonu,
//   - Ana HTTP sunucusunu çalıştıran main fonksiyonu,
//   - Örnek iki endpoint: homeHandler ve checkHandler bulunmaktadır.
//
// Tüm fonksiyon ve yapılar, profesyonel seviyede açıklamalarla
// detaylandırılmıştır.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/middleware"
	"github.com/biyonik/conduit-go/internal/router"
	"github.com/biyonik/conduit-go/pkg/database"
)

// Application yapısı, uygulamanın temel meta bilgilerini saklayan küçük bir
// konteynırdır. Bu bilgi genellikle loglama, izleme veya yanıt çıktılarında
// kullanılabilir.
//
// Alanlar:
//   - Name: Uygulamanın adı.
//   - Version: Uygulamanın versiyon numarası.
type Application struct {
	Name    string
	Version string
	Config  *config.Config
	DB      *sql.DB
	Logger  *log.Logger
	Grammar database.Grammar
}

// NewDB: *sql.DB'yi ve varsayılan Grammar'ı kullanarak
// yeni bir QueryBuilder başlatan bir helper fonksiyon.
func (app *Application) NewDB() *database.QueryBuilder {
	return database.NewBuilder(app.DB, app.Grammar)
}

// main, uygulamanın çalıştırıldığı başlangıç noktasıdır. Burada HTTP sunucusu
// oluşturulur, route tanımlamaları yapılır ve gerekli konfigürasyonlar
// ayarlanır. Ardından sunucu belirtilen port üzerinden dinlemeye başlar.
func main() {
	cfg := config.Load()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := database.Connect(cfg.DB.DSN)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	// Application (DI Container) GÜNCELLENDİ
	app := &Application{
		Name:    "Conduit Go",
		Version: "1.0.7", // Sürüm atladık
		Config:  cfg,
		DB:      db,
		Logger:  logger,
		Grammar: database.NewMySQLGrammar(), // <-- Lehçeyi burada belirliyoruz!
	}

	r := router.New()

	r.Use(middleware.CORSMiddleware("*"))
	r.Use(middleware.Logging)

	r.Handle("GET /", app.homeHandler)
	r.Handle("GET /api/check", app.checkHandler)
	r.Handle("GET /api/testquery", app.testQueryHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	logger.Printf("🚀 %s v%s çalışıyor (Port: %s, Ortam: %s)...",
		app.Name, app.Version, cfg.Server.Port, cfg.App.Env)

	log.Fatal(srv.ListenAndServe())
}

// conduitHandler, gelen HTTP isteklerini uygulamanın geliştirilmiş Request
// yapısına otomatik dönüştüren bir wrapper (ara katman) fonksiyonudur.
// Bir tür middleware görevi görür.
//
// Böylece tüm handler fonksiyonları *http.Request yerine *conduitReq.Request
// kullanabilir, dolayısıyla daha zengin fonksiyonlara doğrudan erişebilir.
//
// Parametre:
//   - h: İşlenmiş Request yapısıyla çalışan gerçek handler fonksiyonu.
//
// Döndürür:
//   - http.HandlerFunc: Standart Go handler formatında fonksiyon.
func (app *Application) conduitHandler(h func(http.ResponseWriter, *conduitReq.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := conduitReq.New(r) // standart request → genişletilmiş model

		h(w, req)
	}
}

// homeHandler, uygulamanın ana sayfa endpoint'idir. Kullanıcının JSON
// isteyip istemediğine göre iki farklı türde yanıt üretir.
//
// Davranış:
//   - Eğer Content-Type: application/json ise → JSON bir başarı yanıtı döndürür.
//   - Değilse → Basit bir metin yanıtı döndürür.
//
// Parametreler:
//   - w: Yanıt yazıcısı
//   - r: Geliştirilmiş Request modeli
func (app *Application) homeHandler(w http.ResponseWriter, r *conduitReq.Request) {
	if r.IsJSON() {
		conduitRes.Success(w, 200, "JSON istediniz, JSON geldi!", nil)
		return
	}

	fmt.Fprintf(w, "Merhaba! Burası %s, Adres: %s", app.Name, r.URL.Path)
}

// checkHandler, Bearer Token doğrulaması yapan küçük bir güvenlik örneği
// endpoint'idir.
//
// Davranış:
//  1. Bearer token okunur.
//  2. Token yoksa → 401 Unauthorized döndürülür.
//  3. Token varsa → Başarılı yanıt + meta veri döndürülür.
//
// Meta örneği olarak zaman damgası (timestamp) eklenmiştir.
func (app *Application) checkHandler(w http.ResponseWriter, r *conduitReq.Request) {
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

// testQueryHandler (Refaktörden sonra hala çalışıyor olmalı)
func (app *Application) testQueryHandler(w http.ResponseWriter, r *conduitReq.Request) {
	app.Logger.Println("Query Builder (Interface'li) testi başladı...")

	// app.NewDB() helper'ı artık bize *doğru* builder'ı veriyor.
	qb := app.NewDB().
		Table("users").
		Select("id", "name").
		Where("status", "=", "active").
		Limit(1)

	sql, args := qb.ToSQL()

	data := map[string]interface{}{
		"message":       "Go Query Builder (Grammar ile) testi başarılı!",
		"generated_sql": sql,
		"arguments":     args,
	}

	conduitRes.Success(w, 200, data, nil)
}
