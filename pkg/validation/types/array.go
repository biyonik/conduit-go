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

// ArrayType, bir dizi (array) değerinin doğrulamasını ve dönüşümünü yönetir.
// BaseType'ı gömerek ortak doğrulama ve transform fonksiyonlarını kullanır.
type ArrayType struct {
	BaseType
	minLength     *int            // Minimum eleman sayısı
	maxLength     *int            // Maksimum eleman sayısı
	elementSchema validation.Type // Dizideki her elemanın uyması gereken şema
}

// --- Akıcı (Fluent) Metotlar ---

// Required, alanın zorunlu olduğunu belirtir.
func (a *ArrayType) Required() *ArrayType {
	a.SetRequired()
	return a
}

// Label, alan için insan okunabilir bir isim atar.
func (a *ArrayType) Label(label string) *ArrayType {
	a.SetLabel(label)
	return a
}

// Min, dizinin minimum eleman sayısını ayarlar.
func (a *ArrayType) Min(length int) *ArrayType {
	a.minLength = &length
	return a
}

// Max, dizinin maksimum eleman sayısını ayarlar.
func (a *ArrayType) Max(length int) *ArrayType {
	a.maxLength = &length
	return a
}

// Elements, dizideki her elemanın uyması gereken şemayı belirler.
func (a *ArrayType) Elements(schema validation.Type) *ArrayType {
	a.elementSchema = schema
	return a
}

// --- Interface Implementasyonu ---

// Transform, dizideki her eleman için dönüşümü uygular.
//
// Parametreler:
//   - value: Doğrulanacak ve dönüştürülecek değer
//
// Döndürür:
//   - any: Dönüştürülmüş dizi
//   - error: Dönüşüm hatası varsa döndürülür
func (a *ArrayType) Transform(value any) (any, error) {
	value, err := a.BaseType.Transform(value)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	slice, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("dizi (array) tipinde olmalıdır")
	}

	if a.elementSchema != nil {
		transformedSlice := make([]any, len(slice))
		for i, item := range slice {
			transformedItem, err := a.elementSchema.Transform(item)
			if err != nil {
				return nil, fmt.Errorf("dizi index %d: %w", i, err)
			}
			transformedSlice[i] = transformedItem
		}
		return transformedSlice, nil
	}

	return slice, nil
}

// Validate, dizinin ve elemanlarının kurallara uygunluğunu kontrol eder.
//
// Parametreler:
//   - field: Alan adı
//   - value: Doğrulanacak değer
//   - result: ValidationResult nesnesi, hatalar buraya eklenir
func (a *ArrayType) Validate(field string, value any, result *validation.ValidationResult) {
	// Temel doğrulama
	a.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}
	if value == nil {
		return
	}

	slice, ok := value.([]any)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı dizi (array) tipinde olmalıdır", a.label))
		return
	}

	fieldName := a.label
	if fieldName == "" {
		fieldName = field
	}

	// Minimum ve maksimum uzunluk kontrolü
	if a.minLength != nil && len(slice) < *a.minLength {
		result.AddError(field, fmt.Sprintf("%s alanında en az %d eleman olmalıdır", fieldName, *a.minLength))
	}
	if a.maxLength != nil && len(slice) > *a.maxLength {
		result.AddError(field, fmt.Sprintf("%s alanında en fazla %d eleman olmalıdır", fieldName, *a.maxLength))
	}

	// Eleman şeması varsa, her elemanı doğrula
	if a.elementSchema != nil {
		for i, item := range slice {
			elementFieldPath := fmt.Sprintf("%s[%d]", field, i)
			a.elementSchema.Validate(elementFieldPath, item, result)
		}
	}
}
