// pkg/database/scanner.go
package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// scannerCache, struct tipleri için 'db' tag eşleşmelerini önbelleğe alır.
// Bu, reflect işleminin maliyetini azaltır.
var scannerCache = sync.Map{} // map[reflect.Type]map[string]string

// fieldMap, bir struct tipi için sütun adı -> alan adı eşleşmesini tutar.
type fieldMap map[string]string

// getStructFieldMap, bir struct tipini analiz eder ve 'db' tag'lerine göre
// bir sütun-alan eşleşme haritası oluşturur.
func getStructFieldMap(structType reflect.Type) fieldMap {
	// 1. Önbellekten kontrol et
	if cachedMap, ok := scannerCache.Load(structType); ok {
		return cachedMap.(fieldMap)
	}

	// 2. Önbellekte yoksa, struct'ı analiz et
	mapping := make(fieldMap)
	numFields := structType.NumField()
	for i := 0; i < numFields; i++ {
		field := structType.Field(i)

		// Gömülü (embedded) struct'ları (örn: models.BaseModel) özyineli olarak işle
		if field.Anonymous {
			// (Go 1.20+ 'unexported' embedded field desteği için ek kontrol gerekebilir)
			if field.Type.Kind() == reflect.Struct {
				for col, fName := range getStructFieldMap(field.Type) {
					mapping[col] = field.Name + "." + fName
				}
			}
			continue
		}

		// 'db' tag'ini oku
		tag := field.Tag.Get("db")
		if tag == "" {
			// 'db' tag'i yoksa, alan adını (snake_case) kullanmayı deneyebiliriz
			// Şimdilik sadece tag olanları alalım
			tag = strings.ToLower(field.Name) // Basit bir varsayım
		}

		// json:"-" veya db:"-" olanları atla
		if tag == "-" {
			continue
		}

		mapping[tag] = field.Name
	}

	// 3. Önbelleğe kaydet
	scannerCache.Store(structType, mapping)
	return mapping
}

// ScanStruct, tek bir *sql.Rows satırını bir struct'a tarar.
// dest, bir struct'a pointer olmalıdır (örn: &models.User).
func ScanStruct(rows *sql.Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("scanner: dest bir struct pointer olmalıdır, %T alındı", dest)
	}
	destElem := destValue.Elem()
	destType := destElem.Type()

	cols, _ := rows.Columns()
	fieldMap := getStructFieldMap(destType)

	// 'rows.Scan()' için '[]any' (veya '[]interface{}') slice'ı oluştur
	scanArgs := make([]any, len(cols))

	for i, colName := range cols {
		// Sütun adıyla eşleşen struct alanını bul
		fieldName, ok := fieldMap[colName]
		if !ok {
			// Eşleşen alan yoksa, veriyi 'boşa' (dummy) tara
			scanArgs[i] = new(sql.RawBytes)
			continue
		}

		// Alan adına göre struct alanını bul
		// Gömülü struct'ları desteklemek için (örn: "BaseModel.ID")
		fieldVal := destElem.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			// Gömülü alanları 'FieldByNameFunc' ile ara (daha yavaş ama doğru)
			fieldVal = findEmbeddedField(destElem, fieldName)
		}

		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			return fmt.Errorf("scanner: '%s' alanı bulunamadı veya ayarlanamıyor", fieldName)
		}

		// 'rows.Scan()' için alanın adresini (pointer) ver
		scanArgs[i] = fieldVal.Addr().Interface()
	}

	// Veritabanından veriyi tara
	if err := rows.Scan(scanArgs...); err != nil {
		return err
	}

	return nil
}

// findEmbeddedField, 'A.B' gibi iç içe alan adlarını bulur.
func findEmbeddedField(v reflect.Value, name string) reflect.Value {
	parts := strings.Split(name, ".")
	current := v
	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		current = current.FieldByName(part)
	}
	return current
}

// ScanSlice, tüm *sql.Rows sonuç kümesini bir struct slice'ına tarar.
// dest, bir struct slice'ına pointer olmalıdır (örn: &[]models.User).
func ScanSlice(rows *sql.Rows, dest any) error {
	sliceValue := reflect.ValueOf(dest)
	if sliceValue.Kind() != reflect.Ptr || sliceValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("scanner: dest bir slice pointer olmalıdır, %T alındı", dest)
	}

	sliceElem := sliceValue.Elem()
	structType := sliceElem.Type().Elem() // Slice'ın eleman tipini al (örn: models.User)

	// 'rows.Next()' döngüsü
	for rows.Next() {
		// Slice'ın eleman tipi neyse (örn: models.User) onun yeni bir pointer'ını oluştur
		newStructPtr := reflect.New(structType) // *models.User

		// Yeni oluşturulan struct pointer'ına veriyi tara
		if err := ScanStruct(rows, newStructPtr.Interface()); err != nil {
			return err
		}

		// Dolu struct'ı (pointer değil, değerini) slice'a ekle
		sliceElem.Set(reflect.Append(sliceElem, newStructPtr.Elem()))
	}

	return rows.Err()
}
