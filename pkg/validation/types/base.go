// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu paket, String, Number, Object, Array gibi tiplerin doğrulama ve
// dönüşüm (transform) işlemlerini kolaylaştırmak için geliştirilmiştir.
package types

import (
	"fmt"

	"github.com/biyonik/conduit-go/pkg/validation"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// BaseType, tüm tiplerin gömeceği temel doğrulama ve dönüşüm işlevlerini sağlar.
// isRequired: alan zorunlu mu?
// label: alanın insan okunabilir adı
// defaultValue: dönüşüm sırasında uygulanacak varsayılan değer
// transformations: değere uygulanacak dönüşüm fonksiyonları (trim, strip tags vb.)
type BaseType struct {
	isRequired      bool
	label           string
	defaultValue    any
	transformations []func(any) (any, error)
}

// --- Akıcı (Fluent) Metotlar ---

// SetRequired, alanı zorunlu olarak işaretler.
// Zorunlu alanlar, değer nil veya boş string olduğunda ValidationResult'a hata ekler.
func (b *BaseType) SetRequired() {
	b.isRequired = true
}

// SetLabel, alan için insan okunabilir bir isim atar.
func (b *BaseType) SetLabel(label string) {
	b.label = label
}

// SetDefault, dönüşüm sırasında kullanılacak varsayılan değeri belirler.
func (b *BaseType) SetDefault(value any) {
	b.defaultValue = value
}

// AddTransform, değere uygulanacak dönüşüm fonksiyonunu ekler.
// Örn: trim, strip tags, normalize vb.
func (b *BaseType) AddTransform(fn func(any) (any, error)) {
	b.transformations = append(b.transformations, fn)
}

// --- Arayüz (Interface) Implementasyonu ---

// Transform, değere tüm tanımlı dönüşümleri uygular.
//
// Parametreler:
//   - value: Dönüştürülecek değer
//
// Döndürür:
//   - any: Dönüştürülmüş değer
//   - error: Dönüşüm sırasında oluşan hata
func (b *BaseType) Transform(value any) (any, error) {
	// 1. Varsayılan değeri uygula
	if value == nil && b.defaultValue != nil {
		value = b.defaultValue
	}

	// 2. Dönüşümleri uygula
	if value == nil {
		return nil, nil
	}

	var err error
	for _, fn := range b.transformations {
		value, err = fn(value)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

// Validate, temel doğrulama mantığını (zorunluluk) sağlar.
//
// Parametreler:
//   - field: Alan adı
//   - value: Doğrulanacak değer
//   - result: ValidationResult, hatalar buraya eklenir
func (b *BaseType) Validate(field string, value any, result *validation.ValidationResult) {
	fieldName := b.label
	if fieldName == "" {
		fieldName = field
	}

	// Zorunlu alan kontrolü
	if b.isRequired {
		// Nil değer kontrolü
		if value == nil {
			result.AddError(field, fmt.Sprintf("%s alanı zorunludur", fieldName))
			return
		}
		// String ise boş string kontrolü
		if str, ok := value.(string); ok && str == "" {
			result.AddError(field, fmt.Sprintf("%s alanı zorunludur", fieldName))
			return
		}
	}
}
