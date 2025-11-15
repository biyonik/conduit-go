// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu paket, String, Number, Array, Object, CreditCard gibi tiplerin doğrulama ve
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

// ObjectType, iç içe nesnelerin doğrulanmasını ve dönüşümünü sağlar.
// Her alan için ayrı bir şema belirlenebilir.
type ObjectType struct {
	BaseType
	shape map[string]validation.Type // İç şemayı Type arayüzü olarak tutar
}

// --- Akıcı (Fluent) Metotlar ---

// Required, alanı zorunlu olarak işaretler.
func (o *ObjectType) Required() *ObjectType {
	o.SetRequired()
	return o
}

// Label, alan için insan okunabilir bir isim atar.
func (o *ObjectType) Label(label string) *ObjectType {
	o.SetLabel(label)
	return o
}

// Shape, iç nesne için alanların şemasını belirler.
func (o *ObjectType) Shape(shape map[string]validation.Type) *ObjectType {
	o.shape = shape
	return o
}

// --- Arayüz (Interface) Implementasyonu ---

// Transform, iç nesnedeki her alan için dönüşümü uygular.
//
// Parametre:
//   - value: Dönüştürülecek veri
//
// Döndürür:
//   - any: Dönüştürülmüş veri
//   - error: Dönüşüm sırasında oluşan hata
func (o *ObjectType) Transform(value any) (any, error) {
	value, err := o.BaseType.Transform(value)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	data, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("nesne (object) tipinde olmalıdır")
	}

	transformedData := make(map[string]any)
	for field, typ := range o.shape {
		subValue := data[field]

		transformedSubValue, err := typ.Transform(subValue)
		if err != nil {
			return nil, fmt.Errorf("alan '%s': %w", field, err)
		}
		transformedData[field] = transformedSubValue
	}

	// Eksik alanları da ekle
	for k, v := range data {
		if _, ok := transformedData[k]; !ok {
			transformedData[k] = v
		}
	}

	return transformedData, nil
}

// Validate, iç nesnedeki her alanın doğrulamasını uygular.
//
// Parametreler:
//   - field: Üst nesne alan adı
//   - value: Doğrulanacak veri
//   - result: ValidationResult, hatalar buraya eklenir
func (o *ObjectType) Validate(field string, value any, result *validation.ValidationResult) {
	o.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}
	if value == nil {
		return
	}

	data, ok := value.(map[string]any)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı nesne (object) tipinde olmalıdır", o.label))
		return
	}

	for subField, subSchema := range o.shape {
		subValue := data[subField]
		fullFieldPath := fmt.Sprintf("%s.%s", field, subField)
		subSchema.Validate(fullFieldPath, subValue, result)
	}
}
