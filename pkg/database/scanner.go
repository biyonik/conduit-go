package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// Reflection-Based SQL Scanner (MEMORY LEAK FIX)
// -----------------------------------------------------------------------------
// Bu dosya, SQL query sonuçlarını Go struct'larına reflection kullanarak tarar.
// 'db' tag'lerine göre kolon-alan eşleşmesi yapar.
//
// MEMORY LEAK FIX:
// scannerCache artık periyodik olarak temizleniyor. Aksi takdirde her farklı
// struct tipi için cache büyümeye devam eder ve memory leak oluşur.
// -----------------------------------------------------------------------------

// scannerCacheEntry, cache entry'lerinin metadata'sını tutar.
type scannerCacheEntry struct {
	fieldMap   fieldMap  // Kolon adı -> alan adı mapping
	lastAccess time.Time // Son erişim zamanı (cleanup için)
}

// scannerCache, struct tipleri için 'db' tag eşleşmelerini önbelleğe alır.
// MEMORY LEAK FIX: sync.Map yerine map[reflect.Type]*scannerCacheEntry kullanıyoruz.
var (
	scannerCache   = make(map[reflect.Type]*scannerCacheEntry)
	scannerCacheMu sync.RWMutex
)

// fieldMap, bir struct tipi için sütun adı -> alan adı eşleşme haritasıdır.
type fieldMap map[string]string

// init, scanner cache cleanup goroutine'ini başlatır.
//
// MEMORY LEAK KORUNMASI:
// Bu fonksiyon package init edildiğinde otomatik olarak çalışır.
// Her 10 dakikada bir, 30 dakikadan uzun süredir erişilmeyen cache entry'lerini siler.
func init() {
	go cleanupScannerCache(10*time.Minute, 30*time.Minute)
}

// cleanupScannerCache, periyodik olarak kullanılmayan cache entry'lerini temizler.
//
// Parametreler:
//   - interval: Cleanup'ın çalışma aralığı
//   - maxAge: Entry'lerin maksimum idle süresi
//
// MEMORY LEAK KORUNMASI:
// Bu fonksiyon, struct tiplerinin cache'de süresiz kalmasını önler.
// Örneğin, test ortamında binlerce farklı anonymous struct oluşturulursa,
// bu cleanup olmadan memory sürekli büyür.
func cleanupScannerCache(interval, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		scannerCacheMu.Lock()

		now := time.Now()
		for typ, entry := range scannerCache {
			// maxAge'den uzun süredir erişilmeyen entry'leri sil
			if now.Sub(entry.lastAccess) > maxAge {
				delete(scannerCache, typ)
			}
		}

		scannerCacheMu.Unlock()
	}
}

// getStructFieldMap, bir struct tipini analiz eder ve 'db' tag'lerine göre
// bir sütun-alan eşleşme haritası oluşturur.
//
// Parametre:
//   - structType: Analiz edilecek struct tipi
//
// Döndürür:
//   - fieldMap: Kolon adı -> alan adı mapping
//
// Özellikler:
// - Embedded (gömülü) struct'ları özyineli olarak işler (örn: BaseModel)
// - Cache'den okur, yoksa hesaplar ve cache'e yazar
// - db:"-" tag'i olan alanları atlar
// - Her erişimde lastAccess zamanını günceller (cleanup için)
func getStructFieldMap(structType reflect.Type) fieldMap {
	// 1. Cache'den kontrol et (read lock)
	scannerCacheMu.RLock()
	if entry, ok := scannerCache[structType]; ok {
		// Son erişim zamanını güncelle
		entry.lastAccess = time.Now()
		scannerCacheMu.RUnlock()
		return entry.fieldMap
	}
	scannerCacheMu.RUnlock()

	// 2. Cache'de yok, struct'ı analiz et (write lock)
	scannerCacheMu.Lock()
	defer scannerCacheMu.Unlock()

	// Double-check: başka goroutine bizden önce ekmiş olabilir
	if entry, ok := scannerCache[structType]; ok {
		entry.lastAccess = time.Now()
		return entry.fieldMap
	}

	// 3. Struct field'larını analiz et
	mapping := make(fieldMap)
	numFields := structType.NumField()

	for i := 0; i < numFields; i++ {
		field := structType.Field(i)

		// Embedded (gömülü) struct'ları özyineli olarak işle
		if field.Anonymous {
			if field.Type.Kind() == reflect.Struct {
				// BaseModel gibi gömülü struct'ların field'larını ekle
				for col, fName := range getStructFieldMap(field.Type) {
					mapping[col] = field.Name + "." + fName
				}
			}
			continue
		}

		// 'db' tag'ini oku
		tag := field.Tag.Get("db")

		// db:"-" ise atla
		if tag == "-" {
			continue
		}

		// Tag yoksa field adını lowercase yap (basit convention)
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		mapping[tag] = field.Name
	}

	// 4. Cache'e kaydet
	scannerCache[structType] = &scannerCacheEntry{
		fieldMap:   mapping,
		lastAccess: time.Now(),
	}

	return mapping
}

// ScanStruct, tek bir *sql.Rows satırını bir struct'a tarar.
//
// Parametre:
//   - rows: SQL query sonucu (rows.Next() çağrılmış olmalı)
//   - dest: Hedef struct pointer (örn: &models.User)
//
// Döndürür:
//   - error: Tarama hatası varsa
//
// Örnek:
//
//	var user User
//	rows, _ := db.Query("SELECT * FROM users WHERE id = ?", 1)
//	if rows.Next() {
//	    ScanStruct(rows, &user)
//	}
//
// Güvenlik Notu:
// Bu fonksiyon reflection kullanır. Performance-critical kod için
// manuel struct scanning tercih edilebilir.
func ScanStruct(rows *sql.Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("scanner: dest bir struct pointer olmalıdır, %T alındı", dest)
	}

	destElem := destValue.Elem()
	destType := destElem.Type()

	// Kolon adlarını al
	cols, _ := rows.Columns()
	fieldMap := getStructFieldMap(destType)

	// rows.Scan() için []any slice'ı oluştur
	scanArgs := make([]any, len(cols))

	for i, colName := range cols {
		// Sütun adıyla eşleşen struct alanını bul
		fieldName, ok := fieldMap[colName]
		if !ok {
			// Eşleşen alan yoksa, veriyi boşa (dummy) tara
			scanArgs[i] = new(sql.RawBytes)
			continue
		}

		// Alan adına göre struct alanını bul
		fieldVal := destElem.FieldByName(fieldName)

		// Embedded struct'lar için (örn: "BaseModel.ID")
		if !fieldVal.IsValid() {
			fieldVal = findEmbeddedField(destElem, fieldName)
		}

		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			return fmt.Errorf("scanner: '%s' alanı bulunamadı veya ayarlanamıyor", fieldName)
		}

		// rows.Scan() için alanın adresini ver
		scanArgs[i] = fieldVal.Addr().Interface()
	}

	// Veritabanından veriyi tara
	if err := rows.Scan(scanArgs...); err != nil {
		return err
	}

	return nil
}

// findEmbeddedField, 'A.B' gibi iç içe alan adlarını bulur.
//
// Parametre:
//   - v: Struct reflection Value
//   - name: Alan adı (nokta ile ayrılmış, örn: "BaseModel.ID")
//
// Döndürür:
//   - reflect.Value: Bulunan alan (veya geçersiz Value)
//
// Örnek:
//
//	// User struct'ı BaseModel'i gömüyor
//	type User struct {
//	    BaseModel // ID, CreatedAt, UpdatedAt
//	    Name string
//	}
//	findEmbeddedField(userValue, "BaseModel.ID") → ID field'ının Value'sı
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
//
// Parametre:
//   - rows: SQL query sonucu
//   - dest: Hedef slice pointer (örn: &[]models.User)
//
// Döndürür:
//   - error: Tarama hatası varsa
//
// Örnek:
//
//	var users []User
//	rows, _ := db.Query("SELECT * FROM users WHERE status = ?", "active")
//	ScanSlice(rows, &users)
//
// Performans Notu:
// Bu fonksiyon her satır için reflection kullanır. Binlerce satır için
// bulk insert gibi optimize edilmiş metotlar tercih edilebilir.
func ScanSlice(rows *sql.Rows, dest any) error {
	sliceValue := reflect.ValueOf(dest)
	if sliceValue.Kind() != reflect.Ptr || sliceValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("scanner: dest bir slice pointer olmalıdır, %T alındı", dest)
	}

	sliceElem := sliceValue.Elem()
	structType := sliceElem.Type().Elem()

	// rows.Next() döngüsü
	for rows.Next() {
		// Slice'ın eleman tipinden yeni bir pointer oluştur
		newStructPtr := reflect.New(structType)

		// Yeni struct'a veriyi tara
		if err := ScanStruct(rows, newStructPtr.Interface()); err != nil {
			return err
		}

		// Dolu struct'ı slice'a ekle
		sliceElem.Set(reflect.Append(sliceElem, newStructPtr.Elem()))
	}

	return rows.Err()
}
