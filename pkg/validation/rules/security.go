// pkg/validation/rules/security.go
package rules

import (
	"regexp"
	"strings"
)

// allowedTagsPattern, izin verilen etiketleri ayıklamak için bir regex.
// Bu, PHP'deki strip_tags fonksiyonunun ikinci parametresinin davranışını
// (örn: "<p><a>") taklit etmek için basit bir regex'tir.
// Go'da strip_tags'in birebir karşılığı olmadığı için,
// kaba bir temizlik yapacağız.
//
// Not: Go'da HTML temizliği için genellikle "golang.org/x/net/html" veya
// "github.com/microcosm-cc/bluemonday" gibi kütüphaneler kullanılır.
// Ancak "hiç paket kullanmama" hedefimiz doğrultusunda,
// PHP'deki basit `strip_tags`'ı taklit edeceğiz.
var (
	// Tüm HTML etiketlerini yakalayan regex
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
	// İzin verilen etiketleri yakalamak için (basit hali)
	allowedTagRegex *regexp.Regexp

	// FilterEmoji
	emojiRegex = regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}]`)

	// SanitizeFilename için: Güvensiz karakterler
	unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9\-\_\.]`)
	consecutiveDots     = regexp.MustCompile(`\.{2,}`)
	leadingDot          = regexp.MustCompile(`^\.`)
	trailingDot         = regexp.MustCompile(`\.$`)

	// CharSetPatterns
	CharSetPatterns = map[string]*regexp.Regexp{
		"latin":        regexp.MustCompile(`^[\p{Latin}]+$`),
		"alphanumeric": regexp.MustCompile(`^[a-zA-Z0-9]+$`),
		"numeric":      regexp.MustCompile(`^[0-9]+$`),
		"alpha":        regexp.MustCompile(`^[a-zA-Z]+$`),
	}

	// SanitizeFilename için Türkçe karakter haritası
	turkishCharReplacer = strings.NewReplacer(
		"ç", "c", "Ç", "C",
		"ğ", "g", "Ğ", "G",
		"ı", "i", "İ", "I",
		"ö", "o", "Ö", "O",
		"ş", "s", "Ş", "S",
		"ü", "u", "Ü", "U",
	)
)

// StripHtmlTags, PHP'deki stripHtmlTags metodunu taklit eder.
// Go'nun standart kütüphanesinde bu işlevin tam bir karşılığı yoktur,
// bu nedenle basit bir regex tabanlı yaklaşım kullanıyoruz.
func StripHtmlTags(input string, allowedTags ...string) string {
	if len(allowedTags) == 0 {
		// İzin verilen etiket yoksa, tüm etiketleri kaldır
		return htmlTagRegex.ReplaceAllString(input, "")
	}

	// İzin verilen etiketler varsa, daha karmaşık bir temizlik gerekir.
	// Örn: "<p><a>" -> (p|a)
	// Bu, 'bluemonday' gibi bir kütüphanenin alanıdır.
	// Şimdilik basit bir "tümünü kaldır" implementasyonu ile devam edelim
	// ve daha sonra bu fonksiyonu geliştirelim.
	//
	// TODO: allowedTags listesini destekleyen daha gelişmiş bir
	// regex motoru veya HTML parser (net/html) yazılmalı.
	// Şimdilik, bu fonksiyonun temel amacı 'strip_tags($input)'
	// (ikinci parametre olmadan) davranışıdır.

	// Geçici olarak, 'allowedTags' parametresini yok sayıp tümünü temizleyelim.
	// Gerçek bir HTML parser olmadan 'allowedTags'ı güvenli şekilde
	// implemente etmek çok zordur.
	return htmlTagRegex.ReplaceAllString(input, "")
}

// PreventXss, PHP'deki preventXss metodunu port eder.
// Bu, Go'daki `html.EscapeString` ile aynı işi yapar.
// (Ancak 'html' paketini import etmemek için manuel yapıyoruz)
func PreventXss(input string) string {
	// html.EscapeString'in manuel implementasyonu
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, `"`, "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")
	return input
}

// SanitizeFilename, PHP'deki sanitizeFilename metodunu port eder.
func SanitizeFilename(filename string) string {
	// 1. Türkçe karakterleri düzelt
	filename = turkishCharReplacer.Replace(filename)

	// 2. Regex ile temizle
	filename = unsafeFilenameChars.ReplaceAllString(filename, "")
	filename = consecutiveDots.ReplaceAllString(filename, ".")
	filename = leadingDot.ReplaceAllString(filename, "")
	filename = trailingDot.ReplaceAllString(filename, "")

	// 3. Maksimum uzunluk (PHP'deki gibi)
	if len(filename) > 255 {
		filename = filename[:255]
	}
	return filename
}

// FilterEmoji, PHP'deki filterEmoji metodunu port eder.
func FilterEmoji(input string, remove bool) string {
	if remove {
		return emojiRegex.ReplaceAllString(input, "")
	}
	return input
}

// ValidateCharSet, PHP'deki validateCharSet metodunu port eder.
func ValidateCharSet(input string, charSet string) bool {
	pattern, ok := CharSetPatterns[charSet]
	if !ok {
		return false // Bilinmeyen karakter seti
	}
	return pattern.MatchString(input)
}
