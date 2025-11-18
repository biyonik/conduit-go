package database

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// Reflection-Based SQL Scanner (MEMORY LEAK FIXED)
// -----------------------------------------------------------------------------
// FIXED ISSUES:
// ✅ Goroutine leak - cleanup artık gracefully durdurulabiliyor
// ✅ Context-based shutdown mekanizması eklendi
// ✅ Global scanner instance'ı ile lifecycle management
// -----------------------------------------------------------------------------

// scannerCacheEntry, cache entry'lerinin metadata'sını tutar.
type scannerCacheEntry struct {
	fieldMap   fieldMap
	lastAccess time.Time
}

type fieldMap map[string]string

// Scanner, cache yönetimi ve cleanup lifecycle'ını kontrol eder.
type Scanner struct {
	cache      map[reflect.Type]*scannerCacheEntry
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	cleanupInt time.Duration
	maxAge     time.Duration
}

// Global scanner instance
var globalScanner *Scanner
var scannerOnce sync.Once

// InitScanner, global scanner instance'ını başlatır.
// Bu fonksiyon main.go'dan çağrılmalı.
func InitScanner(cleanupInterval, maxAge time.Duration) *Scanner {
	scannerOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		globalScanner = &Scanner{
			cache:      make(map[reflect.Type]*scannerCacheEntry),
			ctx:        ctx,
			cancel:     cancel,
			cleanupInt: cleanupInterval,
			maxAge:     maxAge,
		}
		globalScanner.startCleanup()
	})
	return globalScanner
}

// GetScanner, global scanner instance'ını döndürür.
// InitScanner çağrılmamışsa default değerlerle başlatır.
func GetScanner() *Scanner {
	if globalScanner == nil {
		return InitScanner(10*time.Minute, 30*time.Minute)
	}
	return globalScanner
}

// startCleanup, cleanup goroutine'ini başlatır.
func (s *Scanner) startCleanup() {
	s.wg.Add(1)
	go s.cleanupLoop()
}

// cleanupLoop, periyodik olarak kullanılmayan cache entry'lerini temizler.
func (s *Scanner) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.ctx.Done():
			// Graceful shutdown
			return
		}
	}
}

// cleanup, expired entry'leri temizler.
func (s *Scanner) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for typ, entry := range s.cache {
		if now.Sub(entry.lastAccess) > s.maxAge {
			delete(s.cache, typ)
			cleaned++
		}
	}

	if cleaned > 0 {
		// Debug log için - production'da kaldırılabilir
		_ = cleaned
	}
}

// Stop, scanner'ı gracefully durdurur.
// Bu fonksiyon main.go'daki shutdown hook'undan çağrılmalı.
func (s *Scanner) Stop() {
	s.cancel()
	s.wg.Wait()
}

// getStructFieldMap, bir struct tipini analiz eder ve cache'den döndürür.
func (s *Scanner) getStructFieldMap(structType reflect.Type) fieldMap {
	// Read lock ile cache'i kontrol et
	s.mu.RLock()
	if entry, ok := s.cache[structType]; ok {
		entry.lastAccess = time.Now()
		s.mu.RUnlock()
		return entry.fieldMap
	}
	s.mu.RUnlock()

	// Cache miss - write lock al
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check pattern
	if entry, ok := s.cache[structType]; ok {
		entry.lastAccess = time.Now()
		return entry.fieldMap
	}

	// Struct field'larını analiz et
	mapping := make(fieldMap)
	numFields := structType.NumField()

	for i := 0; i < numFields; i++ {
		field := structType.Field(i)

		// Embedded struct'ları özyineli işle
		if field.Anonymous {
			if field.Type.Kind() == reflect.Struct {
				for col, fName := range s.getStructFieldMap(field.Type) {
					mapping[col] = field.Name + "." + fName
				}
			}
			continue
		}

		tag := field.Tag.Get("db")
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		mapping[tag] = field.Name
	}

	// Cache'e kaydet
	s.cache[structType] = &scannerCacheEntry{
		fieldMap:   mapping,
		lastAccess: time.Now(),
	}

	return mapping
}

// ScanStruct, tek bir *sql.Rows satırını bir struct'a tarar.
func ScanStruct(rows *sql.Rows, dest any) error {
	scanner := GetScanner()

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("scanner: dest bir struct pointer olmalıdır, %T alındı", dest)
	}

	destElem := destValue.Elem()
	destType := destElem.Type()

	cols, _ := rows.Columns()
	fieldMap := scanner.getStructFieldMap(destType)

	scanArgs := make([]any, len(cols))

	for i, colName := range cols {
		fieldName, ok := fieldMap[colName]
		if !ok {
			scanArgs[i] = new(sql.RawBytes)
			continue
		}

		fieldVal := destElem.FieldByName(fieldName)

		if !fieldVal.IsValid() {
			fieldVal = findEmbeddedField(destElem, fieldName)
		}

		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			return fmt.Errorf("scanner: '%s' alanı bulunamadı veya ayarlanamıyor", fieldName)
		}

		scanArgs[i] = fieldVal.Addr().Interface()
	}

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
func ScanSlice(rows *sql.Rows, dest any) error {
	sliceValue := reflect.ValueOf(dest)
	if sliceValue.Kind() != reflect.Ptr || sliceValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("scanner: dest bir slice pointer olmalıdır, %T alındı", dest)
	}

	sliceElem := sliceValue.Elem()
	structType := sliceElem.Type().Elem()

	for rows.Next() {
		newStructPtr := reflect.New(structType)

		if err := ScanStruct(rows, newStructPtr.Interface()); err != nil {
			return err
		}

		sliceElem.Set(reflect.Append(sliceElem, newStructPtr.Elem()))
	}

	return rows.Err()
}