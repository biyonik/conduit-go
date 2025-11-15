// pkg/validation/rules/payment.go
package rules

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// luhnCheck, Luhn algoritmasını uygular.
// PHP'deki isValidCreditCard içindeki algoritmadır.
func luhnCheck(number string) bool {
	var sum int
	isEvenIndex := len(number)%2 == 0

	for _, digitChar := range number {
		digit, err := strconv.Atoi(string(digitChar))
		if err != nil {
			return false // Sayısal olmayan karakter
		}

		if isEvenIndex {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isEvenIndex = !isEvenIndex
	}
	return sum%10 == 0
}

// IsValidCreditCard, bir kredi kartı numarasını doğrular.
func IsValidCreditCard(cardNumber string, cardType string) bool {
	// Boşluk ve tireleri kaldır (PHP'deki preg_replace)
	number := regexp.MustCompile(`\D`).ReplaceAllString(cardNumber, "")

	// Kart tipi desenleri
	patterns := map[string]*regexp.Regexp{
		"visa":       regexp.MustCompile(`^4[0-9]{12}(?:[0-9]{3})?$`),
		"mastercard": regexp.MustCompile(`^5[1-5][0-9]{14}$`),
		"amex":       regexp.MustCompile(`^3[47][0-9]{13}$`),
	}

	// Kart tipi kontrolü
	if cardType != "" {
		pattern, ok := patterns[cardType]
		if !ok || !pattern.MatchString(number) {
			return false // Belirtilen tip değil
		}
	}

	// Luhn algoritması
	return luhnCheck(number)
}

// IsValidIBAN, bir IBAN numarasını doğrular.
// DİKKAT: PHP'deki 'bcmod' büyük sayılar içindir. Go'da bu
// 'math/big' paketi ile yapılır.
func IsValidIBAN(iban string, countryCode string) bool {
	iban = strings.ToUpper(strings.ReplaceAll(iban, " ", ""))

	if countryCode != "" {
		expectedLength, ok := ibanCountryLengths[countryCode]
		if !ok || len(iban) != expectedLength {
			return false
		}
	}

	if match, _ := regexp.MatchString(`^[A-Z]{2}\d{2}[A-Z0-9]{4,}$`, iban); !match {
		return false
	}

	rearranged := iban[4:] + iban[:4]

	converted := ""
	for _, char := range rearranged {
		if unicode.IsLetter(char) {
			converted += strconv.Itoa(int(char - 'A' + 10))
		} else {
			converted += string(char)
		}
	}

	ibanNum := new(big.Int)
	ibanNum, ok := ibanNum.SetString(converted, 10)
	if !ok {
		return false
	}

	mod97 := new(big.Int)
	mod97.SetInt64(97)

	remainder := new(big.Int)
	remainder.Mod(ibanNum, mod97)

	return remainder.Int64() == 1
}

var ibanCountryLengths = map[string]int{
	"TR": 26, "DE": 22, "GB": 22, "FR": 27, "IT": 27, "NL": 18,
}
