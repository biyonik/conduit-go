// Package router, HTTP isteklerini yönlendirmek ve route tanımlamak için
// Laravel-inspired bir router implementasyonu sağlar.
package router

import (
	"context"
	"net/http"
	"strings"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	"github.com/biyonik/conduit-go/internal/middleware"
)

// HandlerFunc, Conduit-Go framework'ünün handler fonksiyon tipidir.
// Standard http.HandlerFunc'tan farkı, *conduitReq.Request kullanmasıdır.
type HandlerFunc func(http.ResponseWriter, *conduitReq.Request)

// Router, HTTP routing yapısını temsil eder.
type Router struct {
	routes      []*Route
	middlewares []middleware.Middleware
	groups      []*RouteGroup
}

// Route, tek bir HTTP route'unu temsil eder.
type Route struct {
	method      string
	path        string
	handler     HandlerFunc // Artık kendi type'ımız
	middlewares []middleware.Middleware
	router      *Router
}

// RouteGroup, route gruplarını temsil eder.
type RouteGroup struct {
	prefix      string
	middlewares []middleware.Middleware
	router      *Router
}

// New, yeni bir Router instance'ı oluşturur.
func New() *Router {
	return &Router{
		routes:      make([]*Route, 0),
		middlewares: make([]middleware.Middleware, 0),
		groups:      make([]*RouteGroup, 0),
	}
}

// Use, router seviyesinde global middleware ekler.
func (r *Router) Use(middleware middleware.Middleware) {
	r.middlewares = append(r.middlewares, middleware)
}

// GET, GET metodu için route tanımlar ve Route objesi döndürür.
func (r *Router) GET(path string, handler HandlerFunc) *Route {
	return r.addRoute("GET", path, handler)
}

// POST, POST metodu için route tanımlar ve Route objesi döndürür.
func (r *Router) POST(path string, handler HandlerFunc) *Route {
	return r.addRoute("POST", path, handler)
}

// PUT, PUT metodu için route tanımlar ve Route objesi döndürür.
func (r *Router) PUT(path string, handler HandlerFunc) *Route {
	return r.addRoute("PUT", path, handler)
}

// DELETE, DELETE metodu için route tanımlar ve Route objesi döndürür.
func (r *Router) DELETE(path string, handler HandlerFunc) *Route {
	return r.addRoute("DELETE", path, handler)
}

// PATCH, PATCH metodu için route tanımlar ve Route objesi döndürür.
func (r *Router) PATCH(path string, handler HandlerFunc) *Route {
	return r.addRoute("PATCH", path, handler)
}

// OPTIONS, OPTIONS metodu için route tanımlar ve Route objesi döndürür.
func (r *Router) OPTIONS(path string, handler HandlerFunc) *Route {
	return r.addRoute("OPTIONS", path, handler)
}

// addRoute, yeni bir route ekler ve Route objesi döndürür.
func (r *Router) addRoute(method, path string, handler HandlerFunc) *Route {
	route := &Route{
		method:      method,
		path:        path,
		handler:     handler,
		middlewares: make([]middleware.Middleware, 0),
		router:      r,
	}
	r.routes = append(r.routes, route)
	return route
}

// Middleware, route'a middleware ekler (method chaining için).
//
// Kullanım:
//
//	r.GET("/profile", ProfileHandler).
//	    Middleware(middleware.Auth()).
//	    Middleware(middleware.RateLimit(10, 60))
func (route *Route) Middleware(m middleware.Middleware) *Route {
	route.middlewares = append(route.middlewares, m)
	return route
}

// Group, route grubu oluşturur.
//
// Kullanım:
//
//	api := r.Group("/api")
//	api.Use(middleware.Auth())
//	api.GET("/users", UsersHandler)
func (r *Router) Group(prefix string) *RouteGroup {
	group := &RouteGroup{
		prefix:      prefix,
		middlewares: make([]middleware.Middleware, 0),
		router:      r,
	}
	r.groups = append(r.groups, group)
	return group
}

// Use, grup seviyesinde middleware ekler.
func (g *RouteGroup) Use(middleware middleware.Middleware) {
	g.middlewares = append(g.middlewares, middleware)
}

// GET, grup içinde GET route tanımlar.
func (g *RouteGroup) GET(path string, handler HandlerFunc) *Route {
	fullPath := g.prefix + path
	route := g.router.addRoute("GET", fullPath, handler)
	// Grup middleware'lerini route'a ekle
	route.middlewares = append(g.middlewares, route.middlewares...)
	return route
}

// POST, grup içinde POST route tanımlar.
func (g *RouteGroup) POST(path string, handler HandlerFunc) *Route {
	fullPath := g.prefix + path
	route := g.router.addRoute("POST", fullPath, handler)
	route.middlewares = append(g.middlewares, route.middlewares...)
	return route
}

// PUT, grup içinde PUT route tanımlar.
func (g *RouteGroup) PUT(path string, handler HandlerFunc) *Route {
	fullPath := g.prefix + path
	route := g.router.addRoute("PUT", fullPath, handler)
	route.middlewares = append(g.middlewares, route.middlewares...)
	return route
}

// DELETE, grup içinde DELETE route tanımlar.
func (g *RouteGroup) DELETE(path string, handler HandlerFunc) *Route {
	fullPath := g.prefix + path
	route := g.router.addRoute("DELETE", fullPath, handler)
	route.middlewares = append(g.middlewares, route.middlewares...)
	return route
}

// PATCH, grup içinde PATCH route tanımlar.
func (g *RouteGroup) PATCH(path string, handler HandlerFunc) *Route {
	fullPath := g.prefix + path
	route := g.router.addRoute("PATCH", fullPath, handler)
	route.middlewares = append(g.middlewares, route.middlewares...)
	return route
}

// ServeHTTP, http.Handler interface'ini implement eder.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Global middleware'leri uygula
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.handleRequest(w, req)
	})

	// Global middleware chain oluştur (reverse order)
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	handler.ServeHTTP(w, req)
}

// handleRequest, gelen isteği uygun route'a yönlendirir.
func (r *Router) handleRequest(w http.ResponseWriter, req *http.Request) {
	// Route'ları kontrol et
	for _, route := range r.routes {
		if route.method != req.Method {
			continue
		}

		// Route parametrelerini match et
		params, matched := r.matchRoute(route.path, req.URL.Path)
		if !matched {
			continue
		}

		// Route parametrelerini context'e ekle
		ctx := context.WithValue(req.Context(), conduitReq.RequestParamsKey, params)
		req = req.WithContext(ctx)

		// Route-specific middleware'leri uygula
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// http.Request'i conduitReq.Request'e dönüştür
			conduitRequest := conduitReq.New(req)
			// Handler'ı çağır (artık doğru signature)
			route.handler(w, conduitRequest)
		})

		// Route middleware chain oluştur (reverse order)
		for i := len(route.middlewares) - 1; i >= 0; i-- {
			handler = route.middlewares[i](handler)
		}

		handler.ServeHTTP(w, req)
		return
	}

	// 404 Not Found
	http.NotFound(w, req)
}

// matchRoute, route pattern'i ile URL path'ini karşılaştırır.
// Parametreleri extract eder ve match durumunu döndürür.
//
// Pattern örnekleri:
//
//	/users/{id}
//	/posts/{id}/comments/{commentId}
func (r *Router) matchRoute(pattern, path string) (map[string]string, bool) {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// Part sayısı farklıysa match değildir
	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i, part := range patternParts {
		// Parametre mi? (örn: {id})
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := strings.Trim(part, "{}")
			params[paramName] = pathParts[i]
			continue
		}

		// Statik part eşleşmeli
		if part != pathParts[i] {
			return nil, false
		}
	}

	return params, true
}
