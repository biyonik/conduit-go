package database

// -----------------------------------------------------------------------------
// Grammar Interface (UPDATED FOR ERROR HANDLING)
// -----------------------------------------------------------------------------
// Interface'e error handling eklendi. Tüm compile metotları artık error dönüyor.
// -----------------------------------------------------------------------------

// Grammar, SQL lehçesine özgü sorgu üretimini tanımlar.
//
// Farklı veritabanları için farklı implementasyonlar:
// - MySQLGrammar: MySQL/MariaDB için
// - PostgreSQLGrammar: PostgreSQL için (gelecekte)
// - SQLiteGrammar: SQLite için (gelecekte)
type Grammar interface {
	// Wrap, identifier'ları (kolon/tablo adları) veritabanı lehçesine göre sarmalar.
	// MySQL: backtick (`table`), PostgreSQL: çift tırnak ("table")
	//
	// Döndürür:
	//   - string: Sarmalanmış identifier
	//   - error: Geçersiz identifier varsa
	Wrap(value string) (string, error)

	// CompileSelect, SELECT sorgusu üretir.
	//
	// Döndürür:
	//   - string: SQL sorgusu
	//   - []interface{}: Prepared statement parametreleri
	//   - error: Sorgu oluşturma hatası
	CompileSelect(qb *QueryBuilder) (string, []interface{}, error)

	// CompileInsert, INSERT sorgusu üretir.
	//
	// Döndürür:
	//   - string: SQL sorgusu
	//   - []interface{}: Prepared statement parametreleri
	//   - error: Sorgu oluşturma hatası
	CompileInsert(table string, data map[string]interface{}) (string, []interface{}, error)

	// CompileUpdate, UPDATE sorgusu üretir.
	//
	// Döndürür:
	//   - string: SQL sorgusu
	//   - []interface{}: Prepared statement parametreleri
	//   - error: Sorgu oluşturma hatası
	CompileUpdate(table string, data map[string]interface{}, wheres []WhereClause) (string, []interface{}, error)

	// CompileDelete, DELETE sorgusu üretir.
	//
	// Döndürür:
	//   - string: SQL sorgusu
	//   - []interface{}: Prepared statement parametreleri
	//   - error: Sorgu oluşturma hatası
	CompileDelete(table string, wheres []WhereClause) (string, []interface{}, error)
}