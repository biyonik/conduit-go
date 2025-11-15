// Package validation, veri doğrulama süreçlerini Go dilinde yönetmek için
// geliştirilmiş kapsamlı bir yapıdır. Bu paket, tip bazlı, şema bazlı, çapraz doğrulama
// (cross-field validation) gibi modern web uygulamalarında kritik
// olan doğrulama adımlarını kolaylaştırır.
//
// Laravel/Symfony tarzı kullanım hissi verir; developer, tek bir
// ValidationSchema üzerinden hem tip kontrollerini hem de alanlar arası
// doğrulamayı gerçekleştirebilir.
//
// Paket içerisinde hem helper tipler (String, Number, Object, Array)
// hem de şema oluşturma ve doğrulama mekanizmaları mevcuttur.
package validation

import (
	"fmt"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// conditionalRule, bir 'When' kuralını saklamak için kullanılır.
// PHP'deki $conditionalRules dizisinin bir elemanıdır.
type conditionalRule struct {
	field         string
	expectedValue any
	callback      func() Schema // Alt şemayı döndüren fonksiyon
}

// Bu yapı, veri doğrulama sürecini yönetir; tip bazlı doğrulama, çapraz alan
// doğrulama ve dönüşüm (transform) işlemleri bu sınıf üzerinden yürütülür.
//
// Özellikler:
//   - shape: Alan adı -> Type eşlemesi (tip bazlı doğrulama)
//   - crossValidators: Alanlar arası doğrulama fonksiyonları
type ValidationSchema struct {
	shape            map[string]Type
	crossValidators  []func(data map[string]any) error
	conditionalRules []conditionalRule
}

// Go'da bu bir 'constructor' fonksiyonudur ve yeni ValidationSchema nesnesi döndürür.
//
// Döndürür:
//   - *ValidationSchema: Boş shape ve crossValidators ile yeni şema
func Make() *ValidationSchema {
	return &ValidationSchema{
		shape:            make(map[string]Type),
		conditionalRules: make([]conditionalRule, 0), // Slice'ı başlat
	}
}

// --- Interface Implementasyonları ---

// Shape, şemada alan adları ve tiplerini tanımlar.
//
// Parametre:
//   - shape: Alan adı -> Type eşlemesi
//
// Döndürür:
//   - Schema: Aynı şema nesnesi (method chaining)
func (vs *ValidationSchema) Shape(shape map[string]Type) Schema {
	vs.shape = shape
	return vs
}

// CrossValidate, alanlar arası doğrulama fonksiyonları ekler.
//
// Parametre:
//   - fn: data map'i alıp hata döndürebilecek bir fonksiyon
//
// Döndürür:
//   - Schema: Aynı şema nesnesi (method chaining)
func (vs *ValidationSchema) CrossValidate(fn func(data map[string]any) error) Schema {
	vs.crossValidators = append(vs.crossValidators, fn)
	return vs
}

func (vs *ValidationSchema) When(field string, expectedValue any, callback func() Schema) Schema {
	vs.conditionalRules = append(vs.conditionalRules, conditionalRule{
		field:         field,
		expectedValue: expectedValue,
		callback:      callback,
	})
	return vs // Zincirleme (chaining) için
}

// Validate, tüm şemanın doğrulama sürecini başlatır.
// Bu metod, validation sürecinin kalbidir ve aşağıdaki adımları uygular:
//  1. Transform: Ham veriyi temizler ve tip dönüşümlerini uygular.
//  2. Validate: Temizlenmiş veri üzerinden tip bazlı doğrulamayı çalıştırır.
//  3. Cross-Validate: Alanlar arası mantıksal doğrulamaları uygular.
//  4. Result: Hata yoksa validData ayarlanır, yoksa hata mesajları döner.
//
// Parametre:
//   - data: Doğrulanacak veri haritası
//
// Döndürür:
//   - *ValidationResult: Doğrulama sonucu (hatalar ve temiz veri)
func (vs *ValidationSchema) Validate(data map[string]any) *ValidationResult {
	result := NewResult()
	transformedData := make(map[string]any)

	// 1. AŞAMA: DÖNÜŞTÜRME (TRANSFORM)
	// (Değişiklik yok: Veriyi temizler ve 'transformedData' haritasını doldurur)
	for field, typ := range vs.shape {
		value := data[field]

		transformedValue, err := typ.Transform(value)
		if err != nil {
			result.AddError(field, fmt.Sprintf("Dönüşüm hatası: %s", err.Error()))
			continue
		}
		transformedData[field] = transformedValue
	}

	if result.HasErrors() {
		// Dönüşüm sırasında kritik hata olduysa (örn: tip uyuşmazlığı)
		// doğrulamaya hiç başlamadan dönebiliriz.
		// VEYA devam edip 'Validate'in bu hataları yakalamasına izin veririz.
		// Devam etmek daha sağlamdır.
	}

	// 2. AŞAMA: TEMEL DOĞRULAMA (VALIDATE)
	// (Değişiklik yok: 'transformedData'yı temel 'shape'e göre doğrular)
	for field, typ := range vs.shape {
		typ.Validate(field, transformedData[field], result)
	}

	// 3. AŞAMA (YENİ): KOŞULLU DOĞRULAMA (WHEN)
	// PHP'deki 'applyConditionalValidations' mantığı.
	// Temel doğrulamalardan sonra çalışır.
	if len(vs.conditionalRules) > 0 {
		for _, rule := range vs.conditionalRules {
			// Temizlenmiş veriden koşulun değerini al
			value, exists := transformedData[rule.field]

			// Koşul sağlanıyor mu? (örn: 'payment_type' == 'credit_card')
			if exists && value == rule.expectedValue {

				// Koşul sağlandı! Callback'i çalıştırıp alt şemayı (sub-schema) al.
				subSchema := rule.callback()

				// Alt şemanın 'Validate' metodunu, TÜM veri üzerinde çalıştır.
				// Alt şema, sadece kendi 'shape'i içindeki alanları
				// (örn: 'card_number') kontrol edecektir.
				subResult := subSchema.Validate(transformedData)

				// Alt şemadan gelen hataları ana sonuca (result) ekle.
				if subResult.HasErrors() {
					for field, messages := range subResult.Errors() {
						for _, msg := range messages {
							result.AddError(field, msg)
						}
					}
				}
			}
		}
	}

	// 4. AŞAMA: ÇAPRAZ ALAN DOĞRULAMA (CROSS-VALIDATE)
	// (Değişiklik yok: Sadece HİÇ hata yoksa çalışır)
	if !result.HasErrors() {
		for _, fn := range vs.crossValidators {
			err := fn(transformedData)
			if err != nil {
				result.AddError("_cross_validation", err.Error())
			}
		}
	}

	// 5. AŞAMA: SONUÇ
	// (Değişiklik yok: Sadece HİÇ hata yoksa temiz veriyi ayarlar)
	if !result.HasErrors() {
		result.SetValidData(transformedData)
	}

	return result
}
