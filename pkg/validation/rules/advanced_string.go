package rules

import "regexp"

// HasTurkishChars, PHP'deki hasTurkishChars portudur.
// Go'da 'rune' kullanarak multi-byte karakterleri doğru sayarız.
func HasTurkishChars(text string) bool {
	// PHP'deki [çÇğĞıİöÖşŞüÜ] listesi
	match, _ := regexp.MatchString(`[çÇğĞıİöÖşŞüÜ]`, text)
	return match
}

// IsValidDomain, PHP'deki isValidDomain portudur.
// Go için basitleştirilmiş bir regex kullanıyoruz.
func IsValidDomain(domain string, allowSubdomain bool) bool {
	// (PHP'deki regex'in basitleştirilmiş hali)
	var pattern *regexp.Regexp
	if allowSubdomain {
		// Alt alan adı dahil (örn: blog.example.com)
		pattern = regexp.MustCompile(`^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$`)
	} else {
		// Sadece kök alan adı (örn: example.com)
		pattern = regexp.MustCompile(`^[a-zA-Z0-9-]+\.[a-zA-Z]{2,}$`)
	}
	return pattern.MatchString(domain)
}
