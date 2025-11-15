// ============================================================================
// Route Group Sistemi - Laravel Tarzı Routing Hiyerarşisi
// ----------------------------------------------------------------------------
// Bu dosya, Router yapısına "Route Group" yeteneği kazandırır. Böylece
// geliştirici, birden fazla rotayı tek bir prefix veya middleware grubu altında
// toplamak için aşağıdaki gibi bir yapı kullanabilir:
//
//   api := router.Group("/api")
//   api.GET("/users", UserHandler)
//   api.POST("/login", LoginHandler)
//
// Ayrıca nested (iç içe) gruplar tamamen desteklenir:
//
//   v1 := api.Group("/v1")
//   v1.GET("/articles", ArticleList)
//   v1.GET("/articles/{id}", ShowArticle)
//
// Böylece URL derlenirken prefix'ler doğru şekilde birleştirilir:
//   /api/v1/articles
//
// Bu yapı tamamen Go'nun minimalist HTTP yaklaşımıyla uyumludur.
// ============================================================================
//
// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik
// ============================================================================

package router

import (
	"github.com/biyonik/conduit-go/internal/middleware"
)

// Group, bir routing grubu temsil eder. Router ile aynı metotları taşır fakat
// her eklenen rota otomatik olarak prefix + middleware birleşimine dahil olur.
type Group struct {
	router      *Router
	prefix      string
	middlewares []middleware.Middleware
}

// Group, mevcut grubun altına yeni bir grup oluşturur.
//
// Parametre:
//   - prefix: Ek grup yolu (/api, /v1 gibi)
//
// Notlar:
//   - Nested group desteklenir
//   - Middleware birikimli çalışır
func (g *Group) Group(prefix string) *Group {
	return &Group{
		router:      g.router,
		prefix:      g.prefix + prefix,
		middlewares: append([]middleware.Middleware{}, g.middlewares...),
	}
}

// Use, grup özelinde middleware ekler.
// Bu middleware, grup altındaki tüm rotalara uygulanacaktır.
func (g *Group) Use(mw middleware.Middleware) {
	g.middlewares = append(g.middlewares, mw)
}

// --- Yardımcı Router Metotlarının Grup Versiyonları ---

func (g *Group) GET(path string, handler HandlerFunc) {
	g.router.HandleWithGroup("GET", g.prefix+path, handler, g.middlewares)
}

func (g *Group) POST(path string, handler HandlerFunc) {
	g.router.HandleWithGroup("POST", g.prefix+path, handler, g.middlewares)
}

func (g *Group) PUT(path string, handler HandlerFunc) {
	g.router.HandleWithGroup("PUT", g.prefix+path, handler, g.middlewares)
}

func (g *Group) DELETE(path string, handler HandlerFunc) {
	g.router.HandleWithGroup("DELETE", g.prefix+path, handler, g.middlewares)
}
