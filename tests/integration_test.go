// -----------------------------------------------------------------------------
// Integration Tests - Database & ORM
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder ve ORM özelliklerinin gerçek database ile
// entegrasyon testlerini içerir.
// -----------------------------------------------------------------------------

package tests

import (
	"database/sql"
	"testing"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/internal/models"
	"github.com/biyonik/conduit-go/pkg/database"
)

// TestUser, test için basit bir User struct'ı
type TestUser struct {
	models.BaseModel
	Name   string `db:"name"`
	Email  string `db:"email"`
	Status string `db:"status"`
}

// setupTestDB, test için database bağlantısı oluşturur
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	cfg := config.Load()
	db, err := database.Connect(cfg.DB.DSN)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

// TestQueryBuilder_Select, SELECT sorgusu testlerini yapar
func TestQueryBuilder_Select(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	grammar := database.NewMySQLGrammar()
	qb := database.NewBuilder(db, grammar)

	// Test 1: Basit SELECT
	var users []TestUser
	err := qb.Table("users").
		Where("status", "=", "active").
		Get(&users)

	if err != nil {
		t.Fatalf("SELECT query failed: %v", err)
	}

	if len(users) == 0 {
		t.Log("Warning: No active users found (this is OK if DB is empty)")
	}

	for _, user := range users {
		if user.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", user.Status)
		}
	}

	// Test 2: SELECT with ORDER BY
	qb = database.NewBuilder(db, grammar)
	err = qb.Table("users").
		OrderBy("created_at", "DESC").
		Limit(5).
		Get(&users)

	if err != nil {
		t.Fatalf("SELECT with ORDER BY failed: %v", err)
	}

	// Test 3: First() metodu
	qb = database.NewBuilder(db, grammar)
	var user TestUser
	err = qb.Table("users").
		Where("email", "=", "admin@conduit-go.local").
		First(&user)

	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("First() query failed: %v", err)
	}

	if err == nil && user.Email != "admin@conduit-go.local" {
		t.Errorf("Expected email 'admin@conduit-go.local', got '%s'", user.Email)
	}
}

// TestQueryBuilder_Insert, INSERT sorgusu testlerini yapar
func TestQueryBuilder_Insert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	grammar := database.NewMySQLGrammar()
	qb := database.NewBuilder(db, grammar)

	// Test: INSERT
	timestamp := time.Now()
	result, err := qb.ExecInsert(map[string]interface{}{
		"name":       "Test User",
		"email":      "test@example.com",
		"password":   "hashed_password",
		"status":     "active",
		"created_at": timestamp,
		"updated_at": timestamp,
	})

	if err != nil {
		t.Fatalf("INSERT query failed: %v", err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get LastInsertId: %v", err)
	}

	if lastID == 0 {
		t.Error("LastInsertId should be greater than 0")
	}

	// Cleanup: Eklenen kullanıcıyı sil
	defer func() {
		qb := database.NewBuilder(db, grammar)
		_, _ = qb.Table("users").Where("id", "=", lastID).ExecDelete()
	}()

	// Verify: Kullanıcı gerçekten eklendi mi?
	var insertedUser TestUser
	qb = database.NewBuilder(db, grammar)
	err = qb.Table("users").Where("id", "=", lastID).First(&insertedUser)

	if err != nil {
		t.Fatalf("Failed to verify inserted user: %v", err)
	}

	if insertedUser.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", insertedUser.Email)
	}
}

// TestQueryBuilder_Update, UPDATE sorgusu testlerini yapar
func TestQueryBuilder_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	grammar := database.NewMySQLGrammar()

	// Önce bir test kullanıcısı ekle
	qb := database.NewBuilder(db, grammar)
	timestamp := time.Now()
	result, err := qb.ExecInsert(map[string]interface{}{
		"name":       "Update Test User",
		"email":      "updatetest@example.com",
		"password":   "password",
		"status":     "active",
		"created_at": timestamp,
		"updated_at": timestamp,
	})

	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	userID, _ := result.LastInsertId()

	// Cleanup
	defer func() {
		qb := database.NewBuilder(db, grammar)
		_, _ = qb.Table("users").Where("id", "=", userID).ExecDelete()
	}()

	// Test: UPDATE
	qb = database.NewBuilder(db, grammar)
	updateResult, err := qb.Table("users").
		Where("id", "=", userID).
		ExecUpdate(map[string]interface{}{
			"name":       "Updated Name",
			"updated_at": time.Now(),
		})

	if err != nil {
		t.Fatalf("UPDATE query failed: %v", err)
	}

	affected, _ := updateResult.RowsAffected()
	if affected != 1 {
		t.Errorf("Expected 1 affected row, got %d", affected)
	}

	// Verify: Güncelleme başarılı mı?
	var updatedUser TestUser
	qb = database.NewBuilder(db, grammar)
	err = qb.Table("users").Where("id", "=", userID).First(&updatedUser)

	if err != nil {
		t.Fatalf("Failed to verify updated user: %v", err)
	}

	if updatedUser.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updatedUser.Name)
	}
}

// TestQueryBuilder_Delete, DELETE sorgusu testlerini yapar
func TestQueryBuilder_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	grammar := database.NewMySQLGrammar()

	// Önce bir test kullanıcısı ekle
	qb := database.NewBuilder(db, grammar)
	timestamp := time.Now()
	result, err := qb.ExecInsert(map[string]interface{}{
		"name":       "Delete Test User",
		"email":      "deletetest@example.com",
		"password":   "password",
		"status":     "active",
		"created_at": timestamp,
		"updated_at": timestamp,
	})

	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	userID, _ := result.LastInsertId()

	// Test: DELETE
	qb = database.NewBuilder(db, grammar)
	deleteResult, err := qb.Table("users").
		Where("id", "=", userID).
		ExecDelete()

	if err != nil {
		t.Fatalf("DELETE query failed: %v", err)
	}

	affected, _ := deleteResult.RowsAffected()
	if affected != 1 {
		t.Errorf("Expected 1 affected row, got %d", affected)
	}

	// Verify: Kullanıcı gerçekten silindi mi?
	var deletedUser TestUser
	qb = database.NewBuilder(db, grammar)
	err = qb.Table("users").Where("id", "=", userID).First(&deletedUser)

	if err != sql.ErrNoRows {
		t.Error("User should be deleted (sql.ErrNoRows expected)")
	}
}

// TestTransaction, transaction testlerini yapar
func TestTransaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	grammar := database.NewMySQLGrammar()

	// Test 1: Başarılı transaction (commit)
	tx, err := database.BeginTransaction(db, grammar)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	timestamp := time.Now()
	result, err := tx.NewBuilder().ExecInsert(map[string]interface{}{
		"name":       "Transaction Test User",
		"email":      "txtest@example.com",
		"password":   "password",
		"status":     "active",
		"created_at": timestamp,
		"updated_at": timestamp,
	})

	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	userID, _ := result.LastInsertId()

	// Commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify: Kullanıcı commit sonrası mevcut mu?
	var user TestUser
	qb := database.NewBuilder(db, grammar)
	err = qb.Table("users").Where("id", "=", userID).First(&user)

	if err != nil {
		t.Error("User should exist after commit")
	}

	// Cleanup
	defer func() {
		qb := database.NewBuilder(db, grammar)
		_, _ = qb.Table("users").Where("id", "=", userID).ExecDelete()
	}()

	// Test 2: Başarısız transaction (rollback)
	tx, err = database.BeginTransaction(db, grammar)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	result, err = tx.NewBuilder().ExecInsert(map[string]interface{}{
		"name":       "Rollback Test User",
		"email":      "rollbacktest@example.com",
		"password":   "password",
		"status":     "active",
		"created_at": timestamp,
		"updated_at": timestamp,
	})

	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	rollbackUserID, _ := result.LastInsertId()

	// Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify: Kullanıcı rollback sonrası mevcut olmamalı
	qb = database.NewBuilder(db, grammar)
	err = qb.Table("users").Where("id", "=", rollbackUserID).First(&user)

	if err != sql.ErrNoRows {
		t.Error("User should not exist after rollback")
	}
}
