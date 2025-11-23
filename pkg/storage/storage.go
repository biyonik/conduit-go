// -----------------------------------------------------------------------------
// Storage Package - Laravel-Inspired File Storage System
// -----------------------------------------------------------------------------
// Bu package, dosya depolama işlemleri için Laravel Storage Facade'ine
// benzer bir interface sağlar.
//
// Desteklenen Driver'lar:
// - Local: Yerel dosya sistemi
// - S3: Amazon S3 ve S3-compatible storage (MinIO, DigitalOcean Spaces, vb.)
//
// Özellikler:
// - Multiple driver support (local, S3)
// - Fluent API
// - Stream support (büyük dosyalar için)
// - Visibility control (public/private)
// - URL generation
// - Metadata support
//
// Kullanım:
//
//	storage := storage.NewLocalStorage("/uploads", logger)
//	err := storage.Put("avatars/user-1.jpg", fileData)
//	url := storage.Url("avatars/user-1.jpg")
// -----------------------------------------------------------------------------

package storage

import (
	"fmt"
	"io"
	"time"
)

// Storage, dosya depolama interface'i.
//
// Farklı storage driver'ları (Local, S3, FTP, vb.) bu interface'i
// implement ederek sistemle entegre olabilir.
type Storage interface {
	// Put, dosya yükler.
	//
	// Parametreler:
	//   - path: Dosya yolu (örn: "avatars/user-1.jpg")
	//   - contents: Dosya içeriği
	//
	// Döndürür:
	//   - error: Yükleme başarısızsa hata
	Put(path string, contents []byte) error

	// PutFile, io.Reader'dan dosya yükler (stream için).
	//
	// Büyük dosyalar için memory-efficient.
	//
	// Parametreler:
	//   - path: Dosya yolu
	//   - reader: Dosya içeriği reader'ı
	//
	// Döndürür:
	//   - error: Yükleme başarısızsa hata
	PutFile(path string, reader io.Reader) error

	// Get, dosya içeriğini okur.
	//
	// Parametre:
	//   - path: Dosya yolu
	//
	// Döndürür:
	//   - []byte: Dosya içeriği
	//   - error: Okuma başarısızsa hata
	Get(path string) ([]byte, error)

	// GetStream, dosyayı stream olarak okur (büyük dosyalar için).
	//
	// Parametre:
	//   - path: Dosya yolu
	//
	// Döndürür:
	//   - io.ReadCloser: Dosya stream'i (kullanım sonrası Close() çağrılmalı)
	//   - error: Okuma başarısızsa hata
	GetStream(path string) (io.ReadCloser, error)

	// Delete, dosyayı siler.
	//
	// Parametre:
	//   - path: Silinecek dosya yolu
	//
	// Döndürür:
	//   - error: Silme başarısızsa hata
	Delete(path string) error

	// Exists, dosyanın var olup olmadığını kontrol eder.
	//
	// Parametre:
	//   - path: Kontrol edilecek dosya yolu
	//
	// Döndürür:
	//   - bool: Dosya varsa true
	//   - error: Kontrol sırasında hata oluşursa
	Exists(path string) (bool, error)

	// Size, dosya boyutunu döndürür.
	//
	// Parametre:
	//   - path: Dosya yolu
	//
	// Döndürür:
	//   - int64: Dosya boyutu (byte)
	//   - error: Hata oluşursa
	Size(path string) (int64, error)

	// LastModified, dosyanın son değiştirme zamanını döndürür.
	//
	// Parametre:
	//   - path: Dosya yolu
	//
	// Döndürür:
	//   - time.Time: Son değiştirme zamanı
	//   - error: Hata oluşursa
	LastModified(path string) (time.Time, error)

	// Url, dosyanın erişilebilir URL'ini döndürür.
	//
	// Public dosyalar için HTTP URL döner.
	// Private dosyalar için signed URL gerekebilir (S3 için).
	//
	// Parametre:
	//   - path: Dosya yolu
	//
	// Döndürür:
	//   - string: Dosya URL'i
	Url(path string) string

	// Files, belirtilen dizindeki dosyaları listeler.
	//
	// Parametre:
	//   - directory: Dizin yolu (boş string ise root)
	//
	// Döndürür:
	//   - []string: Dosya yolları listesi
	//   - error: Listeleme başarısızsa hata
	Files(directory string) ([]string, error)

	// Directories, belirtilen dizindeki alt dizinleri listeler.
	//
	// Parametre:
	//   - directory: Dizin yolu (boş string ise root)
	//
	// Döndürür:
	//   - []string: Dizin yolları listesi
	//   - error: Listeleme başarısızsa hata
	Directories(directory string) ([]string, error)

	// MakeDirectory, dizin oluşturur.
	//
	// Parametre:
	//   - path: Oluşturulacak dizin yolu
	//
	// Döndürür:
	//   - error: Oluşturma başarısızsa hata
	MakeDirectory(path string) error

	// DeleteDirectory, dizini ve içeriğini siler.
	//
	// Parametre:
	//   - path: Silinecek dizin yolu
	//
	// Döndürür:
	//   - error: Silme başarısızsa hata
	DeleteDirectory(path string) error
}

// Logger interface - dependency injection için
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// -----------------------------------------------------------------------------
// File Info
// -----------------------------------------------------------------------------

// FileInfo, dosya meta bilgilerini içerir.
type FileInfo struct {
	Path         string    // Dosya yolu
	Size         int64     // Boyut (byte)
	LastModified time.Time // Son değiştirme zamanı
	IsDirectory  bool      // Dizin mi dosya mı
	MimeType     string    // MIME type (örn: "image/jpeg")
}

// -----------------------------------------------------------------------------
// Visibility
// -----------------------------------------------------------------------------

// Visibility, dosya görünürlük seviyesi.
type Visibility string

const (
	VisibilityPublic  Visibility = "public"  // Herkese açık
	VisibilityPrivate Visibility = "private" // Sadece authenticated kullanıcılar
)

// -----------------------------------------------------------------------------
// Storage Options
// -----------------------------------------------------------------------------

// PutOptions, dosya yükleme seçenekleri.
type PutOptions struct {
	Visibility Visibility        // Görünürlük (public/private)
	MimeType   string            // MIME type (auto-detect ise boş)
	Metadata   map[string]string // Özel metadata
}

// DefaultPutOptions, varsayılan yükleme seçenekleri.
var DefaultPutOptions = PutOptions{
	Visibility: VisibilityPrivate,
	Metadata:   make(map[string]string),
}

// -----------------------------------------------------------------------------
// Common Errors
// -----------------------------------------------------------------------------

var (
	ErrFileNotFound      = fmt.Errorf("file not found")
	ErrDirectoryNotFound = fmt.Errorf("directory not found")
	ErrFileAlreadyExists = fmt.Errorf("file already exists")
	ErrInvalidPath       = fmt.Errorf("invalid path")
	ErrPermissionDenied  = fmt.Errorf("permission denied")
)

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// IsImage, dosyanın image olup olmadığını kontrol eder.
//
// Parametre:
//   - path: Dosya yolu
//
// Döndürür:
//   - bool: Image ise true
func IsImage(path string) bool {
	ext := GetExtension(path)
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// GetExtension, dosya uzantısını döndürür.
//
// Parametre:
//   - path: Dosya yolu
//
// Döndürür:
//   - string: Uzantı (nokta ile birlikte, örn: ".jpg")
func GetExtension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			return ""
		}
	}
	return ""
}

// SanitizePath, dosya yolunu güvenli hale getirir.
//
// Path traversal saldırılarını önler (../ gibi).
//
// Parametre:
//   - path: Sanitize edilecek yol
//
// Döndürür:
//   - string: Güvenli yol
//   - error: Geçersiz yol ise hata
func SanitizePath(path string) (string, error) {
	// .. içeren yolları reddet (path traversal)
	if containsPathTraversal(path) {
		return "", ErrInvalidPath
	}

	// Başındaki ve sonundaki / karakterlerini temizle
	for len(path) > 0 && (path[0] == '/' || path[0] == '\\') {
		path = path[1:]
	}
	for len(path) > 0 && (path[len(path)-1] == '/' || path[len(path)-1] == '\\') {
		path = path[:len(path)-1]
	}

	return path, nil
}

// containsPathTraversal, path traversal içerip içermediğini kontrol eder.
func containsPathTraversal(path string) bool {
	// .. sequence kontrolü
	for i := 0; i < len(path)-1; i++ {
		if path[i] == '.' && path[i+1] == '.' {
			// ..'den önce veya sonra separator olmalı
			if i == 0 || path[i-1] == '/' || path[i-1] == '\\' {
				if i+2 >= len(path) || path[i+2] == '/' || path[i+2] == '\\' {
					return true
				}
			}
		}
	}
	return false
}

// GenerateUniqueName, benzersiz dosya adı oluşturur.
//
// Parametreler:
//   - originalName: Orijinal dosya adı
//
// Döndürür:
//   - string: Benzersiz dosya adı (timestamp + original)
//
// Örnek:
//
//	uniqueName := storage.GenerateUniqueName("photo.jpg")
//	// → "1704067200-photo.jpg"
func GenerateUniqueName(originalName string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%d-%s", timestamp, originalName)
}
