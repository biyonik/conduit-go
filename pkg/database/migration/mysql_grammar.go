// -----------------------------------------------------------------------------
// MySQL Grammar for Migration System
// -----------------------------------------------------------------------------
// Bu dosya, MySQL database için SQL sorguları oluşturur.
// -----------------------------------------------------------------------------

package migration

import (
	"fmt"
	"strings"
)

// MySQLGrammar implements Grammar interface for MySQL.
type MySQLGrammar struct{}

// NewMySQLGrammar creates a new MySQLGrammar instance.
func NewMySQLGrammar() *MySQLGrammar {
	return &MySQLGrammar{}
}

// CompileCreateTable generates CREATE TABLE SQL.
func (g *MySQLGrammar) CompileCreateTable(table string, columns []Column, indexes []Index) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("CREATE TABLE `%s` (", table))

	// Add columns
	columnDefs := make([]string, 0, len(columns))
	for _, column := range columns {
		columnDefs = append(columnDefs, g.compileColumn(column))
	}

	parts = append(parts, "  "+strings.Join(columnDefs, ",\n  "))

	// Add indexes
	if len(indexes) > 0 {
		parts = append(parts, ",")
		indexDefs := make([]string, 0, len(indexes))
		for _, index := range indexes {
			indexDefs = append(indexDefs, g.compileIndex(index))
		}
		parts = append(parts, "  "+strings.Join(indexDefs, ",\n  "))
	}

	parts = append(parts, "\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")

	return strings.Join(parts, "\n")
}

// CompileDropTable generates DROP TABLE SQL.
func (g *MySQLGrammar) CompileDropTable(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)
}

// CompileAddColumn generates ALTER TABLE ADD COLUMN SQL.
func (g *MySQLGrammar) CompileAddColumn(table string, column Column) string {
	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", table, g.compileColumn(column))
}

// CompileDropColumn generates ALTER TABLE DROP COLUMN SQL.
func (g *MySQLGrammar) CompileDropColumn(table string, columnName string) string {
	return fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", table, columnName)
}

// CompileAddIndex generates ALTER TABLE ADD INDEX SQL.
func (g *MySQLGrammar) CompileAddIndex(table string, index Index) string {
	return fmt.Sprintf("ALTER TABLE `%s` ADD %s", table, g.compileIndex(index))
}

// CompileDropIndex generates ALTER TABLE DROP INDEX SQL.
func (g *MySQLGrammar) CompileDropIndex(table string, indexName string) string {
	return fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table, indexName)
}

// compileColumn compiles a single column definition.
func (g *MySQLGrammar) compileColumn(column Column) string {
	var parts []string

	// Column name and type
	parts = append(parts, fmt.Sprintf("`%s`", column.Name))

	// Type with length
	if column.Length > 0 && (column.Type == ColumnTypeString) {
		parts = append(parts, fmt.Sprintf("%s(%d)", column.Type, column.Length))
	} else {
		parts = append(parts, string(column.Type))
	}

	// Unsigned
	if column.Unsigned {
		parts = append(parts, "UNSIGNED")
	}

	// Nullable
	if !column.Nullable {
		parts = append(parts, "NOT NULL")
	} else {
		parts = append(parts, "NULL")
	}

	// Auto increment
	if column.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}

	// Default value
	if column.Default != nil {
		if str, ok := column.Default.(string); ok {
			parts = append(parts, fmt.Sprintf("DEFAULT '%s'", str))
		} else {
			parts = append(parts, fmt.Sprintf("DEFAULT %v", column.Default))
		}
	}

	// Primary key
	if column.Primary {
		parts = append(parts, "PRIMARY KEY")
	}

	return strings.Join(parts, " ")
}

// compileIndex compiles an index definition.
func (g *MySQLGrammar) compileIndex(index Index) string {
	columns := make([]string, len(index.Columns))
	for i, col := range index.Columns {
		columns[i] = fmt.Sprintf("`%s`", col)
	}

	switch index.Type {
	case IndexTypePrimary:
		return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(columns, ", "))
	case IndexTypeUnique:
		return fmt.Sprintf("UNIQUE KEY `%s` (%s)", index.Name, strings.Join(columns, ", "))
	case IndexTypeIndex:
		return fmt.Sprintf("INDEX `%s` (%s)", index.Name, strings.Join(columns, ", "))
	default:
		return fmt.Sprintf("INDEX `%s` (%s)", index.Name, strings.Join(columns, ", "))
	}
}
