package database

import (
	"fmt"
	"regexp"
	"strings"
)

// -----------------------------------------------------------------------------
// MySQL Grammar (PANIC RISK FIXED)
// -----------------------------------------------------------------------------
// FIXED ISSUES:
// ✅ Wrap() panic yerine error dönüyor
// ✅ HTTP request ortasında crash riski ortadan kaldırıldı
// ✅ Graceful error handling
// -----------------------------------------------------------------------------

type MySQLGrammar struct{}

func NewMySQLGrammar() *MySQLGrammar {
	return &MySQLGrammar{}
}

var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)

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
// PANIC FIX: Artık error dönüyor, panic atmıyor.
func (g *MySQLGrammar) Wrap(value string) (string, error) {
	// Wildcard için özel durum
	if value == "*" {
		return value, nil
	}

	// Tablo.kolon formatını handle et
	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		wrappedParts := make([]string, len(parts))
		for i, part := range parts {
			// Her parçayı validate et
			if !validIdentifierPattern.MatchString(part) {
				return "", fmt.Errorf("invalid SQL identifier: %s (contains unsafe characters)", part)
			}
			wrappedParts[i] = fmt.Sprintf("`%s`", part)
		}
		return strings.Join(wrappedParts, "."), nil
	}

	// Tek identifier'ı validate et
	if !validIdentifierPattern.MatchString(value) {
		return "", fmt.Errorf("invalid SQL identifier: %s (contains unsafe characters)", value)
	}

	return fmt.Sprintf("`%s`", value), nil
}

// wrapOrPanic, eski API compat için - DEPRECATED
// Yeni kod bu fonksiyonu kullanmamalı, direkt Wrap() kullanmalı
func (g *MySQLGrammar) wrapOrPanic(value string) string {
	result, err := g.Wrap(value)
	if err != nil {
		// Son çare: panic at (eski davranış)
		panic(err)
	}
	return result
}

// validateOperator, verilen operatörün whitelist'te olup olmadığını kontrol eder.
// PANIC FIX: Artık error dönüyor
func (g *MySQLGrammar) validateOperator(operator string) error {
	op := strings.ToUpper(strings.TrimSpace(operator))
	if !allowedOperators[op] {
		return fmt.Errorf("invalid SQL operator: %s (not in whitelist)", operator)
	}
	return nil
}

// CompileSelect, QueryBuilder'dan SELECT sorgusu üretir.
func (g *MySQLGrammar) CompileSelect(qb *QueryBuilder) (string, []interface{}, error) {
	// Kolonları wrap et
	wrappedCols := make([]string, len(qb.columns))
	for i, col := range qb.columns {
		wrapped, err := g.Wrap(col)
		if err != nil {
			return "", nil, fmt.Errorf("column wrap error: %w", err)
		}
		wrappedCols[i] = wrapped
	}

	// Tablo adını wrap et
	wrappedTable, err := g.Wrap(qb.table)
	if err != nil {
		return "", nil, fmt.Errorf("table wrap error: %w", err)
	}

	// SELECT ... FROM ... kısmını oluştur
	sql := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(wrappedCols, ", "),
		wrappedTable,
	)

	var args []interface{}

	// WHERE clause'ları ekle
	if len(qb.wheres) > 0 {
		sql += " WHERE "
		for i, w := range qb.wheres {
			// Operatörü validate et
			if err := g.validateOperator(w.Operator); err != nil {
				return "", nil, fmt.Errorf("where clause error: %w", err)
			}

			// Kolon adını wrap et
			wrappedCol, err := g.Wrap(w.Column)
			if err != nil {
				return "", nil, fmt.Errorf("where column wrap error: %w", err)
			}

			// AND/OR ekle
			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}

			sql += fmt.Sprintf("%s %s ?", wrappedCol, strings.ToUpper(w.Operator))
			args = append(args, w.Value)
		}
	}

	// ORDER BY clause'ları ekle
	if len(qb.orders) > 0 {
		wrappedOrders := make([]string, len(qb.orders))
		for i, order := range qb.orders {
			wrappedCol, err := g.Wrap(order.Column)
			if err != nil {
				return "", nil, fmt.Errorf("order column wrap error: %w", err)
			}
			wrappedOrders[i] = fmt.Sprintf("%s %s", wrappedCol, order.Direction)
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

	return sql, args, nil
}

// CompileInsert, INSERT sorgusu üretir.
func (g *MySQLGrammar) CompileInsert(table string, data map[string]interface{}) (string, []interface{}, error) {
	// Tablo adını wrap et
	wrappedTable, err := g.Wrap(table)
	if err != nil {
		return "", nil, fmt.Errorf("table wrap error: %w", err)
	}

	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for k, v := range data {
		wrappedCol, err := g.Wrap(k)
		if err != nil {
			return "", nil, fmt.Errorf("column wrap error: %w", err)
		}
		cols = append(cols, wrappedCol)
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		wrappedTable,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	return sql, args, nil
}

// CompileUpdate, UPDATE sorgusu üretir.
func (g *MySQLGrammar) CompileUpdate(table string, data map[string]interface{}, wheres []WhereClause) (string, []interface{}, error) {
	// Tablo adını wrap et
	wrappedTable, err := g.Wrap(table)
	if err != nil {
		return "", nil, fmt.Errorf("table wrap error: %w", err)
	}

	sets := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	// SET clause'unu oluştur
	for k, v := range data {
		wrappedCol, err := g.Wrap(k)
		if err != nil {
			return "", nil, fmt.Errorf("column wrap error: %w", err)
		}
		sets = append(sets, fmt.Sprintf("%s = ?", wrappedCol))
		args = append(args, v)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s", wrappedTable, strings.Join(sets, ", "))

	// WHERE clause'ları ekle
	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			// Operatörü validate et
			if err := g.validateOperator(w.Operator); err != nil {
				return "", nil, fmt.Errorf("where operator error: %w", err)
			}

			// Kolon adını wrap et
			wrappedCol, err := g.Wrap(w.Column)
			if err != nil {
				return "", nil, fmt.Errorf("where column wrap error: %w", err)
			}

			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", wrappedCol, strings.ToUpper(w.Operator))
			args = append(args, w.Value)
		}
	}

	return sql, args, nil
}

// CompileDelete, DELETE sorgusu üretir.
func (g *MySQLGrammar) CompileDelete(table string, wheres []WhereClause) (string, []interface{}, error) {
	// Tablo adını wrap et
	wrappedTable, err := g.Wrap(table)
	if err != nil {
		return "", nil, fmt.Errorf("table wrap error: %w", err)
	}

	sql := fmt.Sprintf("DELETE FROM %s", wrappedTable)
	var args []interface{}

	// WHERE clause'ları ekle
	if len(wheres) > 0 {
		sql += " WHERE "
		for i, w := range wheres {
			// Operatörü validate et
			if err := g.validateOperator(w.Operator); err != nil {
				return "", nil, fmt.Errorf("where operator error: %w", err)
			}

			// Kolon adını wrap et
			wrappedCol, err := g.Wrap(w.Column)
			if err != nil {
				return "", nil, fmt.Errorf("where column wrap error: %w", err)
			}

			if i > 0 {
				sql += fmt.Sprintf(" %s ", w.Boolean)
			}
			sql += fmt.Sprintf("%s %s ?", wrappedCol, strings.ToUpper(w.Operator))
			args = append(args, w.Value)
		}
	}

	return sql, args, nil
}

// -----------------------------------------------------------------------------
// ADDITIONAL HELPER METHODS
// -----------------------------------------------------------------------------

// WrapMultiple, birden fazla identifier'ı wrap eder.
func (g *MySQLGrammar) WrapMultiple(values []string) ([]string, error) {
	wrapped := make([]string, len(values))
	for i, value := range values {
		w, err := g.Wrap(value)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap '%s': %w", value, err)
		}
		wrapped[i] = w
	}
	return wrapped, nil
}