// -----------------------------------------------------------------------------
// Router Package
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın HTTP yönlendirme (routing) altyapısını oluşturan
// çekirdek yapı taşını barındırır. Laravel veya Symfony gibi modern web
// framework'lerinde gördüğümüz Router katmanının Go dünyasına sade ama güçlü
// bir uyarlaması olarak tasarlanmıştır.
//
// Temel amacı; gelen HTTP isteklerinin doğru handler fonksiyonlarına
// yönlendirilmesi, bu süreçte middleware zincirinin işletilmesi ve uygulamanın
// net/http tabanlı mimarisini daha esnek ve okunabilir bir hale getirmektir.
//
// Bu dosyada yer alan başlıca yapılar:
// - Middleware: Handler öncesi veya sonrası çalışan fonksiyon zinciri.
// - HandlerFunc: conduit-go'nun Request tipini kullanarak çalışan özel handler.
// - Router: Tüm routing yapısını yöneten merkezî sistem.
//
// Router, middleware'leri global olarak uygular; yani bir kez eklenen middleware
// tüm route’lar için geçerlidir. wrapMiddleware fonksiyonu sayesinde middleware
// zinciri ters sırayla sarılır, tıpkı popüler Go framework'lerinde olduğu gibi.
// -----------------------------------------------------------------------------

package router

import (
	"log"
	"net/http"
	"time"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	"github.com/biyonik/conduit-go/internal/middleware"
)

// Middleware, bir http.Handler alıp onu başka bir http.Handler'a dönüştüren
// fonksiyondur. Bu yapı sayesinde istek işlenmeden önce (ya da sonra)
// çalışması istenen işlemler zincir halinde uygulanabilir.
type Middleware func(next http.Handler) http.Handler

// HandlerFunc, Router'ın standart handler tipidir. Gelen isteği conduit
// Request yapısı üzerinden alır ve böylece framework içerisinde daha zengin
// bir istek nesnesi kullanmak mümkün olur.
type HandlerFunc func(w http.ResponseWriter, r *conduitReq.Request)

// Router, uygulamanın tüm route kayıtlarını, middleware zincirini ve HTTP
// yönlendirme akışını yöneten yapıdır. net/http'nin ServeMux yapısını temel alır
// ancak onu daha soyut bir katmanla zenginleştirir.
type Router struct {
	mux         *http.ServeMux          // Standart Go router'ı (ServeMux)
	middlewares []middleware.Middleware // Global middleware listesi
}

// New, yeni bir Router örneği oluşturur. İçerisinde yeni bir ServeMux ve
// boş bir middleware listesi bulunur.
func New() *Router {
	return &Router{
		mux:         http.NewServeMux(),
		middlewares: []middleware.Middleware{},
	}
}

// Use, router'a yeni bir global middleware ekler. Bu middleware tüm handler'lar
// için geçerli olacaktır. Middleware ekleme sırası önemlidir; wrap edildiğinde
// ters sırada uygulanır.
func (rt *Router) Use(mw middleware.Middleware) {
	rt.middlewares = append(rt.middlewares, mw)
}

// Handle, belirtilen route pattern'i ile bir HandlerFunc eşleştirir. Handler
// önce adapt edilerek http.Handler'a dönüştürülür, ardından mevcut tüm
// middleware'ler iç içe sarılır ve son hali ServeMux'a kaydedilir.
func (rt *Router) Handle(pattern string, handler HandlerFunc) {
	stdHandler := rt.adaptHandler(handler)

	wrappedHandler := rt.wrapMiddleware(stdHandler, rt.middlewares...)

	rt.mux.Handle(pattern, wrappedHandler)
}

// adaptHandler, framework'e özel HandlerFunc'i standart http.Handler'a dönüştürür.
// Böylece net/http ekosistemi ile tam uyum sağlanmış olur.
func (rt *Router) adaptHandler(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := conduitReq.New(r) // Standart request özel conduit request'e çevrilir.
		h(w, req)
	})
}

// wrapMiddleware, verilen handler'ı sırayla tüm middleware'lerle sarar.
// Middleware zincirinin doğru şekilde çalışabilmesi için middleware'ler
// tersten uygulanır (LIFO mantığı). Bu yöntem Go topluluğunda standarttır.
func (rt *Router) wrapMiddleware(handler http.Handler, middlewares ...middleware.Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// ServeHTTP, Router'ın http.Handler arayüzünü implemente etmesini sağlar.
// Gelen her istek ServeMux'a yönlendirilir ve ilgili handler tetiklenir.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("-> %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("<- %s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
