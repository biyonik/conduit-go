package database

import (
	"fmt"
	"regexp"
	"strings"
)

// -----------------------------------------------------------------------------
// MySQL Grammar - SQL Üretim Katmanı (GÜVENLİK İYİLEŞTİRMELERİ İLE)
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder'ın state'ini MySQL lehçesinde SQL sorgularına
// dönüştürür. Grammar pattern sayesinde farklı SQL dialect'leri
// (PostgreSQL, SQLite, vb.) kolayca eklenebilir.
//
// GÜVENLİK İYİLEŞTİRMELERİ:
// - Wrap() fonksiyonu artık backtick injection'a karşı korumalı
// - Kolon/tablo isimleri regex ile validate ediliyor
// - Operator whitelist kontrolü eklendi
// - Tüm kullanıcı input'ları prepared statement ile bağlanıyor
// -----------------------------------------------------------------------------

// MySQLGrammar, MySQL-specific SQL üretim katmanıdır.
type MySQLGrammar struct{}

// NewMySQLGrammar, yeni bir MySQLGrammar instance'ı oluşturur.
func NewMySQLGrammar() *MySQLGrammar {
	return &MySQLGrammar{}
}

// validIdentifierPattern, geçerli SQL identifier pattern'ini tanımlar.
// Sadece alphanumeric karakterler, underscore ve nokta (tablo.kolon için) kabul edilir.
var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)

// allowedOperators, WHERE clause'larda kullanılabilecek güvenli operatörleri tanımlar.
// Bu whitelist sayesinde zararlı SQL komutları operatör olarak kullanılamaz.
var allowedOperators = map[string]bool{
	"=":           true,
	"!=":          true,
	"<>":          true,
	"<":           true,
	">":           true,
	"<=":          true,
	">=":          true,
	"LIKE":        true,
	"NOT LIKE":    true,
	"IN":          true,
	"NOT IN":      true,
	"BETWEEN":     true,
	"NOT BETWEEN": true,
	"IS":          true,
	"IS NOT":      true,
}

// Wrap, kolon ve tablo isimlerini MySQL backtick'leri ile sarmalar.
//
// GÜVENLİK İYİLEŞTİRMESİ:
// - Identifier'lar artık regex ile validate ediliyor
// - Geçersiz karakterler içeren identifier'lar panic'e sebep oluyor
// - Backtick injection riski tamamen ortadan kaldırıldı
//
// Parametre:
//   - value: Sarmalanacak identifier (kolon veya tablo adı)
//
// Döndürür:
//   - string: Backtick'lerle sarmalanmış identifier
//
// Örnek:
//
//	Wrap("users") → `users`
//	Wrap("users.id") → `users`.`id`
//	Wrap("*") → * (özel durum)
//
// Güvenlik Notu:
// Bu fonksiyon sadece güvenilir kaynaklardan gelen identifier'lar için
// kullanılmalıdır. Kullanıcı input'u ASLA direkt olarak buraya gelmemelidir.
func (g *MySQLGrammar) Wrap(value string) string {
	// Wildcard için özel durum
	if value == "*" {
		return value
	}

	// Tablo.kolon formatını handle et
	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		wrappedParts := make([]string, len(parts))
		for i, part := range parts {
			// Her parçayı validate et
			if !validIdentifierPattern.MatchString(part) {
				panic(fmt.Sprintf("Invalid SQL identifier: %s (contains unsafe characters)", part))
			}
			wrappedParts[i] = fmt.Sprintf("`%s`", part)
		}
		return strings.Join(wrappedParts, ".")
	}

	// Tek identifier'ı validate et
	if !validIdentifierPattern.MatchString(value) {
		panic(fmt.Sprintf("Invalid SQL identifier: %s (contains unsafe characters)", value))
	}

	return fmt.Sprintf("`%s`", value)
}

// validateOperator, verilen operatörün whitelist'te olup olmadığını kontrol eder.
//
// Parametre:
//   - operator: Kontrol edilecek operatör
//
// Panic:
// Operatör whitelist'te yoksa panic atar (SQL injection koruması)
func (g *MySQLGrammar) validateOperator(operator string) {
	op := strings.ToUpper(strings.TrimSpace(operator))
	if !allowedOperators[op] {
		panic(fmt.Sprintf("Invalid SQL operator: %s (not in whitelist)", operator))
	}
}

// CompileSelect, QueryBuilder'dan SELECT sorgusu üretir.
//
// Parametre:
//   - qb: QueryBuilder instance'ı
//
// Döndürür:
//   - string: Oluşturulan SQL query
//   - []interface{}: Prepared statement parametreleri
//
// Örnek:
//
//	CompileSelect(qb)
//	→ SQL: "SELECT `id`, `name` FROM `users` WHERE `status` = ? ORDER BY `created_at` DESC LIMIT 10"
//	→ Args: ["active"]
//
// Güvenlik Notu:
// Tüm değerler prepared statement placeholder'ları (?) ile bağlanır.
// Bu sayede SQL injection riski tamamen ortadan kalkar.
func (g *MySQLGrammar) CompileSelect(qb *QueryBuilder) (string, []interface{}) {
	// Kolonları wrap et
	wrappedCols := make([]string, len(qb.columns))
	for i, col := range qb.columns {
		wrappedCols[i] = g.Wrap(col)
	}

	// SELECT ... FROM ... kısmını oluştur
	sql := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(wrappedCols, ", "),
		g.Wrap(qb.table),
	)

	var args []interface{}

	// WHERE clause'ları ekle
	if len(qb.wheres) > 0 {
		sql += " WHERE "
		for i, w := range qb.wheres {
			// Operatörü validate et (SQL injection koruması)
			g.validateOperator(w.Operator)

			// AND/OR ekle (ilk koşul için değil)
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}

			// Koşulu ekle (değer prepared statement ile bağlanacak)
			sql += fmt.Sprintf("%s %s ?", g.Wrap(w.Column), strings.ToUpper(w.Operator))
			args = append(args, w.Value)
		}
	}

	// ORDER BY clause'ları ekle (GÜVENLİK İYİLEŞTİRMESİ: OrderClause kullanıyor)
	if len(qb.orders) > 0 {
		wrappedOrders := make([]string, len(qb.orders))
		for i, order := range qb.orders {
			// OrderClause kullandığımız için direction her zaman güvenli (ASC veya DESC)
			wrappedOrders[i] = fmt.Sprintf("%s %s", g.Wrap(order.Column), order.Direction)
		}
		sql += " ORDER BY " + strings.Join(wrappedOrders, ", ")
	}

	// LIMIT ekle
	if qb.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	// OFFSET ekle
	if qb.offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return sql, args
}

// CompileInsert, INSERT sorgusu üretir.
//
// Parametreler:
//   - table: Hedef tablo adı
//   - data: Eklenecek veri (kolon adı -> değer mapping)
//
// Döndürür:
//   - string: Oluşturulan SQL query
//   - []interface{}: Prepared statement parametreleri
//
// Örnek:
//
//	CompileInsert("users", map[string]interface{}{
//	    "name": "John",
//	    "email": "john@example.com",
//	})
//	→ SQL: "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
//	→ Args: ["John", "john@example.com"]
func (g *MySQLGrammar) CompileInsert(table string, data map[string]interface{}) (string, []interface{}) {
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for k, v := range data {
		cols = append(cols, g.Wrap(k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		g.Wrap(table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, args
}

// CompileUpdate, UPDATE sorgusu üretir.
//
// Parametreler:
//   - table: Hedef tablo adı
//   - data: Güncellenecek veri (kolon adı -> değer mapping)
//   - wheres: WHERE koşulları
//
// Döndürür:
//   - string: Oluşturulan SQL query
//   - []interface{}: Prepared statement parametreleri
//
// Örnek:
//
//	CompileUpdate("users",
//	    map[string]interface{}{"name": "Jane"},
//	    []WhereClause{{Column: "id", Operator: "=", Value: 1}},
//	)
//	→ SQL: "UPDATE `users` SET `name` = ? WHERE `id` = ?"
//	→ Args: ["Jane", 1]
//
// GÜVENLİK UYARISI:
// WHERE clause olmadan UPDATE çalıştırmak TÜM SATIRI GÜNCELLEYEBİLİR!
func (g *MySQLGrammar) CompileUpdate(table string, data map[string]interface{}, wheres []WhereClause) (string, []interface{}) {
	sets := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	// SET clause'unu oluştur
	for k, v := range data {
		sets = append(sets, fmt.Sprintf("%s = ?", g.Wrap(k)))
		args = append(args, v)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s", g.Wrap(table), strings.Join(sets, ", "))

	// WHERE clause'ları ekle
	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			// Operatörü validate et
			g.validateOperator(w.Operator)

			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", g.Wrap(w.Column), strings.ToUpper(w.Operator))
			args = append(args, w.Value)
		}
	}
	return sql, args
}

// CompileDelete, DELETE sorgusu üretir.
//
// Parametreler:
//   - table: Hedef tablo adı
//   - wheres: WHERE koşulları
//
// Döndürür:
//   - string: Oluşturulan SQL query
//   - []interface{}: Prepared statement parametreleri
//
// Örnek:
//
//	CompileDelete("users", []WhereClause{{Column: "id", Operator: "=", Value: 1}})
//	→ SQL: "DELETE FROM `users` WHERE `id` = ?"
//	→ Args: [1]
//
// GÜVENLİK UYARISI:
// WHERE clause olmadan DELETE çalıştırmak TÜM TABLONUN SİLİNMESİNE sebep olur!
func (g *MySQLGrammar) CompileDelete(table string, wheres []WhereClause) (string, []interface{}) {
	sql := fmt.Sprintf("DELETE FROM %s", g.Wrap(table))
	var args []interface{}

	// WHERE clause'ları ekle
	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			// Operatörü validate et
			g.validateOperator(w.Operator)

			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", g.Wrap(w.Column), strings.ToUpper(w.Operator))
			args = append(args, w.Value)
		}
	}
	return sql, args
}
