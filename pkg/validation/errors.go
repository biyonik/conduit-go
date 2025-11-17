// -----------------------------------------------------------------------------
// Validation Errors
// -----------------------------------------------------------------------------
// Bu dosya, validation error'larını oluşturmak için helper fonksiyonlar içerir.
// -----------------------------------------------------------------------------

package validation

import "fmt"

// FieldError, belirli bir field için validation error'u temsil eder.
type FieldError struct {
	Field   string
	Message string
}

// Error, error interface implementasyonu.
func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewFieldError, yeni bir field error oluşturur.
//
// Parametreler:
//   - field: Hata oluşan field adı
//   - message: Hata mesajı
//
// Döndürür:
//   - error: FieldError
//
// Örnek:
//
//	return validation.NewFieldError("password_confirm", "Şifreler eşleşmiyor")
func NewFieldError(field, message string) error {
	return &FieldError{
		Field:   field,
		Message: message,
	}
}
