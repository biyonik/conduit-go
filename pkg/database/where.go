package database

// -----------------------------------------------------------------------------
// WHERE OPERATIONS (GÜVENLİK İYİLEŞTİRMELERİ İLE)
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder için WHERE ile ilgili yardımcı metotları içerir.
// Daha karmaşık where tipleri (IN, BETWEEN, NULL vs.) burada genişletilebilir.
//
// GÜVENLİK NOTU:
// WhereClause yapısı artık types.go dosyasında tanımlı.
// Tüm değerler prepared statement ile bağlandığı için SQL injection korumalıdır.
// -----------------------------------------------------------------------------

// NOT: WhereClause artık types.go dosyasında tanımlı.
// Bu dosya gelecekte gelişmiş WHERE metodları için ayrılmıştır.

// Gelecekte eklenecek metodlar:
// - WhereIn(column string, values []interface{})
// - WhereNotIn(column string, values []interface{})
// - WhereBetween(column string, min, max interface{})
// - WhereNull(column string)
// - WhereNotNull(column string)
// - WhereRaw(sql string, bindings ...interface{}) // Dikkatli kullanılmalı!

// WhereIn örnek implementasyonu (gelecek için):
//
// func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
//     qb.wheres = append(qb.wheres, WhereClause{
//         Column:   column,
//         Operator: "IN",
//         Value:    values, // Grammar katmanında (?, ?, ?) şeklinde expand edilecek
//         Boolean:  "AND",
//     })
//     return qb
// }

// WhereNull örnek implementasyonu (gelecek için):
//
// func (qb *QueryBuilder) WhereNull(column string) *QueryBuilder {
//     qb.wheres = append(qb.wheres, WhereClause{
//         Column:   column,
//         Operator: "IS",
//         Value:    nil, // Grammar katmanında "IS NULL" olarak compile edilecek
//         Boolean:  "AND",
//     })
//     return qb
// }
