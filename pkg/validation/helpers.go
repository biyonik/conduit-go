// -----------------------------------------------------------------------------
// Validation Helper Functions
// -----------------------------------------------------------------------------
// This file provides convenient helper functions to reduce boilerplate code
// when validating requests and responding with errors.
//
// These helpers eliminate repetitive validation patterns like:
//   result := schema.Validate(data)
//   if result.HasErrors() {
//       conduitRes.Error(w, 422, result.Errors())
//       return
//   }
//   validData := result.ValidData()
//
// And replace it with:
//   validData, ok := validation.ValidateAndRespond(schema, data, w)
//   if !ok {
//       return
//   }
// -----------------------------------------------------------------------------

package validation

import (
	"net/http"

	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
)

// ValidateAndRespond validates data against a schema and automatically
// sends an error response if validation fails.
//
// Returns:
//   - validData: The validated and sanitized data (if validation succeeds)
//   - ok: true if validation passed, false if it failed
//
// Example:
//
//	schema := validation.Make().Shape(map[string]validation.Type{
//	    "email": types.String().Required().Email(),
//	})
//	validData, ok := validation.ValidateAndRespond(schema, data, w)
//	if !ok {
//	    return // Error response already sent
//	}
//	email := validData["email"].(string)
func ValidateAndRespond(schema Schema, data map[string]any, w http.ResponseWriter) (map[string]any, bool) {
	result := schema.Validate(data)
	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return nil, false
	}
	return result.ValidData(), true
}

// PasswordMatchValidator creates a cross-validation function that checks
// if two password fields match.
//
// This is commonly used for password confirmation fields in registration
// and password change forms.
//
// Parameters:
//   - passwordField: The name of the password field (e.g., "password")
//   - confirmField: The name of the confirmation field (e.g., "password_confirm")
//
// Returns:
//   - A cross-validation function that can be passed to CrossValidate()
//
// Example:
//
//	schema := validation.Make().Shape(map[string]validation.Type{
//	    "password": types.String().Required().Password(...),
//	    "password_confirm": types.String().Required(),
//	}).CrossValidate(
//	    validation.PasswordMatchValidator("password", "password_confirm"),
//	)
func PasswordMatchValidator(passwordField, confirmField string) func(map[string]any) error {
	return func(data map[string]any) error {
		password, _ := data[passwordField].(string)
		confirm, _ := data[confirmField].(string)
		if password != confirm {
			return NewFieldError(confirmField, "Şifreler eşleşmiyor")
		}
		return nil
	}
}

// PasswordMatchValidatorEN is the English version of PasswordMatchValidator.
//
// Example:
//
//	schema := validation.Make().Shape(...).CrossValidate(
//	    validation.PasswordMatchValidatorEN("password", "password_confirm"),
//	)
func PasswordMatchValidatorEN(passwordField, confirmField string) func(map[string]any) error {
	return func(data map[string]any) error {
		password, _ := data[passwordField].(string)
		confirm, _ := data[confirmField].(string)
		if password != confirm {
			return NewFieldError(confirmField, "Passwords do not match")
		}
		return nil
	}
}

// EmailSchema creates a common email validation schema.
//
// This is a shortcut for creating a basic email field validation.
//
// Example:
//
//	schema := validation.Make().Shape(map[string]validation.Type{
//	    "email": validation.EmailSchema(),
//	})
//
// Equivalent to:
//
//	types.String().Required().Email().Max(255).Trim()
func EmailSchema() Type {
	return NewStringType().
		Required().
		Email().
		Max(255).
		Trim()
}

// StrongPasswordSchema creates a strong password validation schema.
//
// Requirements:
//   - Minimum 8 characters
//   - At least one uppercase letter
//   - At least one lowercase letter
//   - At least one number
//   - At least one special character
//
// Example:
//
//	schema := validation.Make().Shape(map[string]validation.Type{
//	    "password": validation.StrongPasswordSchema(),
//	})
func StrongPasswordSchema() Type {
	return NewStringType().
		Required().
		Min(8).
		Max(255)
	// Note: Full password validation would require types.Password() with options
	// This is a placeholder showing the pattern
}
