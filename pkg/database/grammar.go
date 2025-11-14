package database

import (
	"fmt"
	"strings"
)

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

// selectGrammar, QueryBuilder içindeki state'i alarak SELECT SQL metnini
// oluşturur. Bağlamda kullanılan placeholder: "?" ve arg listesi geri döner.
func selectGrammar(qb *QueryBuilder) (string, []interface{}) {
	sql := fmt.Sprintf("SELECT %s FROM `%s`", strings.Join(qb.columns, ", "), qb.table)
	var args []interface{}

	// WHERE
	if len(qb.wheres) > 0 {
		sql += " WHERE "
		for i, w := range qb.wheres {
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("`%s` %s ?", w.Column, w.Operator)
			args = append(args, w.Value)
		}
	}

	// ORDER BY
	if len(qb.orders) > 0 {
		sql += " ORDER BY " + strings.Join(qb.orders, ", ")
	}

	// LIMIT / OFFSET
	if qb.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	if qb.offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return sql, args
}

// insertGrammar: basit map[string]interface{} kullanarak INSERT sorgusu
// üretir. Dönen args sıra ile placeholder ile eşleşir.
func insertGrammar(table string, data map[string]interface{}) (string, []interface{}) {
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for k, v := range data {
		cols = append(cols, fmt.Sprintf("`%s`", k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	return sql, args
}

// updateGrammar: data map'ine göre UPDATE sorgusu oluşturur. WhereBindings
// ayrıca arg listesine eklenir — bu fonksiyon sadece SET kısmını üretir; where
// bağlamı dışarıdan eklenir.
func updateGrammar(table string, data map[string]interface{}, wheres []WhereClause) (string, []interface{}) {
	sets := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for k, v := range data {
		sets = append(sets, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}

	sql := fmt.Sprintf("UPDATE `%s` SET %s", table, strings.Join(sets, ", "))

	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("`%s` %s ?", w.Column, w.Operator)
			args = append(args, w.Value)
		}
	}

	return sql, args
}

// deleteGrammar: DELETE sorgusu üretir (basit versiyon).
func deleteGrammar(table string, wheres []WhereClause) (string, []interface{}) {
	sql := fmt.Sprintf("DELETE FROM `%s`", table)
	var args []interface{}

	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("`%s` %s ?", w.Column, w.Operator)
			args = append(args, w.Value)
		}
	}

	return sql, args
}
