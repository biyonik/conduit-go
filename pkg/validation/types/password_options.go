// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu dosya, parola doğrulama kuralları için opsiyon fonksiyonlarını içerir.
package types

import "github.com/biyonik/conduit-go/pkg/validation/rules"

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// PasswordOption, PasswordRules struct'ını yapılandırmak için kullanılan fonksiyon tipidir.
type PasswordOption func(*rules.PasswordRules)

// WithMinLength, parola için minimum uzunluğu ayarlar.
func WithMinLength(length int) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.MinLength = length
	}
}

// WithMaxLength, parola için maksimum uzunluğu ayarlar.
func WithMaxLength(length int) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.MaxLength = length
	}
}

// WithRequireUppercase, parolanın büyük harf içermesini zorunlu kılar.
func WithRequireUppercase(required bool) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.RequireUppercase = required
	}
}

// WithRequireLowercase, parolanın küçük harf içermesini zorunlu kılar.
func WithRequireLowercase(required bool) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.RequireLowercase = required
	}
}

// WithRequireNumeric, parolanın rakam içermesini zorunlu kılar.
func WithRequireNumeric(required bool) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.RequireNumeric = required
	}
}

// WithRequireSpecial, parolanın özel karakter içermesini zorunlu kılar.
func WithRequireSpecial(required bool) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.RequireSpecial = required
	}
}

// WithSpecialChars, izin verilen özel karakter setini belirler.
func WithSpecialChars(chars string) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.SpecialChars = chars
	}
}

// WithMinUniqueChars, paroladaki minimum eşsiz karakter sayısını ayarlar.
func WithMinUniqueChars(count int) PasswordOption {
	return func(r *rules.PasswordRules) {
		r.MinUniqueChars = count
	}
}
