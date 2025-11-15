// Package types, tip bazlı doğrulama nesnelerini yönetir.
// Bu paket, boolean, string, number, object, array, date vb.
// tiplerin doğrulama ve dönüşüm işlemlerini kolaylaştırmak için geliştirilmiştir.
package types

import (
	"fmt"
	"time"

	"github.com/biyonik/conduit-go/pkg/validation"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// DateType, tarih değerlerinin doğrulamasını ve dönüşümünü yönetir.
// BaseType'ı gömerek ortak doğrulama ve transform fonksiyonlarını kullanır.
type DateType struct {
	BaseType
	format     string  // Tarih formatı (Go'nun time.Parse formatı, örn: "2006-01-02")
	minDateStr *string // Minimum tarih (string olarak saklanır)
	maxDateStr *string // Maksimum tarih (string olarak saklanır)
}

// --- Akıcı (Fluent) Metotlar ---

// Required, alanın zorunlu olduğunu işaretler.
//
// Döndürür:
//   - *DateType: Kendi örneği (zincirleme kullanım için)
func (d *DateType) Required() *DateType {
	d.SetRequired()
	return d
}

// Label, alan için kullanıcı dostu isim atar.
//
// Parametreler:
//   - label: Alan adı için insan okunabilir isim
//
// Döndürür:
//   - *DateType: Kendi örneği (zincirleme kullanım için)
func (d *DateType) Label(label string) *DateType {
	d.SetLabel(label)
	return d
}

// Default, alan için varsayılan tarih değeri atar.
//
// Parametreler:
//   - value: Varsayılan tarih stringi
//
// Döndürür:
//   - *DateType: Kendi örneği (zincirleme kullanım için)
func (d *DateType) Default(value string) *DateType {
	d.SetDefault(value)
	return d
}

// Format, tarih alanı için özel format belirler.
//
// Parametreler:
//   - goTimeFormat: Go'nun time.Parse formatı (örn: "2006-01-02")
//
// Döndürür:
//   - *DateType: Kendi örneği (zincirleme kullanım için)
func (d *DateType) Format(goTimeFormat string) *DateType {
	d.format = goTimeFormat
	return d
}

// Min, alanın minimum tarih değerini belirler.
//
// Parametreler:
//   - dateStr: Minimum tarih stringi
//
// Döndürür:
//   - *DateType: Kendi örneği (zincirleme kullanım için)
func (d *DateType) Min(dateStr string) *DateType {
	d.minDateStr = &dateStr
	return d
}

// Max, alanın maksimum tarih değerini belirler.
//
// Parametreler:
//   - dateStr: Maksimum tarih stringi
//
// Döndürür:
//   - *DateType: Kendi örneği (zincirleme kullanım için)
func (d *DateType) Max(dateStr string) *DateType {
	d.maxDateStr = &dateStr
	return d
}

// --- Arayüz (Interface) Implementasyonu ---

// Transform, verilen string veya time.Time değerini time.Time nesnesine dönüştürür.
//
// Parametreler:
//   - value: Dönüştürülecek değer (string veya time.Time)
//
// Döndürür:
//   - any: Dönüştürülmüş time.Time değeri
//   - error: Dönüşüm sırasında oluşan hata (varsa)
func (d *DateType) Transform(value any) (any, error) {
	value, err := d.BaseType.Transform(value)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	// Değer zaten time.Time ise olduğu gibi dön
	if _, ok := value.(time.Time); ok {
		return value, nil
	}

	// Değer string ise parse et
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("tarih alanı string veya time.Time tipinde olmalıdır")
	}

	parsedDate, err := time.Parse(d.format, str)
	if err != nil {
		return nil, fmt.Errorf("geçerli bir tarih formatı değil. Beklenen: %s", d.format)
	}
	return parsedDate, nil
}

// Validate, tarih alanının doğrulama mantığını uygular ve hataları ValidationResult'a ekler.
//
// Parametreler:
//   - field: Alan adı
//   - value: Doğrulanacak değer (time.Time olmalıdır)
//   - result: Hataların ekleneceği ValidationResult nesnesi
func (d *DateType) Validate(field string, value any, result *validation.ValidationResult) {
	// Temel doğrulama
	d.BaseType.Validate(field, value, result)
	if result.HasErrors() {
		return
	}
	if value == nil {
		return
	}

	// Tip kontrolü
	parsedDate, ok := value.(time.Time)
	if !ok {
		result.AddError(field, fmt.Sprintf("%s alanı geçerli bir tarih olmalıdır", d.label))
		return
	}

	fieldName := d.label
	if fieldName == "" {
		fieldName = field
	}

	// Min tarih kontrolü
	if d.minDateStr != nil {
		minDate, err := time.Parse(d.format, *d.minDateStr)
		if err != nil {
			result.AddError(field, fmt.Sprintf("%s için tanımlanan min() kuralı geçersiz formatta", fieldName))
		} else if parsedDate.Before(minDate) {
			result.AddError(field, fmt.Sprintf("%s alanı %s tarihinden önce olamaz", fieldName, *d.minDateStr))
		}
	}

	// Max tarih kontrolü
	if d.maxDateStr != nil {
		maxDate, err := time.Parse(d.format, *d.maxDateStr)
		if err != nil {
			result.AddError(field, fmt.Sprintf("%s için tanımlanan max() kuralı geçersiz formatta", fieldName))
		} else if parsedDate.After(maxDate) {
			result.AddError(field, fmt.Sprintf("%s alanı %s tarihinden sonra olamaz", fieldName, *d.maxDateStr))
		}
	}
}
