// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu paket, String, Number, Object, Array gibi tiplerin doğrulama ve
// dönüşüm (transform) işlemlerini kolaylaştırmak için geliştirilmiştir.
package types

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/biyonik/conduit-go/pkg/validation"
	"github.com/biyonik/conduit-go/pkg/validation/rules"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// StringType, metin değerlerinin doğrulamasını ve dönüşümünü yönetir.
// BaseType'ı gömerek ortak doğrulama ve transform fonksiyonlarını kullanır.
type StringType struct {
	BaseType           // Ortak doğrulama ve transform fonksiyonları
	minLength     *int // Minimum uzunluk kısıtı
	maxLength     *int // Maksimum uzunluk kısıtı
	emailRegex    *regexp.Regexp
	urlRegex      *regexp.Regexp
	allowedValues []string
	passwordRules *rules.PasswordRules
	ipVersion     *int
	phoneCountry  *string
}

// --- Akıcı (Fluent) Metotlar ---
// Bu metotlar zincirleme kullanım için tasarlanmıştır.

// Required, alanın zorunlu olduğunu belirtir.
func (s *StringType) Required() *StringType {
	s.SetRequired()
	return s
}

// Label, alan için insan okunabilir bir isim atar.
func (s *StringType) Label(label string) *StringType {
	s.SetLabel(label)
	return s
}

// Default, alan için varsayılan değer atar.
func (s *StringType) Default(value string) *StringType {
	s.SetDefault(value)
	return s
}

// Min, metin alanının minimum uzunluğunu ayarlar.
func (s *StringType) Min(length int) *StringType {
	s.minLength = &length
	return s
}

// Max, metin alanının maksimum uzunluğunu ayarlar.
func (s *StringType) Max(length int) *StringType {
	s.maxLength = &length
	return s
}

// Email, alanın e-posta formatında olmasını zorunlu kılar.
func (s *StringType) Email() *StringType {
	s.emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return s
}

func (s *StringType) URL() *StringType {
	s.urlRegex = regexp.MustCompile(`^(https?:\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?$`)
	return s
}

// OneOf, alanın belirli değerlerden biri olmasını sağlar.
func (s *StringType) OneOf(values []string) *StringType {
	s.allowedValues = values
	return s
}

// IP, alanın bir IP adresi olmasını gerektirir.
// Parametre yoksa (IP()), hem v4 hem v6 kabul edilir.
// IP(4) -> sadece IPv4
// IP(6) -> sadece IPv6
func (s *StringType) IP(version ...int) *StringType {
	v := 0 // Varsayılan: v4/v6
	if len(version) > 0 {
		v = version[0]
	}
	s.ipVersion = &v
	return s
}

// Phone, alanın bir telefon numarası olmasını gerektirir.
func (s *StringType) Phone(countryCode string) *StringType {
	s.phoneCountry = &countryCode
	return s
}

// Trim, alanın başındaki ve sonundaki boşlukları temizler.
func (s *StringType) Trim() *StringType {
	s.AddTransform(func(value any) (any, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("Trim sadece string değerler için uygulanabilir")
		}
		return strings.TrimSpace(str), nil
	})
	return s
}

// StripTags, alanın HTML etiketlerini temizler, izin verilen etiketler korunabilir.
func (s *StringType) StripTags(allowedTags ...string) *StringType {
	s.AddTransform(func(value any) (any, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("StripTags sadece string değerler için uygulanabilir")
		}
		return rules.StripHtmlTags(str, allowedTags...), nil
	})
	return s
}

// Password, alanın parola kurallarına uygun olmasını sağlar.
func (s *StringType) Password(options ...PasswordOption) *StringType {
	defaults := &rules.PasswordRules{
		MinLength:         8,
		MaxLength:         72,
		RequireUppercase:  true,
		RequireLowercase:  true,
		RequireNumeric:    true,
		RequireSpecial:    true,
		SpecialChars:      `!@#$%^&*(),.?":{}|<>+-`,
		MinUniqueChars:    6,
		MaxRepeatingChars: 3,
		DisallowCommon:    true,
		DisallowKeyboard:  true,
		MinEntropy:        50.0,
	}

	for _, option := range options {
		option(defaults)
	}

	s.passwordRules = defaults
	return s
}

// --- Interface Implementasyonu ---

// Validate, verilen değeri StringType kurallarına göre doğrular.
//
// Parametreler:
//   - field: Alan adı
//   - value: Doğrulanacak değer
//   - result: ValidationResult nesnesi, hatalar buraya eklenir
func (s *StringType) Validate(field string, value any, result *validation.ValidationResult) {
	// Temel doğrulama
	s.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}

	if value == nil {
		return
	}

	str, ok := value.(string)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı metin tipinde olmalıdır", s.label))
		return
	}

	fieldName := s.label
	if fieldName == "" {
		fieldName = field
	}

	// Minimum ve maksimum uzunluk
	if s.minLength != nil && len(str) < *s.minLength {
		result.AddError(field, fmt.Sprintf("%s alanı en az %d karakter olmalıdır", fieldName, *s.minLength))
	}
	if s.maxLength != nil && len(str) > *s.maxLength {
		result.AddError(field, fmt.Sprintf("%s alanı en fazla %d karakter olmalıdır", fieldName, *s.maxLength))
	}

	// E-posta kontrolü
	if s.emailRegex != nil && !s.emailRegex.MatchString(str) {
		result.AddError(field, fmt.Sprintf("%s alanı geçerli bir e-posta formatında değil", fieldName))
	}

	// Parola kuralları
	if s.passwordRules != nil && str != "" {
		passwordErrors := rules.ValidatePassword(str, s.passwordRules)
		for _, err := range passwordErrors {
			result.AddError(field, fmt.Sprintf("%s %s", fieldName, err))
		}
	}

	if s.ipVersion != nil {
		if !rules.IsValidIP(str, *s.ipVersion) {
			versionText := ""
			if *s.ipVersion == 4 {
				versionText = " (IPv4)"
			}
			if *s.ipVersion == 6 {
				versionText = " (IPv6)"
			}
			result.AddError(field, fmt.Sprintf("%s alanı geçerli bir IP%s adresi olmalıdır", fieldName, versionText))
		}
	}

	if s.phoneCountry != nil {
		if !rules.IsValidPhoneNumber(str, *s.phoneCountry) {
			result.AddError(field, fmt.Sprintf("%s alanı geçerli bir %s telefon numarası olmalıdır", fieldName, *s.phoneCountry))
		}
	}
}
