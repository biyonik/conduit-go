// Package types, tip bazlı doğrulama nesnelerini ve kurallarını yönetir.
// Bu paket, UUID, String, Number, Object, Array gibi tiplerin doğrulama
// ve dönüşüm işlemlerini merkezi bir şekilde yönetmek için geliştirilmiştir.
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

// UuidType, UUID değerlerinin doğrulamasını ve dönüşümünü yönetir.
// BaseType'ı gömerek ortak doğrulama ve transform fonksiyonlarını kullanır.
type UuidType struct {
	BaseType
	version int // 0 = tüm UUID formatları, 1-5 = belirli versiyon
}

// --- Akıcı (Fluent) Metotlar ---
// Bu metotlar zincirleme kullanım için tasarlanmıştır.

// Required, alanın zorunlu olduğunu belirtir.
func (u *UuidType) Required() *UuidType {
	u.SetRequired()
	return u
}

// Label, alan için insan okunabilir bir isim atar.
func (u *UuidType) Label(label string) *UuidType {
	u.SetLabel(label)
	return u
}

// Version, doğrulama sırasında belirli UUID versiyonunu zorunlu kılar.
// 0 = tüm versiyonlar, 1-5 = spesifik versiyon
func (u *UuidType) Version(v int) *UuidType {
	if v >= 0 && v <= 5 {
		u.version = v
	}
	return u
}

// --- Arayüz (Interface) Implementasyonu ---

// Validate, verilen değeri UUID kurallarına göre doğrular.
//
// Parametreler:
//   - field: Doğrulanan alanın adı
//   - value: Doğrulanacak değer
//   - result: ValidationResult nesnesi; oluşan hatalar buraya eklenir
//
// İşleyiş:
//  1. Zorunluluk kontrolü yapılır.
//  2. Değer nil ise işlem sonlandırılır.
//  3. Tip kontrolü: değer string olmalıdır.
//  4. UUID doğrulama: versiyon parametresi varsa buna göre doğrulama yapılır.
//  5. Hata varsa ValidationResult nesnesine eklenir.
func (u *UuidType) Validate(field string, value any, result *validation.ValidationResult) {
	u.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}
	if value == nil {
		return
	}

	str, ok := value.(string)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı metin tipinde olmalıdır", u.label))
		return
	}

	fieldName := u.label
	if fieldName == "" {
		fieldName = field
	}

	versionText := ""
	if u.version > 0 {
		versionText = fmt.Sprintf(" (v%d)", u.version)
	}

	if !rules.IsValidUUID(str, u.version) {
		result.AddError(field, fmt.Sprintf("%s alanı geçerli bir UUID%s olmalıdır", fieldName, versionText))
	}
}
