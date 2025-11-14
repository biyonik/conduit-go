// pkg/database/transaction.go
//
// Bu dosya, ORM altyapısının en kritik parçalarından biri olan
// veritabanı işlemlerinin (transaction) güvenli, yönetilebilir ve
// okunabilir bir şekilde kontrol edilmesini sağlar.
//
// Bir transaction; ACID prensiplerine uygun olarak bir grup veritabanı
// işleminin tamamının *ya tamamen başarılı olmasını* ya da *hiçbirinin
// uygulanmamış kabul edilmesini* sağlar. Özellikle birden fazla tablonun
// veya karmaşık CRUD işlemlerinin yer aldığı senaryolarda, veri bütünlüğü
// için hayati önem taşır.
//
// Buradaki Transaction yapısı, Go'nun sql.Tx tipine bir sarmalayıcıdır.
// Böylece hem okunabilirliği artırır hem de ORM mimarisi içinde standart
// bir transaction kullanım modeli sunar.
//
// Örnek kullanım:
//
//   tx, _ := BeginTransaction(db)
//   qb := NewBuilder(tx.Tx) // builder transaction içinde çalışır
//   qb.Table(\"users\").Where(\"id\", \"=\", 1).Update(...)
//   tx.Commit()
//
// Eğer işlem sırasında hata olursa:
//   tx.Rollback()

package database

import (
    "database/sql"
    "log"
)

// Transaction
//
// Veritabanı transaction yapısını temsil eder.
// sql.Tx nesnesini saklar ve commit/rollback operasyonlarını
// daha okunabilir bir API ile gerçekleştirir.
type Transaction struct {
    Tx *sql.Tx
}

// BeginTransaction
//
// Yeni bir veritabanı transaction’ı başlatır.
// Başlatılan transaction'ı Transaction yapısı içinde sararak döndürür.
//
// Dönen Transaction yapısı mutlaka `Commit()` veya `Rollback()`
// ile sonlandırılmalıdır.
//
// Parametreler:
//   - db: *sql.DB — işlem yapılacak veritabanı havuzu
//
// Dönüş:
//   - *Transaction — başlatılan işlem
//   - error — başarısız olursa hata
func BeginTransaction(db *sql.DB) (*Transaction, error) {
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }

    log.Println("🔄 Transaction başladı.")
    return &Transaction{Tx: tx}, nil
}

// Commit
//
// Başlatılmış olan transaction’ı başarılı şekilde sonlandırır.
// Eğer hata oluşmazsa commit edildiğine dair log basar.
//
// Dönüş: error
func (t *Transaction) Commit() error {
    err := t.Tx.Commit()
    if err == nil {
        log.Println("✅ Transaction commit edildi.")
    }
    return err
}

// Rollback
//
// Transaction sırasında bir hata oluştuğunda çağrılır.
// Yapılmış tüm değişiklikler geri alınır.
//
// Dönüş: error
func (t *Transaction) Rollback() error {
    err := t.Tx.Rollback()
    if err == nil {
        log.Println("❌ Transaction geri alındı.")
    }
    return err
}
