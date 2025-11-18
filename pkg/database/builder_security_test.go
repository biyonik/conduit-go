package database

import (
	"testing"
)

// -----------------------------------------------------------------------------
// SQL INJECTION GÜVENLİK TESTLERİ
// -----------------------------------------------------------------------------
// Bu testler, SQL injection saldırılarına karşı korumanın çalıştığını doğrular.
// Her test case bir exploit senaryosunu simüle eder.
// -----------------------------------------------------------------------------

// TestSQLInjection_OrderBy_MaliciousColumn tests SQL injection prevention in OrderBy
func TestSQLInjection_OrderBy_MaliciousColumn(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	maliciousInputs := []struct {
		name   string
		column string
	}{
		{
			name:   "DROP TABLE attack",
			column: "id; DROP TABLE users--",
		},
		{
			name:   "OR injection",
			column: "id' OR '1'='1",
		},
		{
			name:   "UNION attack",
			column: "id UNION SELECT * FROM passwords--",
		},
		{
			name:   "Comment injection",
			column: "id--",
		},
		{
			name:   "Semicolon injection",
			column: "id; UPDATE users SET admin=1",
		},
		{
			name:   "Quote injection",
			column: "id'",
		},
		{
			name:   "Double quote injection",
			column: `id"`,
		},
		{
			name:   "Backtick injection",
			column: "id`",
		},
	}

	for _, tc := range maliciousInputs {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for malicious input '%s', but no panic occurred", tc.column)
				}
			}()

			// Bu çağrı panic atmalı
			qb.Table("users").OrderBy(tc.column, "DESC")
		})
	}
}

// TestSQLInjection_Where_MaliciousColumn tests SQL injection prevention in Where
func TestSQLInjection_Where_MaliciousColumn(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	maliciousInputs := []string{
		"id; DROP TABLE users--",
		"id' OR '1'='1",
		"id/**/OR/**/1=1",
		"id'; DELETE FROM users WHERE '1'='1",
	}

	for _, column := range maliciousInputs {
		t.Run(column, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for malicious input '%s', but no panic occurred", column)
				}
			}()

			qb.Table("users").Where(column, "=", 1)
		})
	}
}

// TestSQLInjection_Table_MaliciousName tests SQL injection prevention in Table
func TestSQLInjection_Table_MaliciousName(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	maliciousInputs := []string{
		"users; DROP TABLE sessions--",
		"users' OR '1'='1",
		"users/**/UNION/**/SELECT",
	}

	for _, table := range maliciousInputs {
		t.Run(table, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for malicious input '%s', but no panic occurred", table)
				}
			}()

			qb.Table(table)
		})
	}
}

// TestSQLInjection_Select_MaliciousColumn tests SQL injection prevention in Select
func TestSQLInjection_Select_MaliciousColumn(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	maliciousInputs := []string{
		"id; DROP TABLE users--",
		"id, (SELECT password FROM admin)",
		"*; DELETE FROM users--",
	}

	for _, column := range maliciousInputs {
		t.Run(column, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for malicious input '%s', but no panic occurred", column)
				}
			}()

			qb.Table("users").Select(column)
		})
	}
}

// TestSQLInjection_Insert_MaliciousColumn tests SQL injection prevention in Insert
func TestSQLInjection_Insert_MaliciousColumn(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	data := map[string]interface{}{
		"name; DROP TABLE users--": "test",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for malicious column name in Insert, but no panic occurred")
		}
	}()

	qb.Table("users").ExecInsert(data)
}

// TestSQLInjection_Update_MaliciousColumn tests SQL injection prevention in Update
func TestSQLInjection_Update_MaliciousColumn(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	data := map[string]interface{}{
		"id' OR '1'='1": "hacked",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for malicious column name in Update, but no panic occurred")
		}
	}()

	qb.Table("users").Where("id", "=", 1).ExecUpdate(data)
}

// TestValidIdentifiers tests that legitimate identifiers are accepted
func TestValidIdentifiers(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	validCases := []struct {
		name   string
		method func()
	}{
		{
			name: "Simple column",
			method: func() {
				qb.Table("users").OrderBy("id", "DESC")
			},
		},
		{
			name: "Underscore column",
			method: func() {
				qb.Table("users").OrderBy("user_id", "ASC")
			},
		},
		{
			name: "Table.column format",
			method: func() {
				qb.Table("users").OrderBy("users.created_at", "DESC")
			},
		},
		{
			name: "Numeric in name",
			method: func() {
				qb.Table("table123").OrderBy("column2", "ASC")
			},
		},
		{
			name: "Wildcard select",
			method: func() {
				qb.Table("users").Select("*")
			},
		},
		{
			name: "Multiple columns",
			method: func() {
				qb.Table("users").Select("id", "name", "email")
			},
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Valid identifier caused panic: %v", r)
				}
			}()

			// Reset builder
			qb = NewBuilder(nil, grammar)
			tc.method()
		})
	}
}

// TestSQLFunctions tests that SQL functions are handled correctly
func TestSQLFunctions(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	validFunctions := []string{
		"COUNT(*) as total",
		"SUM(price)",
		"MAX(id)",
		"MIN(created_at)",
		"AVG(rating)",
	}

	for _, fn := range validFunctions {
		t.Run(fn, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Valid SQL function '%s' caused panic: %v", fn, r)
				}
			}()

			qb = NewBuilder(nil, grammar)
			qb.Table("users").Select(fn)
		})
	}
}

// TestMaliciousSQLFunctions tests that malicious SQL functions are blocked
func TestMaliciousSQLFunctions(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	maliciousFunctions := []string{
		"COUNT(*); DROP TABLE users--",
		"SUM(price)--comment",
	}

	for _, fn := range maliciousFunctions {
		t.Run(fn, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Malicious SQL function '%s' should have caused panic", fn)
				}
			}()

			qb = NewBuilder(nil, grammar)
			qb.Table("users").Select(fn)
		})
	}
}

// TestEmptyIdentifiers tests that empty identifiers are rejected
func TestEmptyIdentifiers(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	testCases := []struct {
		name   string
		method func()
	}{
		{
			name: "Empty table name",
			method: func() {
				qb.Table("")
			},
		},
		{
			name: "Empty column in Where",
			method: func() {
				qb.Table("users").Where("", "=", 1)
			},
		},
		{
			name: "Empty column in OrderBy",
			method: func() {
				qb.Table("users").OrderBy("", "ASC")
			},
		},
		{
			name: "Whitespace only table",
			method: func() {
				qb.Table("   ")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for empty identifier, but no panic occurred")
				}
			}()

			qb = NewBuilder(nil, grammar)
			tc.method()
		})
	}
}

// TestMultipleDots tests that multiple dots in identifiers are rejected
func TestMultipleDots(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for identifier with multiple dots")
		}
	}()

	qb.Table("users").OrderBy("schema.table.column", "ASC")
}

// BenchmarkValidation_OrderBy benchmarks the validation overhead
func BenchmarkValidation_OrderBy(b *testing.B) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb.Table("users").OrderBy("created_at", "DESC")
	}
}

// BenchmarkValidation_Where benchmarks the validation overhead
func BenchmarkValidation_Where(b *testing.B) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb.Table("users").Where("status", "=", "active")
	}
}