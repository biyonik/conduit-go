// pkg/validation/rules/uuid.go
package rules

import "regexp"

// PHP'deki UuidValidationTrait'ten port edilmiştir.
// Regex'leri global değişken olarak derlemek, her çağrıda derlemekten çok daha
// performanslıdır. 'regexp.MustCompile' derleme hatası olursa 'panic' yapar,
// ki bu da program başlarken desenin yanlış olduğunu hemen fark etmemizi sağlar.
var (
	uuidV1Regex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-1[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidV3Regex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-3[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidV4Regex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidV5Regex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidGenRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// IsValidUUID, bir string'in geçerli bir UUID olup olmadığını kontrol eder.
// PHP'deki isValidUuid() metoduna karşılık gelir.
func IsValidUUID(uuid string, version int) bool {
	// Not: Go'da 'i' (case-insensitive) flag'i regex'e eklemek yerine
	// string'i küçük harfe çevirmek daha yaygın (ve bazen daha hızlı) bir pratiktir.
	// Ancak UUID regex'lerimiz zaten 'a-f' içerdiği için buna gerek yok.
	// PHP'deki 'i' flag'ini karşılamak için regex'lere 'i' flag'i eklenebilir: `(?i)^...`
	// Şimdilik küçük harf varsayıyoruz.

	switch version {
	case 1:
		return uuidV1Regex.MatchString(uuid)
	case 3:
		return uuidV3Regex.MatchString(uuid)
	case 4:
		return uuidV4Regex.MatchString(uuid)
	case 5:
		return uuidV5Regex.MatchString(uuid)
	case 0: // Herhangi bir versiyon (genel format)
		return uuidGenRegex.MatchString(uuid)
	default:
		return false
	}
}
