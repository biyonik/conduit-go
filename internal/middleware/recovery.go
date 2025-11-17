package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// PanicRecovery, bir handler'da panic oluştuğunda sunucunun çökmesini engeller
// ve istemciye standart bir JSON 500 hatası döndürür.
func PanicRecovery(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {

					logger.Printf("PANIC: %v\n%s", err, debug.Stack())

					response.Error(w, http.StatusInternalServerError, "Sunucuda beklenmedik bir hata oluştu")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
