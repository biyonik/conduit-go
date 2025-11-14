package database

// -----------------------------------------------------------------------------
// WHERE OPERATIONS
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder için WHERE ile ilgili yardımcı metotları içerir.
// Daha karmaşık where tipleri (IN, BETWEEN, NULL vs.) burada genişletilebilir.
// -----------------------------------------------------------------------------

// WhereClause — WHERE koşulunu temsil eden yapı.
// Column: kolon adı
// Operator: =, <, >, LIKE, vb.
// Value: placeholder değeri
// Boolean: AND / OR
type WhereClause struct {
	Column   string
	Operator string
	Value    interface{}
	Boolean  string
}

// Where ekleme fonksiyonu builder.go içinde implement edilmiştir. Burada
// daha karmaşık where tipleri eklenebilir (WhereIn, WhereBetween, vb.).
