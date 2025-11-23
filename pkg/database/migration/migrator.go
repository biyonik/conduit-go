// -----------------------------------------------------------------------------
// Database Migration System - Laravel-Inspired
// -----------------------------------------------------------------------------
// Bu package, Laravel'in migration system'ine benzer şekilde veritabanı
// şema değişikliklerini yönetir.
//
// Özellikler:
// - Schema builder (CreateTable, AlterTable, DropTable)
// - Column types (String, Integer, Boolean, Timestamps, etc.)
// - Indexes (primary, unique, index, foreign keys)
// - Migration tracking (migrations table)
// - Rollback support
//
// Kullanım:
//
//	func (m *CreateUsersTable) Up(migrator *Migrator) error {
//	    return migrator.CreateTable("users", func(t *Blueprint) {
//	        t.ID()
//	        t.String("name", 255)
//	        t.String("email", 255).Unique()
//	        t.String("password", 255)
//	        t.Timestamps()
//	    })
//	}
//
//	func (m *CreateUsersTable) Down(migrator *Migrator) error {
//	    return migrator.DropTable("users")
//	}
// -----------------------------------------------------------------------------

package migration

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Migrator manages database migrations.
type Migrator struct {
	db      *sql.DB
	grammar Grammar // SQL dialect (MySQL, PostgreSQL, etc.)
}

// Grammar defines SQL generation interface for different databases.
type Grammar interface {
	CompileCreateTable(table string, columns []Column, indexes []Index) string
	CompileDropTable(table string) string
	CompileAddColumn(table string, column Column) string
	CompileDropColumn(table string, columnName string) string
	CompileAddIndex(table string, index Index) string
	CompileDropIndex(table string, indexName string) string
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(db *sql.DB, grammar Grammar) *Migrator {
	return &Migrator{
		db:      db,
		grammar: grammar,
	}
}

// CreateTable creates a new table.
func (m *Migrator) CreateTable(tableName string, callback func(*Blueprint)) error {
	blueprint := NewBlueprint(tableName)
	callback(blueprint)

	sql := m.grammar.CompileCreateTable(
		blueprint.table,
		blueprint.columns,
		blueprint.indexes,
	)

	_, err := m.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	fmt.Printf("✅ Created table: %s\n", tableName)
	return nil
}

// DropTable drops a table.
func (m *Migrator) DropTable(tableName string) error {
	sql := m.grammar.CompileDropTable(tableName)

	_, err := m.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	fmt.Printf("✅ Dropped table: %s\n", tableName)
	return nil
}

// AlterTable modifies an existing table.
func (m *Migrator) AlterTable(tableName string, callback func(*Blueprint)) error {
	blueprint := NewBlueprint(tableName)
	callback(blueprint)

	// Execute column additions
	for _, column := range blueprint.columns {
		sql := m.grammar.CompileAddColumn(tableName, column)
		if _, err := m.db.Exec(sql); err != nil {
			return fmt.Errorf("failed to add column %s: %w", column.Name, err)
		}
	}

	// Execute index additions
	for _, index := range blueprint.indexes {
		sql := m.grammar.CompileAddIndex(tableName, index)
		if _, err := m.db.Exec(sql); err != nil {
			return fmt.Errorf("failed to add index: %w", err)
		}
	}

	fmt.Printf("✅ Altered table: %s\n", tableName)
	return nil
}

// HasTable checks if a table exists.
func (m *Migrator) HasTable(tableName string) (bool, error) {
	// MySQL specific query
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"

	var count int
	err := m.db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateMigrationsTable creates the migrations tracking table.
func (m *Migrator) CreateMigrationsTable() error {
	exists, err := m.HasTable("migrations")
	if err != nil {
		return err
	}

	if exists {
		return nil // Already exists
	}

	sql := `
		CREATE TABLE migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`

	_, err = m.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	fmt.Println("✅ Created migrations table")
	return nil
}

// RecordMigration records a migration as run.
func (m *Migrator) RecordMigration(name string, batch int) error {
	sql := "INSERT INTO migrations (migration, batch) VALUES (?, ?)"
	_, err := m.db.Exec(sql, name, batch)
	return err
}

// DeleteMigration removes a migration record.
func (m *Migrator) DeleteMigration(name string) error {
	sql := "DELETE FROM migrations WHERE migration = ?"
	_, err := m.db.Exec(sql, name)
	return err
}

// GetRanMigrations returns all migrations that have been run.
func (m *Migrator) GetRanMigrations() ([]string, error) {
	sql := "SELECT migration FROM migrations ORDER BY id ASC"

	rows, err := m.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var migration string
		if err := rows.Scan(&migration); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// GetLastBatch returns the last batch number.
func (m *Migrator) GetLastBatch() (int, error) {
	sql := "SELECT MAX(batch) FROM migrations"

	var batch sql.NullInt64
	err := m.db.QueryRow(sql).Scan(&batch)
	if err != nil {
		return 0, err
	}

	if batch.Valid {
		return int(batch.Int64), nil
	}

	return 0, nil
}

// -----------------------------------------------------------------------------
// Blueprint - Table Schema Builder
// -----------------------------------------------------------------------------

// Blueprint defines the structure of a table.
type Blueprint struct {
	table   string
	columns []Column
	indexes []Index
}

// NewBlueprint creates a new Blueprint instance.
func NewBlueprint(tableName string) *Blueprint {
	return &Blueprint{
		table:   tableName,
		columns: make([]Column, 0),
		indexes: make([]Index, 0),
	}
}

// ID adds an auto-incrementing primary key column.
func (b *Blueprint) ID() *Column {
	return b.addColumn(Column{
		Name:          "id",
		Type:          ColumnTypeUnsignedBigInt,
		AutoIncrement: true,
		Primary:       true,
	})
}

// String adds a VARCHAR column.
func (b *Blueprint) String(name string, length int) *Column {
	return b.addColumn(Column{
		Name:   name,
		Type:   ColumnTypeString,
		Length: length,
	})
}

// Text adds a TEXT column.
func (b *Blueprint) Text(name string) *Column {
	return b.addColumn(Column{
		Name: name,
		Type: ColumnTypeText,
	})
}

// Integer adds an INT column.
func (b *Blueprint) Integer(name string) *Column {
	return b.addColumn(Column{
		Name: name,
		Type: ColumnTypeInteger,
	})
}

// BigInteger adds a BIGINT column.
func (b *Blueprint) BigInteger(name string) *Column {
	return b.addColumn(Column{
		Name: name,
		Type: ColumnTypeBigInt,
	})
}

// Boolean adds a TINYINT(1) column.
func (b *Blueprint) Boolean(name string) *Column {
	return b.addColumn(Column{
		Name: name,
		Type: ColumnTypeBoolean,
	})
}

// Timestamp adds a TIMESTAMP column.
func (b *Blueprint) Timestamp(name string) *Column {
	return b.addColumn(Column{
		Name: name,
		Type: ColumnTypeTimestamp,
	})
}

// Timestamps adds created_at and updated_at columns.
func (b *Blueprint) Timestamps() {
	b.Timestamp("created_at").Nullable()
	b.Timestamp("updated_at").Nullable()
}

// SoftDeletes adds a deleted_at column for soft delete support.
func (b *Blueprint) SoftDeletes() {
	b.Timestamp("deleted_at").Nullable()
}

// addColumn adds a column to the blueprint.
func (b *Blueprint) addColumn(column Column) *Column {
	b.columns = append(b.columns, column)
	return &b.columns[len(b.columns)-1]
}

// Unique adds a unique index.
func (b *Blueprint) Unique(columns ...string) {
	indexName := fmt.Sprintf("%s_%s_unique", b.table, strings.Join(columns, "_"))
	b.indexes = append(b.indexes, Index{
		Name:    indexName,
		Columns: columns,
		Type:    IndexTypeUnique,
	})
}

// Index adds a regular index.
func (b *Blueprint) Index(columns ...string) {
	indexName := fmt.Sprintf("%s_%s_index", b.table, strings.Join(columns, "_"))
	b.indexes = append(b.indexes, Index{
		Name:    indexName,
		Columns: columns,
		Type:    IndexTypeIndex,
	})
}

// Foreign adds a foreign key constraint.
func (b *Blueprint) Foreign(column string) *ForeignKey {
	return &ForeignKey{
		blueprint: b,
		column:    column,
	}
}

// -----------------------------------------------------------------------------
// Column Definition
// -----------------------------------------------------------------------------

// ColumnType represents a database column type.
type ColumnType string

const (
	ColumnTypeString         ColumnType = "VARCHAR"
	ColumnTypeText           ColumnType = "TEXT"
	ColumnTypeInteger        ColumnType = "INT"
	ColumnTypeBigInt         ColumnType = "BIGINT"
	ColumnTypeUnsignedBigInt ColumnType = "BIGINT UNSIGNED"
	ColumnTypeBoolean        ColumnType = "TINYINT(1)"
	ColumnTypeTimestamp      ColumnType = "TIMESTAMP"
	ColumnTypeDateTime       ColumnType = "DATETIME"
	ColumnTypeDate           ColumnType = "DATE"
	ColumnTypeDecimal        ColumnType = "DECIMAL"
)

// Column represents a table column.
type Column struct {
	Name          string
	Type          ColumnType
	Length        int
	Nullable      bool
	Default       interface{}
	Unsigned      bool
	AutoIncrement bool
	Primary       bool
	Unique        bool
}

// Nullable marks the column as nullable.
func (c *Column) Nullable() *Column {
	c.Nullable = true
	return c
}

// Default sets a default value.
func (c *Column) Default(value interface{}) *Column {
	c.Default = value
	return c
}

// Unsigned marks the column as unsigned (for numeric types).
func (c *Column) Unsigned() *Column {
	c.Unsigned = true
	return c
}

// Unique adds a unique constraint.
func (c *Column) Unique() *Column {
	c.Unique = true
	return c
}

// -----------------------------------------------------------------------------
// Index Definition
// -----------------------------------------------------------------------------

// IndexType represents the type of index.
type IndexType string

const (
	IndexTypeIndex   IndexType = "INDEX"
	IndexTypeUnique  IndexType = "UNIQUE"
	IndexTypePrimary IndexType = "PRIMARY KEY"
	IndexTypeForeign IndexType = "FOREIGN KEY"
)

// Index represents a table index.
type Index struct {
	Name    string
	Columns []string
	Type    IndexType
}

// ForeignKey represents a foreign key constraint.
type ForeignKey struct {
	blueprint       *Blueprint
	column          string
	referencedTable string
	referencedColumn string
	onDelete        string
	onUpdate        string
}

// References sets the referenced table and column.
func (fk *ForeignKey) References(column string) *ForeignKey {
	fk.referencedColumn = column
	return fk
}

// On sets the referenced table.
func (fk *ForeignKey) On(table string) *ForeignKey {
	fk.referencedTable = table
	return fk
}

// OnDelete sets the ON DELETE action.
func (fk *ForeignKey) OnDelete(action string) *ForeignKey {
	fk.onDelete = action
	return fk
}

// OnUpdate sets the ON UPDATE action.
func (fk *ForeignKey) OnUpdate(action string) *ForeignKey {
	fk.onUpdate = action
	return fk
}

// Cascade sets both ON DELETE and ON UPDATE to CASCADE.
func (fk *ForeignKey) Cascade() *ForeignKey {
	fk.onDelete = "CASCADE"
	fk.onUpdate = "CASCADE"
	return fk
}
