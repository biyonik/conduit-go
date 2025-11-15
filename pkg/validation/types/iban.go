// pkg/validation/types/iban.go
package types

import (
	"fmt"

	"github.com/biyonik/conduit-go/pkg/validation"
	"github.com/biyonik/conduit-go/pkg/validation/rules"
)

// IbanType, PHP'deki IbanType sınıfının portudur.
type IbanType struct {
	BaseType
	countryCode string
}

// --- Akıcı (Fluent) Metotlar ---

func (i *IbanType) Required() *IbanType {
	i.SetRequired()
	return i
}

func (i *IbanType) Label(label string) *IbanType {
	i.SetLabel(label)
	return i
}

// Country, PHP'deki country() metodu.
func (i *IbanType) Country(code string) *IbanType {
	i.countryCode = code
	return i
}

// --- Arayüz (Interface) Implementasyonu ---

func (i *IbanType) Validate(field string, value any, result *validation.ValidationResult) {
	i.BaseType.Validate(field, value, result)
	if result.HasErrors() || value == nil {
		return
	}

	str, ok := value.(string)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı metin tipinde olmalıdır", i.label))
		return
	}

	// Kuralı çağır
	if !rules.IsValidIBAN(str, i.countryCode) {
		fieldName := i.label
		if fieldName == "" {
			fieldName = field
		}
		countryText := ""
		if i.countryCode != "" {
			countryText = fmt.Sprintf(" (%s)", i.countryCode)
		}
		result.AddError(field, fmt.Sprintf("%s alanı geçerli bir IBAN%s olmalıdır", fieldName, countryText))
	}
}
