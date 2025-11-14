// internal/middleware/cors.go
package middleware

import (
	"net/http"
)

func CORSMiddleware(allowedOrigin string) Middleware {
	
	return func(next http.Handler) http.Handler {
		
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				
				w.WriteHeader(http.StatusNoContent) 
				return 
			}
			next.ServeHTTP(w, r)
			
		})
	}
}