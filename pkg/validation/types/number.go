// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu paket, String, Number, Array, CreditCard gibi tiplerin doğrulama ve
// dönüşüm işlemlerini kolaylaştırmak için geliştirilmiştir.
package types

import (
	"fmt"

	"github.com/biyonik/conduit-go/pkg/validation"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// NumberType, bir sayısal alanın doğrulanmasını sağlar.
// min, max ve tamsayı kontrolü gibi opsiyonel kuralları içerir.
type NumberType struct {
	BaseType
	min       *float64 // Minimum değer (opsiyonel)
	max       *float64 // Maksimum değer (opsiyonel)
	isInteger bool     // Sadece tamsayı olmalı mı
}

// --- Akıcı (Fluent) Metotlar ---

// Required, alanı zorunlu olarak işaretler.
func (n *NumberType) Required() *NumberType {
	n.SetRequired()
	return n
}

// Label, alan için insan okunabilir bir isim atar.
func (n *NumberType) Label(label string) *NumberType {
	n.SetLabel(label)
	return n
}

// Default, sayısal bir varsayılan değer atar.
func (n *NumberType) Default(value any) *NumberType {
	switch v := value.(type) {
	case int:
		n.SetDefault(float64(v))
	case float64:
		n.SetDefault(v)
	case float32:
		n.SetDefault(float64(v))
	default:
		n.SetDefault(value)
	}
	return n
}

// Min, alan için minimum değeri belirler.
func (n *NumberType) Min(val float64) *NumberType {
	n.min = &val
	return n
}

// Max, alan için maksimum değeri belirler.
func (n *NumberType) Max(val float64) *NumberType {
	n.max = &val
	return n
}

// Integer, değerin tamsayı olması gerektiğini işaretler.
func (n *NumberType) Integer() *NumberType {
	n.isInteger = true
	return n
}

// --- Arayüz (Interface) Implementasyonu ---

// Validate, sayısal alanın doğrulama mantığını çalıştırır.
//
// Parametreler:
//   - field: Alan adı
//   - value: Doğrulanacak değer
//   - result: ValidationResult, hatalar buraya eklenir
func (n *NumberType) Validate(field string, value any, result *validation.ValidationResult) {
	// Temel zorunluluk kontrolünü uygula
	n.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}

	// Değer nil ise ve zorunlu değilse dur
	if value == nil {
		return
	}

	// Sayısal tip kontrolü
	var num float64
	var ok bool

	switch v := value.(type) {
	case int:
		num = float64(v)
		ok = true
	case float64:
		num = v
		ok = true
	case float32:
		num = float64(v)
		ok = true
	default:
		ok = false
	}

	fieldName := n.label
	if fieldName == "" {
		fieldName = field
	}

	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı sayısal bir değer olmalıdır", fieldName))
		return
	}

	// Tamsayı kontrolü
	if n.isInteger && num != float64(int64(num)) {
		result.AddError(field, fmt.Sprintf("%s alanı tamsayı olmalıdır", fieldName))
	}

	// Minimum değer kontrolü
	if n.min != nil && num < *n.min {
		result.AddError(field, fmt.Sprintf("%s alanı %v değerinden küçük olamaz", fieldName, *n.min))
	}

	// Maksimum değer kontrolü
	if n.max != nil && num > *n.max {
		result.AddError(field, fmt.Sprintf("%s alanı %v değerinden büyük olamaz", fieldName, *n.max))
	}
}
