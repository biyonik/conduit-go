// pkg/validation/rules/password.go
package rules

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// PasswordRules, PHP'deki $passwordRules dizisinin Go struct karşılığıdır.
type PasswordRules struct {
	MinLength         int
	MaxLength         int
	RequireUppercase  bool
	RequireLowercase  bool
	RequireNumeric    bool
	RequireSpecial    bool
	SpecialChars      string
	MinUniqueChars    int
	MaxRepeatingChars int
	DisallowCommon    bool
	DisallowKeyboard  bool
	MinEntropy        float64
}

// commonPasswords, PHP'deki hardcoded listeye karşılık gelir.
var commonPasswords = map[string]bool{
	"password": true, "123456": true, "qwerty": true, "111111": true, "abc123": true,
	"letmein": true, "admin": true, "welcome": true, "monkey": true, "dragon": true,
}

// keyboardPatterns, PHP'deki $keyboardPatterns listesi.
var keyboardPatterns = []string{
	"qwerty", "asdfgh", "zxcvbn",
	"123456", "654321",
	"abc", "cba", "xyz",
}

// stringReverse, PHP'deki strrev() fonksiyonunu Go'da uygular.
func stringReverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// hasKeyboardPattern, PHP'deki hasKeyboardPattern() portu.
func hasKeyboardPattern(password string) bool {
	loweredPass := strings.ToLower(password)
	for _, pattern := range keyboardPatterns {
		if strings.Contains(loweredPass, pattern) || strings.Contains(loweredPass, stringReverse(pattern)) {
			return true
		}
	}
	return false
}

// hasRepeatingChars, PHP'deki hasRepeatingChars() portu.
func hasRepeatingChars(password string, maxRepeats int) bool {
	if maxRepeats <= 0 {
		return false
	}
	chars := []rune(password)
	consecutive := 1
	var lastChar rune

	for i, char := range chars {
		if i > 0 && char == lastChar {
			consecutive++
			if consecutive > maxRepeats {
				return true
			}
		} else {
			consecutive = 1
		}
		lastChar = char
	}
	return false
}

// calculatePasswordEntropy, PHP'deki calculatePasswordEntropy() portu.
func calculatePasswordEntropy(password string) float64 {
	length := float64(len(password))
	if length == 0 {
		return 0
	}
	charPool := 0.0

	if regexp.MustCompile(`[a-z]`).MatchString(password) {
		charPool += 26
	}
	if regexp.MustCompile(`[A-Z]`).MatchString(password) {
		charPool += 26
	}
	if regexp.MustCompile(`[0-9]`).MatchString(password) {
		charPool += 10
	}
	if regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		charPool += 32 // PHP'deki gibi 32 varsayalım
	}

	if charPool == 0 {
		return 0
	}

	// PHP'deki log(pool, 2) Go'da math.Log2(pool) demektir.
	return length * math.Log2(charPool)
}

// ValidatePassword, PHP'deki validatePassword() metodunun ana mantığıdır.
// Hata mesajlarını bir slice olarak döndürür.
func ValidatePassword(password string, rules *PasswordRules) []string {
	errors := []string{}
	if rules == nil {
		return errors
	}

	passLen := len(password)

	if passLen < rules.MinLength {
		errors = append(errors, fmt.Sprintf("en az %d karakter uzunluğunda olmalıdır", rules.MinLength))
	}
	if passLen > rules.MaxLength {
		errors = append(errors, fmt.Sprintf("en fazla %d karakter uzunluğunda olmalıdır", rules.MaxLength))
	}
	if rules.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		errors = append(errors, "en az bir büyük harf içermelidir")
	}
	if rules.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		errors = append(errors, "en az bir küçük harf içermelidir")
	}
	if rules.RequireNumeric && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		errors = append(errors, "en az bir rakam içermelidir")
	}
	if rules.RequireSpecial {
		specialChars := regexp.QuoteMeta(rules.SpecialChars)
		if !regexp.MustCompile(fmt.Sprintf("[%s]", specialChars)).MatchString(password) {
			errors = append(errors, fmt.Sprintf("en az bir özel karakter içermelidir (%s)", rules.SpecialChars))
		}
	}

	// Gelişmiş kontroller
	uniqueChars := make(map[rune]bool)
	for _, char := range password {
		uniqueChars[char] = true
	}
	if len(uniqueChars) < rules.MinUniqueChars {
		errors = append(errors, fmt.Sprintf("en az %d farklı karakter içermelidir", rules.MinUniqueChars))
	}

	if rules.DisallowKeyboard && hasKeyboardPattern(password) {
		errors = append(errors, "klavye düzeninde sıralı karakterler içeremez")
	}
	if rules.MaxRepeatingChars > 0 && hasRepeatingChars(password, rules.MaxRepeatingChars) {
		errors = append(errors, fmt.Sprintf("en fazla %d adet tekrar eden karakter içerebilir", rules.MaxRepeatingChars))
	}
	if rules.DisallowCommon && commonPasswords[strings.ToLower(password)] {
		errors = append(errors, "çok yaygın bir şifre, lütfen daha güvenli bir şifre seçin")
	}

	entropy := calculatePasswordEntropy(password)
	if entropy < rules.MinEntropy {
		errors = append(errors, "yeterince karmaşık değil, lütfen daha güçlü bir şifre seçin")
	}

	return errors
}
