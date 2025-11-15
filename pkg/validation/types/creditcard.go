// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu paket, String, Number, Array, CreditCard gibi tiplerin doğrulama ve
// dönüşüm işlemlerini kolaylaştırmak için geliştirilmiştir.
package types

import (
	"fmt"

	"github.com/biyonik/conduit-go/pkg/validation"
	"github.com/biyonik/conduit-go/pkg/validation/rules"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// CreditCardType, bir kredi kartı alanının doğrulanmasını sağlar.
// cardType: "visa", "mastercard", "amex" vb.
type CreditCardType struct {
	BaseType
	cardType string
}

// --- Akıcı (Fluent) Metotlar ---

// Required, alanı zorunlu olarak işaretler.
func (c *CreditCardType) Required() *CreditCardType {
	c.SetRequired()
	return c
}

// Label, alan için insan okunabilir bir isim atar.
func (c *CreditCardType) Label(label string) *CreditCardType {
	c.SetLabel(label)
	return c
}

// Type, kabul edilen kredi kartı tipini belirler.
// Örn: "visa", "mastercard", "amex"
func (c *CreditCardType) Type(typ string) *CreditCardType {
	c.cardType = typ
	return c
}

// --- Arayüz (Interface) Implementasyonu ---

// Validate, kredi kartı numarasının geçerliliğini kontrol eder.
//
// Parametreler:
//   - field: Alan adı
//   - value: Doğrulanacak değer
//   - result: ValidationResult, hatalar buraya eklenir
func (c *CreditCardType) Validate(field string, value any, result *validation.ValidationResult) {
	// Temel zorunluluk kontrolünü uygula
	c.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}

	if value == nil {
		return
	}

	str, ok := value.(string)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı metin tipinde olmalıdır", c.label))
		return
	}

	// Kredi kartı numarasını doğrula
	if !rules.IsValidCreditCard(str, c.cardType) {
		fieldName := c.label
		if fieldName == "" {
			fieldName = field
		}

		typeText := ""
		if c.cardType != "" {
			typeText = fmt.Sprintf(" (%s)", c.cardType)
		}

		result.AddError(field, fmt.Sprintf("%s alanı geçerli bir kredi kartı numarası%s olmalıdır", fieldName, typeText))
	}
}
