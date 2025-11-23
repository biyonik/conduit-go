// -----------------------------------------------------------------------------
// WHERE Methods Security Tests
// -----------------------------------------------------------------------------
// Bu testler, WhereIn, WhereBetween, WhereNull ve diğer WHERE metodlarının
// SQL injection'a karşı korumalı olduğunu doğrular.
//
// Test edilen metodlar:
// - WhereIn / WhereNotIn
// - WhereBetween / WhereNotBetween
// - WhereNull / WhereNotNull
// - WhereDate, WhereYear, WhereMonth, WhereDay
// -----------------------------------------------------------------------------

package database

import (
	"strings"
	"testing"
)

// TestWhereIn_BasicUsage tests basic WhereIn functionality.
func TestWhereIn_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("users").
		Select("id", "name", "email").
		WhereIn("status", []interface{}{"active", "pending", "approved"})

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// SQL should use placeholders
	expected := "SELECT `id`, `name`, `email` FROM `users` WHERE `status` IN (?, ?, ?)"
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	// Args should match
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}

// TestWhereIn_SQLInjectionPrevention tests SQL injection prevention in WhereIn.
func TestWhereIn_SQLInjectionPrevention(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	// Malicious values should be safely bound as parameters
	maliciousValues := []interface{}{
		"active",
		"'; DROP TABLE users--",
		"' OR '1'='1",
		"admin' UNION SELECT * FROM passwords--",
	}

	qb.Table("users").WhereIn("status", maliciousValues)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// SQL should NOT contain the malicious code directly
	if strings.Contains(sql, "DROP TABLE") {
		t.Error("SQL injection detected in WhereIn: DROP TABLE found in query")
	}
	if strings.Contains(sql, "UNION SELECT") {
		t.Error("SQL injection detected in WhereIn: UNION SELECT found in query")
	}

	// Should use placeholders
	if !strings.Contains(sql, "IN (?, ?, ?, ?)") {
		t.Error("WhereIn should use placeholders")
	}

	// All values should be in args
	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d", len(args))
	}
}

// TestWhereIn_MaliciousColumn tests malicious column names in WhereIn.
func TestWhereIn_MaliciousColumn(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	maliciousColumns := []string{
		"status; DROP TABLE users--",
		"status' OR '1'='1",
		"status/**/UNION/**/SELECT",
	}

	for _, col := range maliciousColumns {
		t.Run(col, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for malicious column '%s', got none", col)
				}
			}()

			qb = NewBuilder(nil, grammar)
			qb.Table("users").WhereIn(col, []interface{}{"active"})
		})
	}
}

// TestWhereBetween_BasicUsage tests basic WhereBetween functionality.
func TestWhereBetween_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("users").
		Select("id", "name", "age").
		WhereBetween("age", 18, 65)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// SQL should use BETWEEN with placeholders
	expected := "SELECT `id`, `name`, `age` FROM `users` WHERE `age` BETWEEN ? AND ?"
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	// Args should be [18, 65]
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
	if args[0] != 18 || args[1] != 65 {
		t.Errorf("Args mismatch: got %v", args)
	}
}

// TestWhereBetween_SQLInjection tests SQL injection prevention in WhereBetween.
func TestWhereBetween_SQLInjection(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	// Try to inject SQL in min/max values
	qb.Table("users").WhereBetween("age", "18; DROP TABLE users--", 65)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// SQL should NOT contain DROP TABLE
	if strings.Contains(sql, "DROP TABLE") {
		t.Error("SQL injection in WhereBetween: DROP TABLE in query")
	}

	// Should use placeholders
	if !strings.Contains(sql, "BETWEEN ? AND ?") {
		t.Error("WhereBetween should use placeholders")
	}

	// Malicious value should be safely bound
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

// TestWhereNull_BasicUsage tests basic WhereNull functionality.
func TestWhereNull_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("users").
		Select("id", "name").
		WhereNull("deleted_at")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// SQL should use IS NULL
	expected := "SELECT `id`, `name` FROM `users` WHERE `deleted_at` IS NULL"
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}

	// No args for IS NULL
	if len(args) != 0 {
		t.Errorf("Expected 0 args for WhereNull, got %d", len(args))
	}
}

// TestWhereNotNull_BasicUsage tests basic WhereNotNull functionality.
func TestWhereNotNull_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("users").
		Select("id", "name").
		WhereNotNull("email_verified_at")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// SQL should use IS NOT NULL
	expected := "SELECT `id`, `name` FROM `users` WHERE `email_verified_at` IS NOT NULL"
	if sql != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, sql)
	}
}

// TestWhereDate_BasicUsage tests basic WhereDate functionality.
func TestWhereDate_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("orders").
		Select("id", "total").
		WhereDate("created_at", "2024-01-15")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should use DATE() function
	if !strings.Contains(sql, "DATE(`created_at`)") {
		t.Error("WhereDate should use DATE() function")
	}

	// Date should be parameterized
	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}
	if args[0] != "2024-01-15" {
		t.Errorf("Expected date '2024-01-15', got %v", args[0])
	}
}

// TestWhereYear_BasicUsage tests basic WhereYear functionality.
func TestWhereYear_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("posts").
		Select("id", "title").
		WhereYear("created_at", 2024)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should use YEAR() function
	if !strings.Contains(sql, "YEAR(`created_at`)") {
		t.Error("WhereYear should use YEAR() function")
	}

	// Year should be parameterized
	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}
}

// TestWhereMonth_BasicUsage tests basic WhereMonth functionality.
func TestWhereMonth_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("sales").
		Select("id", "amount").
		WhereMonth("sale_date", 12)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should use MONTH() function
	if !strings.Contains(sql, "MONTH(`sale_date`)") {
		t.Error("WhereMonth should use MONTH() function")
	}
}

// TestWhereDay_BasicUsage tests basic WhereDay functionality.
func TestWhereDay_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("appointments").
		Select("id", "time").
		WhereDay("scheduled_at", 15)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should use DAY() function
	if !strings.Contains(sql, "DAY(`scheduled_at`)") {
		t.Error("WhereDay should use DAY() function")
	}
}

// TestCombinedWhereMethods tests combining multiple WHERE methods.
func TestCombinedWhereMethods(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("users").
		Select("id", "name", "email").
		Where("active", "=", true).
		WhereIn("role", []interface{}{"admin", "moderator"}).
		WhereBetween("age", 18, 65).
		WhereNotNull("email_verified_at").
		WhereNull("deleted_at")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should have all WHERE clauses
	if !strings.Contains(sql, "WHERE") {
		t.Error("Missing WHERE clause")
	}
	if !strings.Contains(sql, "IN (?, ?)") {
		t.Error("Missing IN clause")
	}
	if !strings.Contains(sql, "BETWEEN ? AND ?") {
		t.Error("Missing BETWEEN clause")
	}
	if !strings.Contains(sql, "IS NOT NULL") {
		t.Error("Missing IS NOT NULL clause")
	}
	if !strings.Contains(sql, "IS NULL") {
		t.Error("Missing IS NULL clause")
	}

	// Args count: 1 (active) + 2 (role IN) + 2 (age BETWEEN) = 5
	expectedArgCount := 5
	if len(args) != expectedArgCount {
		t.Errorf("Expected %d args, got %d", expectedArgCount, len(args))
	}
}

// TestWhereNotIn_BasicUsage tests WhereNotIn.
func TestWhereNotIn_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("users").
		WhereNotIn("role", []interface{}{"banned", "suspended"})

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should use NOT IN
	if !strings.Contains(sql, "NOT IN (?, ?)") {
		t.Error("WhereNotIn should use NOT IN with placeholders")
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

// TestWhereNotBetween_BasicUsage tests WhereNotBetween.
func TestWhereNotBetween_BasicUsage(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	qb.Table("scores").
		WhereNotBetween("score", 0, 50)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL: %v", err)
	}

	// Should use NOT BETWEEN
	if !strings.Contains(sql, "NOT BETWEEN ? AND ?") {
		t.Error("WhereNotBetween should use NOT BETWEEN with placeholders")
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

// TestWhereMethods_EmptyArrays tests edge cases with empty arrays.
func TestWhereMethods_EmptyArrays(t *testing.T) {
	grammar := NewMySQLGrammar()
	qb := NewBuilder(nil, grammar)

	// Empty WhereIn - should still work
	qb.Table("users").WhereIn("status", []interface{}{})

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("Failed to compile SQL with empty WhereIn: %v", err)
	}

	// Should have IN () clause
	if !strings.Contains(sql, "IN ()") {
		t.Logf("SQL with empty IN: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args for empty WhereIn, got %d", len(args))
	}
}

// BenchmarkWhereIn benchmarks WhereIn performance.
func BenchmarkWhereIn(b *testing.B) {
	grammar := NewMySQLGrammar()

	values := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		values[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewBuilder(nil, grammar)
		qb.Table("users").WhereIn("id", values)
		qb.ToSQL()
	}
}

// BenchmarkWhereBetween benchmarks WhereBetween performance.
func BenchmarkWhereBetween(b *testing.B) {
	grammar := NewMySQLGrammar()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewBuilder(nil, grammar)
		qb.Table("users").WhereBetween("age", 18, 65)
		qb.ToSQL()
	}
}
