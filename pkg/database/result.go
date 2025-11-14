package database

import (
	"database/sql"
)

// -----------------------------------------------------------------------------
// RESULT HELPERS
// -----------------------------------------------------------------------------
// Bu dosya, SQL'den dönen sonuçları okuma ve map[string]interface{} şeklinde
// döndürme yardımcılarını içerir. Basit ve generic bir dönüşüm mekanizması
// sağlar; küçük projeler için yeterlidir.
// -----------------------------------------------------------------------------

// rowsToMaps: sql.Rows'ı []map[string]interface{} biçimine dönüştürür.
func rowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	res := make([]map[string]interface{}, 0)

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}

		res = append(res, m)
	}

	return res, nil
}
