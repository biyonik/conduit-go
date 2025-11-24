// -----------------------------------------------------------------------------
// Token Generation Utility
// -----------------------------------------------------------------------------
// This package provides cryptographically secure token generation utilities.
//
// These helpers eliminate duplicate token generation code found in:
//   - Password reset tokens (PasswordController)
//   - CSRF tokens (CSRF middleware)
//   - Session IDs (CSRF middleware)
//   - API keys
//   - Verification tokens
//
// All tokens are generated using crypto/rand for cryptographic security.
// -----------------------------------------------------------------------------

package token

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

// GenerateSecureToken generates a cryptographically secure random token.
//
// Parameters:
//   - length: The length of the random bytes to generate (default: 32)
//
// Returns:
//   - string: Base64 URL-encoded token
//   - error: Error if random number generation fails
//
// The token is base64 URL-encoded, making it safe for use in URLs.
//
// Example:
//
//	token, err := token.GenerateSecureToken(32)
//	if err != nil {
//	    return err
//	}
//	// token is a 32-byte random string, base64 encoded
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		length = 32 // Default to 32 bytes
	}

	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		// Fallback to time-based token if crypto/rand fails
		// This should never happen in practice
		return fallbackToken(length), err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// MustGenerateSecureToken is like GenerateSecureToken but panics on error.
//
// Use this in initialization code where errors should be fatal.
//
// Example:
//
//	token := token.MustGenerateSecureToken(32)
func MustGenerateSecureToken(length int) string {
	token, err := GenerateSecureToken(length)
	if err != nil {
		panic(fmt.Sprintf("failed to generate secure token: %v", err))
	}
	return token
}

// GenerateSecureTokenHex generates a cryptographically secure random token
// encoded as hexadecimal instead of base64.
//
// Parameters:
//   - length: The length of the random bytes to generate (default: 32)
//
// Returns:
//   - string: Hex-encoded token
//   - error: Error if random number generation fails
//
// Hex encoding produces a longer string (2 characters per byte) but may be
// preferred in some contexts.
//
// Example:
//
//	token, err := token.GenerateSecureTokenHex(32)
//	// token is 64 characters (32 bytes * 2)
func GenerateSecureTokenHex(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return fallbackTokenHex(length), err
	}

	return hex.EncodeToString(bytes), nil
}

// GeneratePasswordResetToken generates a token suitable for password reset.
//
// This is a convenience function that generates a 32-byte token, which is
// appropriate for password reset use cases.
//
// Example:
//
//	resetToken, err := token.GeneratePasswordResetToken()
//	if err != nil {
//	    return err
//	}
func GeneratePasswordResetToken() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateCSRFToken generates a token suitable for CSRF protection.
//
// This is a convenience function that generates a 32-byte token, which is
// appropriate for CSRF use cases.
//
// Example:
//
//	csrfToken, err := token.GenerateCSRFToken()
//	if err != nil {
//	    return err
//	}
func GenerateCSRFToken() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateSessionID generates a token suitable for session IDs.
//
// This is a convenience function that generates a 32-byte token, which is
// appropriate for session ID use cases.
//
// Example:
//
//	sessionID, err := token.GenerateSessionID()
//	if err != nil {
//	    return err
//	}
func GenerateSessionID() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateAPIKey generates a token suitable for API keys.
//
// This generates a longer token (48 bytes) for added security in API key scenarios.
//
// Example:
//
//	apiKey, err := token.GenerateAPIKey()
//	if err != nil {
//	    return err
//	}
func GenerateAPIKey() (string, error) {
	return GenerateSecureToken(48)
}

// fallbackToken generates a time-based fallback token when crypto/rand fails.
//
// This should never be used in production as it's not cryptographically secure.
// It's only here as a last resort if the system's random number generator fails.
func fallbackToken(length int) string {
	// Use time-based seed for fallback (NOT cryptographically secure)
	timeBytes := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))

	// Pad or truncate to desired length
	if len(timeBytes) < length {
		padding := make([]byte, length-len(timeBytes))
		timeBytes = append(timeBytes, padding...)
	} else if len(timeBytes) > length {
		timeBytes = timeBytes[:length]
	}

	return base64.URLEncoding.EncodeToString(timeBytes)
}

// fallbackTokenHex is the hex version of fallbackToken.
func fallbackTokenHex(length int) string {
	timeBytes := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))

	if len(timeBytes) < length {
		padding := make([]byte, length-len(timeBytes))
		timeBytes = append(timeBytes, padding...)
	} else if len(timeBytes) > length {
		timeBytes = timeBytes[:length]
	}

	return hex.EncodeToString(timeBytes)
}
