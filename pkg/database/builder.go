package database

import (
	"database/sql"
	_ "fmt"
	"strings"
)

// -----------------------------------------------------------------------------
// QUERY BUILDER — TEMEL (GÜVENLİK İYİLEŞTİRMELERİ İLE)
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder'ın ana gövdesini içerir. Builder; tablo, kolonlar,
// where'lar, order, limit, offset gibi state bilgilerini tutar. Ayrıca gelişmiş
// CRUD metodları (Insert, Update, Delete, Get, First, Exec) bu yapı üzerinden
// sağlanır.
//
// GÜVENLİK İYİLEŞTİRMELERİ:
// - OrderBy artık OrderClause kullanıyor (SQL injection koruması)
// - Direction parametresi whitelist kontrolünden geçiyor
// - Tüm kullanıcı input'ları prepared statement'lar ile bağlanıyor
// -----------------------------------------------------------------------------

type QueryBuilder struct {
	executor QueryExecutor
	grammar  Grammar
	table    string
	columns  []string
	wheres   []WhereClause
	orders   []OrderClause // string yerine OrderClause kullanıyoruz (GÜVENLİK)
	limit    int
	offset   int
}

// NewBuilder, veritabanı bağlantısını alarak yeni QueryBuilder üretir.
//
// Parametreler:
//   - executor: SQL komutlarını çalıştıracak executor (*sql.DB veya *sql.Tx)
//   - grammar: SQL dialect'ini yöneten grammar (MySQL, PostgreSQL, vb.)
//
// Döndürür:
//   - *QueryBuilder: Yeni QueryBuilder instance'ı
func NewBuilder(executor QueryExecutor, grammar Grammar) *QueryBuilder {
	return &QueryBuilder{
		executor: executor,
		grammar:  grammar,
		columns:  []string{"*"},
		limit:    0,
		offset:   0,
	}
}

// Table, sorgunun çalışacağı tablo adını belirler.
//
// Parametre:
//   - tableName: Tablo adı
//
// Döndürür:
//   - *QueryBuilder: Zincirleme (method chaining) için kendi instance'ını döner
//
// Örnek:
//
//	qb.Table("users")
func (qb *QueryBuilder) Table(tableName string) *QueryBuilder {
	qb.table = tableName
	return qb
}

// Select, sorgudan döndürülecek kolonları belirler.
//
// Parametre:
//   - columns: Seçilecek kolon adları (variadic)
//
// Döndürür:
//   - *QueryBuilder: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	qb.Select("id", "name", "email")
//	qb.Select("COUNT(*) as total")
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Where, sorguya bir WHERE koşulu ekler.
// Tüm değerler prepared statement ile bağlandığı için SQL injection korumalıdır.
//
// Parametreler:
//   - column: Koşul uygulanacak kolon adı
//   - operator: Karşılaştırma operatörü (=, !=, <, >, <=, >=, LIKE, IN, vb.)
//   - value: Karşılaştırılacak değer
//
// Döndürür:
//   - *QueryBuilder: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	qb.Where("status", "=", "active")
//	qb.Where("age", ">", 18)
//	qb.Where("name", "LIKE", "%john%")
//
// Güvenlik Notu:
// Operator whitelist kontrolü Grammar katmanında yapılır.
func (qb *QueryBuilder) Where(column string, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereClause{
		Column:   column,
		Operator: operator,
		Value:    value,
		Boolean:  "AND",
	})
	return qb
}

// OrWhere, sorguya bir OR WHERE koşulu ekler.
//
// Parametreler:
//   - column: Koşul uygulanacak kolon adı
//   - operator: Karşılaştırma operatörü
//   - value: Karşılaştırılacak değer
//
// Döndürür:
//   - *QueryBuilder: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	qb.Where("role", "=", "admin").OrWhere("role", "=", "moderator")
//	→ SQL: WHERE `role` = ? OR `role` = ?
func (qb *QueryBuilder) OrWhere(column string, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereClause{
		Column:   column,
		Operator: operator,
		Value:    value,
		Boolean:  "OR",
	})
	return qb
}

// OrderBy, sorgu sonuçlarını belirtilen kolona göre sıralar.
//
// GÜVENLİK İYİLEŞTİRMESİ:
// Direction parametresi artık whitelist kontrolünden geçiyor.
// Sadece "ASC", "asc", "DESC", "desc" değerleri kabul edilir.
// Geçersiz değerler için varsayılan olarak "ASC" kullanılır.
//
// Parametreler:
//   - column: Sıralama yapılacak kolon adı
//   - direction: Sıralama yönü ("ASC" veya "DESC", case-insensitive)
//
// Döndürür:
//   - *QueryBuilder: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	qb.OrderBy("created_at", "DESC")
//	qb.OrderBy("name", "asc")
//
// Güvenlik Notu:
// Geçersiz direction değerleri otomatik olarak "ASC"e dönüştürülür.
// Bu sayede SQL injection riski tamamen ortadan kalkar.
func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder {
	// Direction'ı normalize et ve whitelist kontrolü yap
	dir := strings.ToUpper(strings.TrimSpace(direction))

	var orderDir OrderDirection
	switch dir {
	case "DESC":
		orderDir = OrderDesc
	case "ASC":
		orderDir = OrderAsc
	default:
		// Geçersiz değer için varsayılan olarak ASC kullan
		// Bu sayede "DESC; DROP TABLE users--" gibi injection denemeleri etkisiz kalır
		orderDir = OrderAsc
	}

	qb.orders = append(qb.orders, OrderClause{
		Column:    column,
		Direction: orderDir,
	})
	return qb
}

// Limit, döndürülecek maksimum satır sayısını belirler.
//
// Parametre:
//   - limit: Maksimum satır sayısı
//
// Döndürür:
//   - *QueryBuilder: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	qb.Limit(10) → LIMIT 10
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset, atlanacak satır sayısını belirler (pagination için).
//
// Parametre:
//   - offset: Atlanacak satır sayısı
//
// Döndürür:
//   - *QueryBuilder: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	qb.Limit(10).Offset(20) → LIMIT 10 OFFSET 20 (3. sayfa)
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Get, sorguyu çalıştırır ve sonuçları bir struct slice'ına tarar.
//
// Parametre:
//   - dest: Sonuçların doldurulacağı slice pointer (örn: &[]models.User)
//
// Döndürür:
//   - error: Sorgu veya tarama hatası varsa
//
// Örnek:
//
//	var users []User
//	err := qb.Table("users").Where("status", "=", "active").Get(&users)
//
// Güvenlik Notu:
// Tüm parametreler prepared statement ile bağlandığı için SQL injection korumalıdır.
func (qb *QueryBuilder) Get(dest any) error {
	sqlStr, args := qb.ToSQL()

	rows, err := qb.executor.Query(sqlStr, args...)
	if err != nil {
		return err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			panic(err)
		}
	}(rows)

	// Reflection-based scanner kullanarak sonuçları tara
	return ScanSlice(rows, dest)
}

// First, sorguyu çalıştırır (otomatik 'LIMIT 1' ekler) ve
// ilk sonucu tek bir struct'a tarar.
//
// Parametre:
//   - dest: Sonucun doldurulacağı struct pointer (örn: &models.User)
//
// Döndürür:
//   - error: Sorgu hatası, satır bulunamazsa sql.ErrNoRows döner
//
// Örnek:
//
//	var user User
//	err := qb.Table("users").Where("id", "=", 1).First(&user)
//	if err == sql.ErrNoRows {
//	    // Kullanıcı bulunamadı
//	}
func (qb *QueryBuilder) First(dest any) error {
	// First her zaman LIMIT 1 olmalıdır
	qb.Limit(1)

	sqlStr, args := qb.ToSQL()

	rows, err := qb.executor.Query(sqlStr, args...)
	if err != nil {
		return err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			panic(err)
		}
	}(rows)

	// İlk satıra ilerle
	if !rows.Next() {
		// Satır bulunamadı
		return sql.ErrNoRows
	}

	// Reflection-based scanner kullanarak sonucu tara
	return ScanStruct(rows, dest)
}

// ToSQL, QueryBuilder'ın state'ini SQL string'e ve parametrelere dönüştürür.
// Bu metod Grammar katmanına delegate eder.
//
// Döndürür:
//   - string: Oluşturulan SQL query
//   - []interface{}: Prepared statement parametreleri
//
// Örnek:
//
//	sql, args := qb.ToSQL()
//	// sql: "SELECT `id`, `name` FROM `users` WHERE `status` = ? ORDER BY `created_at` DESC LIMIT 10"
//	// args: ["active"]
func (qb *QueryBuilder) ToSQL() (string, []interface{}) {
	return qb.grammar.CompileSelect(qb)
}

// ExecInsert, INSERT sorgusunu çalıştırır.
//
// Parametre:
//   - data: Eklenecek veri (kolon adı -> değer mapping)
//
// Döndürür:
//   - sql.Result: LastInsertId() ve RowsAffected() metodlarını içerir
//   - error: Sorgu hatası varsa
//
// Örnek:
//
//	result, err := qb.ExecInsert(map[string]interface{}{
//	    "name": "John Doe",
//	    "email": "john@example.com",
//	})
//	lastID, _ := result.LastInsertId()
func (qb *QueryBuilder) ExecInsert(data map[string]interface{}) (sql.Result, error) {
	sqlStr, args := qb.grammar.CompileInsert(qb.table, data)
	return qb.executor.Exec(sqlStr, args...)
}

// ExecUpdate, UPDATE sorgusunu çalıştırır.
//
// Parametre:
//   - data: Güncellenecek veri (kolon adı -> değer mapping)
//
// Döndürür:
//   - sql.Result: RowsAffected() metodunu içerir
//   - error: Sorgu hatası varsa
//
// Örnek:
//
//	result, err := qb.Table("users").
//	    Where("id", "=", 1).
//	    ExecUpdate(map[string]interface{}{
//	        "name": "Jane Doe",
//	    })
//	affected, _ := result.RowsAffected()
//
// Güvenlik Notu:
// WHERE clause olmadan UPDATE çalıştırmak tehlikelidir!
// Production'da mutlaka WHERE kontrolü eklenmelidir.
func (qb *QueryBuilder) ExecUpdate(data map[string]interface{}) (sql.Result, error) {
	sqlStr, args := qb.grammar.CompileUpdate(qb.table, data, qb.wheres)
	return qb.executor.Exec(sqlStr, args...)
}

// ExecDelete, DELETE sorgusunu çalıştırır.
//
// Döndürür:
//   - sql.Result: RowsAffected() metodunu içerir
//   - error: Sorgu hatası varsa
//
// Örnek:
//
//	result, err := qb.Table("users").
//	    Where("status", "=", "inactive").
//	    ExecDelete()
//	affected, _ := result.RowsAffected()
//
// GÜVENLİK UYARISI:
// WHERE clause olmadan DELETE çalıştırmak TÜM TABLONUN SİLİNMESİNE sebep olur!
// Production'da mutlaka WHERE kontrolü eklenmelidir.
func (qb *QueryBuilder) ExecDelete() (sql.Result, error) {
	sqlStr, args := qb.grammar.CompileDelete(qb.table, qb.wheres)
	return qb.executor.Exec(sqlStr, args...)
}
