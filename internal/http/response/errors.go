// -----------------------------------------------------------------------------
// Standardized Error Response Helpers
// -----------------------------------------------------------------------------
// This file provides convenient helper functions for common error responses,
// ensuring consistency across all controllers.
//
// Benefits:
//   - Consistent error messages
//   - Consistent HTTP status codes
//   - Reduced boilerplate code
//   - Easier to update error messages globally
// -----------------------------------------------------------------------------

package response

import (
	"net/http"
)

// InvalidJSON sends a 400 Bad Request error for invalid JSON format.
//
// Example:
//
//	if err := r.ParseJSON(&reqData); err != nil {
//	    response.InvalidJSON(w)
//	    return
//	}
func InvalidJSON(w http.ResponseWriter) {
	Error(w, http.StatusBadRequest, "Geçersiz JSON formatı")
}

// InvalidJSONEN is the English version of InvalidJSON.
func InvalidJSONEN(w http.ResponseWriter) {
	Error(w, http.StatusBadRequest, "Invalid JSON format")
}

// ValidationError sends a 422 Unprocessable Entity error with validation errors.
//
// Parameters:
//   - w: HTTP response writer
//   - errors: Map of field names to error messages (e.g., {"email": ["Email is required"]})
//
// Example:
//
//	if result.HasErrors() {
//	    response.ValidationError(w, result.Errors())
//	    return
//	}
func ValidationError(w http.ResponseWriter, errors map[string][]string) {
	Error(w, http.StatusUnprocessableEntity, errors)
}

// FieldError sends a 422 Unprocessable Entity error for a single field.
//
// This is a convenience function for when you need to return a validation
// error for a single field without building the full error map.
//
// Parameters:
//   - w: HTTP response writer
//   - field: Field name (e.g., "email")
//   - message: Error message (e.g., "Email already exists")
//
// Example:
//
//	if emailExists {
//	    response.FieldError(w, "email", "Bu email adresi zaten kullanımda")
//	    return
//	}
func FieldError(w http.ResponseWriter, field string, message string) {
	Error(w, http.StatusUnprocessableEntity, map[string][]string{
		field: {message},
	})
}

// Unauthorized sends a 401 Unauthorized error.
//
// Use this when authentication is required but not provided, or when
// authentication credentials are invalid.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Optional error message (if empty, uses default)
//
// Example:
//
//	if authHeader == "" {
//	    response.Unauthorized(w, "")
//	    return
//	}
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Kimlik doğrulaması gerekli"
	}
	Error(w, http.StatusUnauthorized, message)
}

// UnauthorizedEN is the English version of Unauthorized.
func UnauthorizedEN(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden error.
//
// Use this when the user is authenticated but doesn't have permission
// to access the resource.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Optional error message (if empty, uses default)
//
// Example:
//
//	if !user.IsAdmin() {
//	    response.Forbidden(w, "")
//	    return
//	}
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Bu işlem için yetkiniz yok"
	}
	Error(w, http.StatusForbidden, message)
}

// ForbiddenEN is the English version of Forbidden.
func ForbiddenEN(w http.ResponseWriter, message string) {
	if message == "" {
		message = "You don't have permission to perform this action"
	}
	Error(w, http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found error.
//
// Use this when a requested resource doesn't exist.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Optional error message (if empty, uses default)
//
// Example:
//
//	if err == sql.ErrNoRows {
//	    response.NotFound(w, "")
//	    return
//	}
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Kayıt bulunamadı"
	}
	Error(w, http.StatusNotFound, message)
}

// NotFoundEN is the English version of NotFound.
func NotFoundEN(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource not found"
	}
	Error(w, http.StatusNotFound, message)
}

// ServerError sends a 500 Internal Server Error.
//
// Use this for unexpected server-side errors. Should be used sparingly
// and logged appropriately.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Optional error message (if empty, uses default)
//
// Example:
//
//	if err != nil {
//	    logger.Printf("❌ Database error: %v", err)
//	    response.ServerError(w, "")
//	    return
//	}
func ServerError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Sunucu hatası"
	}
	Error(w, http.StatusInternalServerError, message)
}

// ServerErrorEN is the English version of ServerError.
func ServerErrorEN(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal server error"
	}
	Error(w, http.StatusInternalServerError, message)
}

// BadRequest sends a 400 Bad Request error.
//
// Use this for malformed requests or invalid request parameters.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Error message
//
// Example:
//
//	if id <= 0 {
//	    response.BadRequest(w, "Geçersiz ID")
//	    return
//	}
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Conflict sends a 409 Conflict error.
//
// Use this when there's a conflict with the current state of the resource
// (e.g., trying to create a duplicate resource).
//
// Parameters:
//   - w: HTTP response writer
//   - message: Error message
//
// Example:
//
//	if emailExists {
//	    response.Conflict(w, "Email already in use")
//	    return
//	}
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, message)
}

// TooManyRequests sends a 429 Too Many Requests error.
//
// Use this when rate limiting is exceeded.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Optional error message
//
// Example:
//
//	if rateLimitExceeded {
//	    response.TooManyRequests(w, "")
//	    return
//	}
func TooManyRequests(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Çok fazla istek gönderdiniz. Lütfen daha sonra tekrar deneyin."
	}
	Error(w, http.StatusTooManyRequests, message)
}

// TooManyRequestsEN is the English version of TooManyRequests.
func TooManyRequestsEN(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Too many requests. Please try again later."
	}
	Error(w, http.StatusTooManyRequests, message)
}
