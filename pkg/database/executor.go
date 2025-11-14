package database

import "database/sql"

/*
*
//QueryExecutor, Go'nun 'database/sql' paketindeki
// hem *sql.DB (havuz) hem de *sql.Tx (transaction) tarafından
// örtük olarak uygulanan metodları tanımlayan bir arayüzdür.
//
// QueryBuilder'ımız *sql.DB'ye kilitlenmek yerine bu arayüze
// kilitlenecek. Bu, onun hem normal sorgularda hem de
// transaction'lar içinde çalışabilmesini sağlar.
*/
type QueryExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
