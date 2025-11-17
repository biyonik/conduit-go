// -----------------------------------------------------------------------------
// Password Hashing Package
// -----------------------------------------------------------------------------
// Bu dosya, kullanıcı şifrelerinin güvenli bir şekilde hash'lenmesi ve
// doğrulanması için fonksiyonlar sağlar. bcrypt algoritması kullanılır.
//
// bcrypt neden?
// - Brute force saldırılarına karşı yavaş (kasıtlı olarak)
// - Salt otomatik olarak eklenir
// - Zaman içinde cost factor artırılabilir (güvenlik artışı)
// - Endüstri standardı
//
// Güvenlik Notu:
// - Minimum cost: 10 (development), 12 (production)
// - Her şifre için unique salt kullanılır (bcrypt otomatik halleder)
// - Rainbow table saldırılarına karşı korumalıdır
// -----------------------------------------------------------------------------

package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashCost, bcrypt hash algoritmasının maliyet faktörüdür.
// Yüksek değer = daha güvenli ama daha yavaş
//
// Önerilen değerler:
//   - Development: 10 (hızlı test için)
//   - Production: 12-14 (güvenlik için)
//   - High Security: 15+ (bankacılık gibi kritik sistemler)
const HashCost = 12

// Hash, düz metin şifreyi bcrypt ile hash'ler.
//
// Parametre:
//   - password: Hash'lenecek düz metin şifre
//
// Döndürür:
//   - string: Bcrypt hash'i (60 karakter, $2a$ ile başlar)
//   - error: Hash işlemi başarısız olursa
//
// Örnek:
//
//	hashed, err := auth.Hash("mySecretPassword123")
//	// hashed: "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYvXr6rKW9W"
//
// Güvenlik Notu:
// - Asla orijinal şifreyi veritabanına kaydetmeyin!
// - Hash'i kaydedin, doğrulama için Check() kullanın
func Hash(password string) (string, error) {
	// Boş şifre kontrolü
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// bcrypt ile hash oluştur
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), HashCost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// Check, düz metin şifreyi hash ile karşılaştırır.
//
// Parametreler:
//   - password: Kullanıcının girdiği düz metin şifre
//   - hash: Veritabanında saklanan bcrypt hash'i
//
// Döndürür:
//   - bool: Şifre eşleşiyorsa true, değilse false
//
// Örnek:
//
//	isValid := auth.Check("mySecretPassword123", hashedFromDB)
//	if !isValid {
//	    return errors.New("invalid credentials")
//	}
//
// Güvenlik Notu:
// - Bu fonksiyon kasıtlı olarak yavaştır (timing attack koruması)
// - Hatalı şifre için bile aynı sürede döner
func Check(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// NeedsRehash, mevcut hash'in yeni cost factor ile tekrar hash'lenmesi
// gerekip gerekmediğini kontrol eder.
//
// Parametre:
//   - hash: Kontrol edilecek bcrypt hash'i
//
// Döndürür:
//   - bool: Yeniden hash gerekiyorsa true
//
// Kullanım Senaryosu:
// Zaman içinde güvenlik standartları değişir. Eski kullanıcıların şifreleri
// düşük cost factor ile hash'lenmiş olabilir. Bu fonksiyon, kullanıcı login
// olduğunda şifresinin yeni standarda göre güncellenmesi gerekip gerekmediğini
// söyler.
//
// Örnek:
//
//	if auth.Check(password, user.Password) {
//	    // Şifre doğru, ama güncel mi?
//	    if auth.NeedsRehash(user.Password) {
//	        // Yeni hash oluştur ve güncelle
//	        newHash, _ := auth.Hash(password)
//	        db.Update(user.ID, newHash)
//	    }
//	    // Login başarılı
//	}
func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false
	}
	return cost < HashCost
}

// MustHash, Hash fonksiyonunun panic atan versiyonudur.
// Test veya seed data oluştururken kullanışlıdır.
//
// Parametre:
//   - password: Hash'lenecek şifre
//
// Döndürür:
//   - string: Bcrypt hash'i
//
// Panic:
// Hash işlemi başarısız olursa panic atar
//
// Örnek (seed data):
//
//	users := []User{
//	    {Email: "admin@example.com", Password: auth.MustHash("secret")},
//	    {Email: "user@example.com", Password: auth.MustHash("password")},
//	}
//
// UYARI: Production kodunda MustHash kullanmayın! Sadece test/seed için.
func MustHash(password string) string {
	hash, err := Hash(password)
	if err != nil {
		panic("failed to hash password: " + err.Error())
	}
	return hash
}
