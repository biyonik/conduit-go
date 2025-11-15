// Package types, tip bazlı doğrulama nesnelerini yönetir.
// Bu paket, boolean, string, number, object, array vb. tiplerin
// doğrulama ve dönüşüm işlemlerini kolaylaştırmak için geliştirilmiştir.
package types

import (
	"fmt"

	"github.com/biyonik/conduit-go/pkg/validation"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// BooleanType, boolean değerlerinin doğrulamasını ve dönüşümünü yönetir.
// BaseType'ı gömerek ortak doğrulama ve transform fonksiyonlarını kullanır.
type BooleanType struct {
	BaseType // Ortak doğrulama ve transform fonksiyonları
}

// --- Akıcı (Fluent) Metotlar ---

// Required, alanın zorunlu olduğunu işaretler ve zincirlemeye izin verir.
//
// Döndürür:
//   - *BooleanType: Kendi örneği (zincirleme kullanım için)
func (b *BooleanType) Required() *BooleanType {
	b.SetRequired()
	return b
}

// Label, alan için insan okunabilir bir isim atar.
//
// Parametreler:
//   - label: Alan adı için kullanıcı dostu isim
//
// Döndürür:
//   - *BooleanType: Kendi örneği (zincirleme kullanım için)
func (b *BooleanType) Label(label string) *BooleanType {
	b.SetLabel(label)
	return b
}

// Default, alan için varsayılan boolean değer atar.
//
// Parametreler:
//   - value: Varsayılan boolean değeri
//
// Döndürür:
//   - *BooleanType: Kendi örneği (zincirleme kullanım için)
func (b *BooleanType) Default(value bool) *BooleanType {
	b.SetDefault(value)
	return b
}

// --- Arayüz (Interface) Implementasyonu ---

// Validate, verilen boolean değeri doğrular ve hataları ValidationResult'a ekler.
//
// Parametreler:
//   - field: Doğrulanan alanın adı
//   - value: Doğrulanacak değer
//   - result: Hataların ekleneceği ValidationResult nesnesi
func (b *BooleanType) Validate(field string, value any, result *validation.ValidationResult) {
	// Temel doğrulama: zorunlu alan kontrolü
	b.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}

	// Nil değer kontrolü (zorunlu değilse dur)
	if value == nil {
		return
	}

	// Tip kontrolü
	_, ok := value.(bool)
	if !ok {
		fieldName := b.label
		if fieldName == "" {
			fieldName = field
		}
		result.AddError(field, fmt.Sprintf("%s alanı boolean tipinde olmalıdır", fieldName))
	}
}
