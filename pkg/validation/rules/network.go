// Package rules, network doğrulama kurallarını içerir.
// Bu dosya IP ve telefon numarası doğrulaması gibi network odaklı kuralları barındırır.
package rules

import (
	"net"    // IP doğrulaması için standart kütüphane
	"regexp" // Regex işlemleri için
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// IsValidIP, verilen IP adresinin geçerli olup olmadığını kontrol eder.
//
// Parametreler:
//   - ip: Doğrulanacak IP adresi (string)
//   - version: IP versiyonu. 4 = IPv4, 6 = IPv6, 0 = hem IPv4 hem IPv6
//
// Dönüş:
//   - bool: IP geçerliyse true, geçersizse false
func IsValidIP(ip string, version int) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	switch version {
	case 4:
		return parsedIP.To4() != nil
	case 6:
		return parsedIP.To4() == nil && parsedIP.To16() != nil
	case 0:
		return true
	default:
		return false
	}
}

// phonePatterns, ülke bazlı telefon numarası regexlerini tutar.
var phonePatterns = map[string]*regexp.Regexp{
	"TR": regexp.MustCompile(`^(05|5)[0-9]{9}$`),                    // Türkiye GSM
	"US": regexp.MustCompile(`^(\+1|1)?[2-9]\d{2}[2-9]\d{2}\d{4}$`), // ABD
}

// IsValidPhoneNumber, verilen telefon numarasının geçerli olup olmadığını kontrol eder.
//
// Parametreler:
//   - phone: Doğrulanacak telefon numarası (string)
//   - country: Ülke kodu (string), örn: "TR", "US"
//
// Dönüş:
//   - bool: Telefon numarası geçerliyse true, geçersizse false
func IsValidPhoneNumber(phone string, country string) bool {
	pattern, ok := phonePatterns[country]
	if !ok {
		return false
	}

	// Boşluk, tire ve parantezleri temizle
	cleanNumber := regexp.MustCompile(`\s+|-|\(|\)`).ReplaceAllString(phone, "")
	return pattern.MatchString(cleanNumber)
}
