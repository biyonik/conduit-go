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
	"fmt"
	"log"
	"net/http"
	"time"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/router"
    "github.com/biyonik/conduit-go/internal/middleware"
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
}

// main, uygulamanın çalıştırıldığı başlangıç noktasıdır. Burada HTTP sunucusu
// oluşturulur, route tanımlamaları yapılır ve gerekli konfigürasyonlar
// ayarlanır. Ardından sunucu belirtilen port üzerinden dinlemeye başlar.
func main() {
	app := &Application{
		Name:    "Conduit Go",
		Version: "1.0.3",
	}

	r := router.New()

    r.Use(middleware.CORSMiddleware("*"))
	r.Use(middleware.Logging)

	// Rotalar aynı
	r.Handle("GET /", app.homeHandler)
	r.Handle("GET /api/check", app.checkHandler)

	srv := &http.Server{
		Addr:    ":8000",
		Handler: r,
	}

	fmt.Printf("🚀 %s v%s çalışıyor (Port: 8000)...\n", app.Name, app.Version)
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
