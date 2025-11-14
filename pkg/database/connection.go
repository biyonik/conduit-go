// -----------------------------------------------------------------------------
// Database Package
// -----------------------------------------------------------------------------
// Bu dosya, uygulamanın MySQL veritabanına bağlanmasını sağlayan merkezi
// bağlantı fonksiyonunu içerir. Laravel veya Symfony frameworklerinde olduğu
// gibi, veritabanı bağlantısı yapılandırmasını merkezi bir noktadan yönetir.
//
// Buradaki Connect fonksiyonu, DSN (Data Source Name) parametresi alır,
// bağlantıyı başlatır ve bağlantı havuzlaması ile performans optimizasyonu
// sağlar. Bağlantı başarılı olduğunda db nesnesi geri döndürülür, hata
// durumunda uygun error handling yapılır.
// -----------------------------------------------------------------------------

package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Connect, verilen DSN ile MySQL veritabanına bağlanır ve *sql.DB nesnesini döndürür.
// Bağlantı sırasında şu adımlar gerçekleştirilir:
//  1. sql.Open ile sürücü ve DSN kullanılarak bağlantı nesnesi oluşturulur.
//  2. Bağlantı havuzu için max open ve idle connection değerleri belirlenir.
//  3. Bağlantı ömrü (ConnMaxLifetime) 5 dakika olarak ayarlanır.
//  4. db.Ping ile veritabanının ulaşılabilirliği kontrol edilir.
//  5. Başarılı olursa db nesnesi döndürülür, hata varsa connection kapatılır ve error döner.
func Connect(dsn string) (*sql.DB, error) {

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err // Bağlantı açma hatası
	}

	// Bağlantı havuzu ayarları: performans ve kaynak yönetimi için
	db.SetMaxOpenConns(25)                 // Maksimum açık bağlantı sayısı
	db.SetMaxIdleConns(25)                 // Maksimum idle bağlantı sayısı
	db.SetConnMaxLifetime(5 * time.Minute) // Bağlantı ömrü

	log.Println("Veritabanına bağlanılıyor...")
	err = db.Ping() // Gerçek bağlantıyı test et
	if err != nil {
		db.Close() // Hata durumunda bağlantıyı kapat
		return nil, err
	}

	log.Println("✅ Veritabanı bağlantısı başarılı!")
	return db, nil
}
