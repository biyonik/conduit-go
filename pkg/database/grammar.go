package database

// -----------------------------------------------------------------------------
// GRAMMAR — SQL ÜRETİM KATMANI
// -----------------------------------------------------------------------------
// Grammar katmanı, QueryBuilder'ın derlediği internal state'i alır ve
// farklı SQL cümleleri (SELECT, INSERT, UPDATE, DELETE) üretir. Laravel
// mimarisindeki Grammar/Compiler katmanının sade bir Go uyarlamasıdır.
//
// Neden ayrı bir katman?
// - Ayrık sorumluluk (SRP): Builder sadece state yönetir; SQL üretimi Grammar'a
// bırakılır.
// - İleride farklı SQL dialektleri (Postgres, SQLite) eklemek kolaylaşır.
// -----------------------------------------------------------------------------

type Grammar interface {
	CompileSelect(*QueryBuilder) (string, []interface{})
	CompileInsert(string, map[string]interface{}) (string, []interface{})
	CompileUpdate(string, map[string]interface{}, []WhereClause) (string, []interface{})
	CompileDelete(string, []WhereClause) (string, []interface{})
	Wrap(string) string // Sütun ve tablo adlarını sarmalar
}
