package database

import (
	"fmt"
	"strings"
)

type MySQLGrammar struct{}

func NewMySQLGrammar() *MySQLGrammar {
	return &MySQLGrammar{}
}

func (g *MySQLGrammar) Wrap(value string) string {
	if value == "*" {
		return value
	}
	// 'table.column' gibi durumları handle et
	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		wrappedParts := make([]string, len(parts))
		for i, part := range parts {
			wrappedParts[i] = fmt.Sprintf("`%s`", part)
		}
		return strings.Join(wrappedParts, ".")
	}
	return fmt.Sprintf("`%s`", value)
}

// CompileSelect, MySQL lehçesinde SELECT sorgusu derler.
func (g *MySQLGrammar) CompileSelect(qb *QueryBuilder) (string, []interface{}) {
	wrappedCols := make([]string, len(qb.columns))
	for i, col := range qb.columns {
		wrappedCols[i] = g.Wrap(col)
	}

	sql := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(wrappedCols, ", "),
		g.Wrap(qb.table),
	)
	var args []interface{}

	if len(qb.wheres) > 0 {
		sql += " WHERE "
		for i, w := range qb.wheres {
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", g.Wrap(w.Column), w.Operator)
			args = append(args, w.Value)
		}
	}

	if len(qb.orders) > 0 {
		wrappedOrders := make([]string, len(qb.orders))
		for i, order := range qb.orders {
			parts := strings.Fields(order) // "name DESC" -> ["name", "DESC"]
			if len(parts) == 2 {
				wrappedOrders[i] = fmt.Sprintf("%s %s", g.Wrap(parts[0]), strings.ToUpper(parts[1]))
			} else {
				wrappedOrders[i] = g.Wrap(order)
			}
		}
		sql += " ORDER BY " + strings.Join(wrappedOrders, ", ")
	}
	if qb.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	if qb.offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", qb.offset)
	}
	return sql, args
}

// CompileInsert, MySQL lehçesinde INSERT sorgusu derler.
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

// CompileUpdate, MySQL lehçesinde UPDATE sorgusu derler.
func (g *MySQLGrammar) CompileUpdate(table string, data map[string]interface{}, wheres []WhereClause) (string, []interface{}) {
	sets := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for k, v := range data {
		sets = append(sets, fmt.Sprintf("%s = ?", g.Wrap(k)))
		args = append(args, v)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s", g.Wrap(table), strings.Join(sets, ", "))

	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", g.Wrap(w.Column), w.Operator)
			args = append(args, w.Value)
		}
	}
	return sql, args
}

// CompileDelete, MySQL lehçesinde DELETE sorgusu derler.
func (g *MySQLGrammar) CompileDelete(table string, wheres []WhereClause) (string, []interface{}) {
	sql := fmt.Sprintf("DELETE FROM %s", g.Wrap(table))
	var args []interface{}

	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", g.Wrap(w.Column), w.Operator)
			args = append(args, w.Value)
		}
	}
	return sql, args
}
