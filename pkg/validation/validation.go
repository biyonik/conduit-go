// Package validation, veri doğrulama (validation) süreçlerini profesyonel bir şekilde
// yönetmek için tasarlanmış küçük, ama güçlü bir yardımcı pakettir. Bu paket, form verileri,
// API payload'ları veya genel veri haritalarının temizlenmesini ve doğrulanmasını
// kolaylaştırır.
//
// Paket sayesinde geliştiriciler, doğrulama sonuçlarını merkezi bir yerde toplar,
// hataları yönetir ve temizlenmiş veriye kolayca erişebilir. Aynı zamanda tip bazlı
// doğrulama (Type Interface) ve şema bazlı doğrulama (Schema Interface) desteklenir.
//
// Modern web uygulamalarında veri doğrulama kritik bir öneme sahiptir; bu paket,
// Laravel veya Symfony gibi frameworklerdeki validation mantığını Go diline taşır.
package validation

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// ValidationResult, bir doğrulama işleminin sonucunu temsil eder.
// Bu yapı hem hataları hem de temizlenmiş, doğrulanmış veriyi tutar.
type ValidationResult struct {
	errors    map[string][]string // Alan bazlı doğrulama hataları
	validData map[string]any      // Doğrulanmış ve temizlenmiş veriler
}

// NewResult, yeni bir ValidationResult nesnesi oluşturur.
// Bu nesne, doğrulama işlemleri için başlangıç noktasıdır.
//
// Döndürür:
//   - *ValidationResult: Boş hata ve validData haritası ile yeni nesne
func NewResult() *ValidationResult {
	return &ValidationResult{
		errors:    make(map[string][]string),
		validData: make(map[string]any),
	}
}

// AddError, belirtilen alan için bir doğrulama hatası ekler.
//
// Parametreler:
//   - field: Hatanın ait olduğu alan adı
//   - message: Hata mesajı
func (r *ValidationResult) AddError(field, message string) {
	r.errors[field] = append(r.errors[field], message)
}

// HasErrors, ValidationResult içinde herhangi bir hata olup olmadığını kontrol eder.
//
// Döndürür:
//   - bool: Eğer en az bir hata varsa true, aksi halde false
func (r *ValidationResult) HasErrors() bool {
	return len(r.errors) > 0
}

// Errors, doğrulama sırasında oluşan tüm hataları döndürür.
//
// Döndürür:
//   - map[string][]string: Alan bazlı hata mesajları
func (r *ValidationResult) Errors() map[string][]string {
	return r.errors
}

// ValidData, doğrulama sırasında temizlenmiş ve onaylanmış veriyi döndürür.
//
// Döndürür:
//   - map[string]any: Doğrulanmış veri haritası
func (r *ValidationResult) ValidData() map[string]any {
	return r.validData
}

// SetValidData, doğrulama sonucu elde edilen temiz veriyi ayarlar.
// Bu metot sayesinde validData manuel olarak güncellenebilir.
//
// Parametre:
//   - data: Doğrulanmış veri haritası
func (r *ValidationResult) SetValidData(data map[string]any) {
	r.validData = data
}

// --- ARAYÜZLER (INTERFACES / CONTRACTS) ---

// Her veri tipi (örn: StringType, NumberType) bu arayüzü uygulamalıdır.
// Bu yapı, alan bazlı doğrulama ve ön işleme (transform) mekanizmasını sağlar.
type Type interface {
	// Validate, asıl doğrulama mantığını çalıştırır.
	// Parametreler:
	//   - field: Doğrulanan alan adı
	//   - value: Doğrulanan alanın değeri
	//   - result: ValidationResult nesnesi, hatalar buraya eklenir
	Validate(field string, value any, result *ValidationResult)

	// Transform, doğrulama öncesinde veriyi temizler ve dönüştürür.
	// Örnek: string trim, sayısal tip dönüşümü vb.
	//
	// Parametre:
	//   - value: İşlenecek veri
	// Döndürür:
	//   - any: Dönüştürülmüş veri
	//   - error: Dönüşüm sırasında hata oluşursa
	Transform(value any) (any, error)
}

// Tüm veri setini doğrulamak ve şema tanımlamak için kullanılır.
type Schema interface {
	// Validate, verilen veri haritasını doğrular ve ValidationResult döndürür.
	// Parametre:
	//   - data: Doğrulanacak veri haritası
	// Döndürür:
	//   - *ValidationResult: Doğrulama sonucu
	Validate(data map[string]any) *ValidationResult

	// Shape, şemada alan tiplerini tanımlar.
	// Parametre:
	//   - shape: Alan adı -> Type eşlemesi
	// Döndürür:
	//   - Schema: Aynı şema nesnesi (method chaining)
	Shape(shape map[string]Type) Schema

	// CrossValidate, alanlar arası doğrulama yapmak için kullanılabilir.
	// Parametre:
	//   - fn: cross validation fonksiyonu, hata oluşursa error döndürür
	// Döndürür:
	//   - Schema: Aynı şema nesnesi (method chaining)
	CrossValidate(fn func(data map[string]any) error) Schema

	// Bir alanın değeri beklenen değerle eşleşirse,
	// callback'den dönen alt şemayı (sub-schema) da doğrulamaya dahil eder.
	When(field string, expectedValue any, callback func() Schema) Schema
}
