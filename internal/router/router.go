// ============================================================================
// Router - Trie Tabanlı, Route Group Destekli HTTP Yönlendirme Sistemi
// ----------------------------------------------------------------------------
// Bu dosya, framework'ün HTTP routing altyapısını yönetir. Laravel benzeri bir
// route deneyimi sunmak için aşağıdaki güçlü özellikler sağlanır:
//
//   • Trie tabanlı ultra hızlı rota eşleşmesi
//   • Statik + parametrik path desteği (/users/{id})
//   • Rota başına middleware
//   • Global middleware
//   • Route Group (prefix + middleware birleştirme)
//   • HTTP Method bazlı ayrılmış routing ağaçları
//
// Bu dosya, performans, okunabilirlik ve genişletilebilirlik dikkate alınarak
// yazılmıştır. Tüm routing davranışı tek merkezden kontrol edilir.
// ============================================================================
//
// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik
// ============================================================================

package router

import (
	"context"
	"net/http"
	"strings"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	"github.com/biyonik/conduit-go/internal/middleware"
)

// Middleware: Standart middleware fonksiyon tipi
type Middleware func(next http.Handler) http.Handler

// HandlerFunc: framework'e özel handler tipi
type HandlerFunc func(w http.ResponseWriter, r *conduitReq.Request)

// Trie düğümü
type node struct {
	pathPart string
	isParam  bool

	handler http.Handler

	children   map[string]*node
	paramChild *node
}

// Router: Ana yönlendirici yapısı
type Router struct {
	trees           map[string]*node
	middlewares     []middleware.Middleware
	NotFoundHandler http.Handler
}

// New, boş bir router oluşturur
func New() *Router {
	return &Router{
		trees:           make(map[string]*node),
		middlewares:     []middleware.Middleware{},
		NotFoundHandler: http.NotFoundHandler(),
	}
}

// Group, router seviyesinde yeni bir route group başlatır.
//
//	api := router.Group("/api")
//	api.GET("/users", ...)
func (rt *Router) Group(prefix string) *Group {
	return &Group{
		router:      rt,
		prefix:      prefix,
		middlewares: []middleware.Middleware{},
	}
}

// Use, global middleware ekler
func (rt *Router) Use(mw middleware.Middleware) {
	rt.middlewares = append(rt.middlewares, mw)
}

// splitPath: "/api/users" → ["api","users"]
func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ============================================================================
// Rota ekleme (normal Handle)
// ============================================================================

func (rt *Router) Handle(method, path string, handler HandlerFunc) {
	rt.HandleWithGroup(method, path, handler, []middleware.Middleware{})
}

// HandleWithGroup: grup middleware'leriyle birlikte ekleme
func (rt *Router) HandleWithGroup(method, path string, handler HandlerFunc, groupMiddleware []middleware.Middleware) {

	if rt.trees[method] == nil {
		rt.trees[method] = &node{pathPart: "/", children: make(map[string]*node)}
	}

	currentNode := rt.trees[method]
	parts := splitPath(path)

	for _, part := range parts {
		var child *node
		isParam := part[0] == '{' && part[len(part)-1] == '}'

		if isParam {
			if currentNode.paramChild == nil {
				currentNode.paramChild = &node{
					pathPart: part,
					isParam:  true,
					children: make(map[string]*node),
				}
			}
			child = currentNode.paramChild
		} else {
			if currentNode.children[part] == nil {
				currentNode.children[part] = &node{
					pathPart: part,
					children: make(map[string]*node),
				}
			}
			child = currentNode.children[part]
		}
		currentNode = child
	}

	adapted := rt.adaptHandler(handler)

	// Bu rotaya özel middleware zinciri: global + group
	final := rt.wrapMiddleware(adapted, append(rt.middlewares, groupMiddleware...)...)

	currentNode.handler = final
}

// ============================================================================
// Rota bulma (ServeHTTP)
// ============================================================================

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	root := rt.trees[r.Method]
	if root == nil {
		rt.NotFoundHandler.ServeHTTP(w, r)
		return
	}

	pathParts := splitPath(r.URL.Path)
	currentNode := root
	params := make(map[string]string)

	for _, part := range pathParts {
		child, found := currentNode.children[part]
		if found {
			currentNode = child
		} else if currentNode.paramChild != nil {
			currentNode = currentNode.paramChild
			paramName := currentNode.pathPart[1 : len(currentNode.pathPart)-1]
			params[paramName] = part
		} else {
			rt.NotFoundHandler.ServeHTTP(w, r)
			return
		}
	}

	if currentNode.handler == nil {
		rt.NotFoundHandler.ServeHTTP(w, r)
		return
	}

	if len(params) > 0 {
		ctx := context.WithValue(r.Context(), conduitReq.RequestParamsKey, params)
		r = r.WithContext(ctx)
	}

	currentNode.handler.ServeHTTP(w, r)
}

// adaptHandler, HandlerFunc → http.Handler dönüştürür
func (rt *Router) adaptHandler(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := conduitReq.New(r)
		h(w, req)
	})
}

// wrapMiddleware, handler'ı middleware zinciriyle sarar (LIFO)
func (rt *Router) wrapMiddleware(handler http.Handler, middlewares ...middleware.Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Kolaylık metotları
func (rt *Router) GET(path string, h HandlerFunc)    { rt.Handle("GET", path, h) }
func (rt *Router) POST(path string, h HandlerFunc)   { rt.Handle("POST", path, h) }
func (rt *Router) PUT(path string, h HandlerFunc)    { rt.Handle("PUT", path, h) }
func (rt *Router) DELETE(path string, h HandlerFunc) { rt.Handle("DELETE", path, h) }
