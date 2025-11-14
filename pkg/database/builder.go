package database

import (
	"database/sql"
	"fmt"
)

// -----------------------------------------------------------------------------
// QUERY BUILDER — TEMEL
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder'ın ana gövdesini içerir. Builder; tablo, kolonlar,
// where'lar, order, limit, offset gibi state bilgilerini tutar. Ayrıca gelişmiş
// CRUD metodları (Insert, Update, Delete, Get, First, Exec) bu yapı üzerinden
// sağlanır.
// -----------------------------------------------------------------------------

type QueryBuilder struct {
	executor QueryExecutor
	grammar  Grammar
	table    string
	columns  []string
	wheres   []WhereClause
	orders   []string
	limit    int
	offset   int
}

// NewBuilder, veritabanı bağlantısını alarak yeni QueryBuilder üretir.
func NewBuilder(executor QueryExecutor, grammar Grammar) *QueryBuilder {
	return &QueryBuilder{
		executor: executor,
		grammar:  grammar,
		columns:  []string{"*"},
		limit:    0,
		offset:   0,
	}
}

// Table belirler
func (qb *QueryBuilder) Table(tableName string) *QueryBuilder {
	qb.table = tableName
	return qb
}

// Select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Where
func (qb *QueryBuilder) Where(column string, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereClause{Column: column, Operator: operator, Value: value, Boolean: "AND"})
	return qb
}

// OrWhere
func (qb *QueryBuilder) OrWhere(column string, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereClause{Column: column, Operator: operator, Value: value, Boolean: "OR"})
	return qb
}

// OrderBy
func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder {
	qb.orders = append(qb.orders, fmt.Sprintf("`%s` %s", column, direction))
	return qb
}

// Limit
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// ToSQL delegasyonu — selectGrammar kullanır.
func (qb *QueryBuilder) ToSQL() (string, []interface{}) {
	return qb.grammar.CompileSelect(qb)
}

// Exec: yazma sorgularını (INSERT/UPDATE/DELETE) çalıştırmak için kullanılır.
// Bu metodlar grammar fonksiyonlarını çağırır ve db.Exec ile yürütür.
func (qb *QueryBuilder) ExecInsert(data map[string]interface{}) (sql.Result, error) {
	sqlStr, args := qb.grammar.CompileInsert(qb.table, data)
	return qb.executor.Exec(sqlStr, args...)
}

func (qb *QueryBuilder) ExecUpdate(data map[string]interface{}) (sql.Result, error) {
	sqlStr, args := qb.grammar.CompileUpdate(qb.table, data, qb.wheres)
	return qb.executor.Exec(sqlStr, args...)
}

func (qb *QueryBuilder) ExecDelete() (sql.Result, error) {
	sqlStr, args := qb.grammar.CompileDelete(qb.table, qb.wheres)
	return qb.executor.Exec(sqlStr, args...)
}
