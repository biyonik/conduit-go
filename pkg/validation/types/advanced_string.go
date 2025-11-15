// Package types pkg/validation/types/advanced_string.go
package types

import (
	"fmt"

	"github.com/biyonik/conduit-go/pkg/validation"
	"github.com/biyonik/conduit-go/pkg/validation/rules"
)

// AdvancedStringType AdvancedStringType, PHP'deki AdvancedStringType ve
// çeşitli Trait'lerin birleşimidir.
type AdvancedStringType struct {
	StringType // StringType'ı gömüyoruz (embedding).
	// Artık Min, Max, Email, IP, Phone, Password vb.
	// tüm metotlara sahip.

	// 'Advanced' doğrulama kuralları
	turkishChars *bool
	domainCheck  *bool
	charSet      *string // CharSet kuralı için
}

// --- Yeni Gelişmiş Transform (Temizleme) Metotları ---

// StripTags StripTags, PHP'deki stripHtmlTags
func (as *AdvancedStringType) StripTags(allowedTags ...string) *AdvancedStringType {
	as.AddTransform(func(value any) (any, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("StripTags sadece string'lere uygulanabilir")
		}
		// 'allowedTags' parametresini kural fonksiyonuna iletiyoruz
		return rules.StripHtmlTags(str, allowedTags...), nil
	})
	return as
}

// EscapeHTML EscapeHTML, PHP'deki preventXss.
func (as *AdvancedStringType) EscapeHTML() *AdvancedStringType {
	as.AddTransform(func(value any) (any, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("EscapeHTML sadece string'lere uygulanabilir")
		}
		return rules.PreventXss(str), nil
	})
	return as
}

// SanitizeFilename SanitizeFilename, PHP'deki sanitizeFilename.
func (as *AdvancedStringType) SanitizeFilename() *AdvancedStringType {
	as.AddTransform(func(value any) (any, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("SanitizeFilename sadece string'lere uygulanabilir")
		}
		return rules.SanitizeFilename(str), nil
	})
	return as
}

// FilterEmoji FilterEmoji, PHP'deki filterEmoji.
func (as *AdvancedStringType) FilterEmoji(remove bool) *AdvancedStringType {
	as.AddTransform(func(value any) (any, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("FilterEmoji sadece string'lere uygulanabilir")
		}
		return rules.FilterEmoji(str, remove), nil
	})
	return as
}

// --- Yeni Gelişmiş Doğrulama Metotları ---

// TurkishChars TurkishChars, PHP'deki turkishChars() metodu.
func (as *AdvancedStringType) TurkishChars(allow bool) *AdvancedStringType {
	as.turkishChars = &allow
	return as
}

// Domain Domain, PHP'deki domain() metodu.
func (as *AdvancedStringType) Domain(allowSubdomain bool) *AdvancedStringType {
	as.domainCheck = &allowSubdomain
	return as
}

// CharSet CharSet, PHP'deki validateCharSet.
// Örn: CharSet("alphanumeric")
func (as *AdvancedStringType) CharSet(set string) *AdvancedStringType {
	as.charSet = &set
	return as
}

// --- Miras Alınan Metotların Override Edilmesi (ZİNCİRLEME için) ---
// Bu, as.Min(5).StripTags() gibi zincirlerin çalışmasını sağlar.

func (as *AdvancedStringType) Required() *AdvancedStringType {
	as.StringType.Required()
	return as
}

func (as *AdvancedStringType) Label(label string) *AdvancedStringType {
	as.StringType.Label(label)
	return as
}

func (as *AdvancedStringType) Default(value string) *AdvancedStringType {
	as.StringType.Default(value)
	return as
}

func (as *AdvancedStringType) Min(length int) *AdvancedStringType {
	as.StringType.Min(length)
	return as
}

func (as *AdvancedStringType) Max(length int) *AdvancedStringType {
	as.StringType.Max(length)
	return as
}

func (as *AdvancedStringType) Email() *AdvancedStringType {
	as.StringType.Email()
	return as
}

func (as *AdvancedStringType) URL() *AdvancedStringType {
	as.StringType.URL()
	return as
}

func (as *AdvancedStringType) OneOf(values []string) *AdvancedStringType {
	as.StringType.OneOf(values)
	return as
}

func (as *AdvancedStringType) Password(options ...PasswordOption) *AdvancedStringType {
	as.StringType.Password(options...)
	return as
}

func (as *AdvancedStringType) Trim() *AdvancedStringType {
	as.StringType.Trim()
	return as
}

func (as *AdvancedStringType) IP(version ...int) *AdvancedStringType {
	as.StringType.IP(version...)
	return as
}

func (as *AdvancedStringType) Phone(countryCode string) *AdvancedStringType {
	as.StringType.Phone(countryCode)
	return as
}

// --- Arayüz (Interface) Implementasyonu ---

// Validate Validate, 'parent::validate()' mantığını uygular.
func (as *AdvancedStringType) Validate(field string, value any, result *validation.ValidationResult) {
	// 1. Önce temel StringType doğrulamalarını çalıştır (Min, Max, Email, IP, Phone, Password vb.)
	as.StringType.Validate(field, value, result)
	if result.HasErrors() || value == nil {
		return
	}

	str, _ := value.(string) // Tipi zaten StringType.Validate garantiledi

	fieldName := as.label
	if fieldName == "" {
		fieldName = field
	}

	// 2. Gelişmiş kuralları uygula
	if as.turkishChars != nil {
		hasTurkish := rules.HasTurkishChars(str)
		if *as.turkishChars && !hasTurkish {
			result.AddError(field, fmt.Sprintf("%s alanında Türkçe karakter bulunmalıdır", fieldName))
		} else if !*as.turkishChars && hasTurkish {
			result.AddError(field, fmt.Sprintf("%s alanında Türkçe karakter bulunmamalıdır", fieldName))
		}
	}

	if as.domainCheck != nil {
		if !rules.IsValidDomain(str, *as.domainCheck) {
			result.AddError(field, fmt.Sprintf("%s alanı geçerli bir alan adı olmalıdır", fieldName))
		}
	}

	if as.charSet != nil {
		if !rules.ValidateCharSet(str, *as.charSet) {
			result.AddError(field, fmt.Sprintf("%s alanı '%s' karakter setine uymalıdır", fieldName, *as.charSet))
		}
	}
}
