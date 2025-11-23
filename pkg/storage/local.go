// -----------------------------------------------------------------------------
// Local Storage Driver
// -----------------------------------------------------------------------------
// Bu driver, yerel dosya sistemi üzerinde dosya depolama sağlar.
//
// Özellikler:
// - Fast access (local filesystem)
// - No external dependencies
// - Simple setup
// - Ideal for development and small-scale deployments
//
// Kullanım:
//
//	storage := storage.NewLocalStorage("/var/www/uploads", logger)
//	err := storage.Put("avatars/user-1.jpg", imageData)
//	url := storage.Url("avatars/user-1.jpg") // → "/uploads/avatars/user-1.jpg"
// -----------------------------------------------------------------------------

package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage, yerel dosya sisteminde depolama yapan driver.
type LocalStorage struct {
	basePath string // Temel dizin yolu (örn: "/var/www/uploads")
	baseURL  string // Temel URL (örn: "/uploads" veya "https://cdn.example.com")
	logger   Logger
}

// NewLocalStorage, yeni bir LocalStorage oluşturur.
//
// Parametreler:
//   - basePath: Dosyaların saklanacağı dizin (mutlak yol)
//   - logger: Logger instance
//
// Döndürür:
//   - *LocalStorage: Yeni local storage instance
//
// Örnek:
//
//	storage := storage.NewLocalStorage("/var/www/uploads", logger)
//
// Not:
// basePath dizini yoksa otomatik oluşturulur.
func NewLocalStorage(basePath string, logger Logger) (*LocalStorage, error) {
	// Dizini oluştur (yoksa)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	logger.Printf("✅ Local storage initialized: %s", basePath)

	return &LocalStorage{
		basePath: basePath,
		baseURL:  "/uploads", // Varsayılan URL prefix
		logger:   logger,
	}, nil
}

// SetBaseURL, URL prefix'ini ayarlar.
//
// Parametre:
//   - baseURL: URL prefix (örn: "/uploads" veya "https://cdn.example.com")
//
// Örnek:
//
//	storage.SetBaseURL("https://cdn.myapp.com")
func (s *LocalStorage) SetBaseURL(baseURL string) {
	s.baseURL = baseURL
}

// Put, dosya yükler.
func (s *LocalStorage) Put(path string, contents []byte) error {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dizini oluştur (yoksa)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Dosyayı yaz
	if err := os.WriteFile(fullPath, contents, 0644); err != nil {
		s.logger.Printf("❌ Failed to write file: %s - %v", sanitized, err)
		return fmt.Errorf("failed to write file: %w", err)
	}

	s.logger.Printf("✅ File saved: %s (%d bytes)", sanitized, len(contents))

	return nil
}

// PutFile, io.Reader'dan dosya yükler (stream).
func (s *LocalStorage) PutFile(path string, reader io.Reader) error {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dizini oluştur (yoksa)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Dosyayı oluştur
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Reader'dan dosyaya kopyala
	written, err := io.Copy(file, reader)
	if err != nil {
		s.logger.Printf("❌ Failed to write file stream: %s - %v", sanitized, err)
		return fmt.Errorf("failed to write file stream: %w", err)
	}

	s.logger.Printf("✅ File saved (stream): %s (%d bytes)", sanitized, written)

	return nil
}

// Get, dosya içeriğini okur.
func (s *LocalStorage) Get(path string) ([]byte, error) {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return nil, err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dosyayı oku
	contents, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return contents, nil
}

// GetStream, dosyayı stream olarak okur.
func (s *LocalStorage) GetStream(path string) (io.ReadCloser, error) {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return nil, err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dosyayı aç
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete, dosyayı siler.
func (s *LocalStorage) Delete(path string) error {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dosyayı sil
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Printf("🗑️  File deleted: %s", sanitized)

	return nil
}

// Exists, dosyanın var olup olmadığını kontrol eder.
func (s *LocalStorage) Exists(path string) (bool, error) {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return false, err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dosya var mı kontrol et
	_, err = os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Size, dosya boyutunu döndürür.
func (s *LocalStorage) Size(path string) (int64, error) {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return 0, err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dosya bilgisi al
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrFileNotFound
		}
		return 0, err
	}

	return info.Size(), nil
}

// LastModified, dosyanın son değiştirme zamanını döndürür.
func (s *LocalStorage) LastModified(path string) (time.Time, error) {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return time.Time{}, err
	}

	// Tam dosya yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dosya bilgisi al
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, ErrFileNotFound
		}
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

// Url, dosyanın URL'ini döndürür.
func (s *LocalStorage) Url(path string) string {
	// Path'i sanitize et (hata ignore et, URL generation için)
	sanitized, _ := SanitizePath(path)

	// / ile ayır
	if !strings.HasSuffix(s.baseURL, "/") && !strings.HasPrefix(sanitized, "/") {
		return s.baseURL + "/" + sanitized
	}

	return s.baseURL + sanitized
}

// Files, belirtilen dizindeki dosyaları listeler.
func (s *LocalStorage) Files(directory string) ([]string, error) {
	// Path'i sanitize et
	sanitized := directory
	if directory != "" {
		var err error
		sanitized, err = SanitizePath(directory)
		if err != nil {
			return nil, err
		}
	}

	// Tam dizin yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dizini oku
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}

	// Sadece dosyaları al
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			// Relative path döndür
			relativePath := filepath.Join(sanitized, entry.Name())
			files = append(files, relativePath)
		}
	}

	return files, nil
}

// Directories, belirtilen dizindeki alt dizinleri listeler.
func (s *LocalStorage) Directories(directory string) ([]string, error) {
	// Path'i sanitize et
	sanitized := directory
	if directory != "" {
		var err error
		sanitized, err = SanitizePath(directory)
		if err != nil {
			return nil, err
		}
	}

	// Tam dizin yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dizini oku
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}

	// Sadece dizinleri al
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Relative path döndür
			relativePath := filepath.Join(sanitized, entry.Name())
			dirs = append(dirs, relativePath)
		}
	}

	return dirs, nil
}

// MakeDirectory, dizin oluşturur.
func (s *LocalStorage) MakeDirectory(path string) error {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return err
	}

	// Tam dizin yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dizini oluştur
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	s.logger.Printf("📁 Directory created: %s", sanitized)

	return nil
}

// DeleteDirectory, dizini ve içeriğini siler.
func (s *LocalStorage) DeleteDirectory(path string) error {
	// Path'i sanitize et
	sanitized, err := SanitizePath(path)
	if err != nil {
		return err
	}

	// Tam dizin yolu
	fullPath := filepath.Join(s.basePath, sanitized)

	// Dizini sil (recursive)
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}

	s.logger.Printf("🗑️  Directory deleted: %s", sanitized)

	return nil
}

// GetBasePath, base path'i döndürür (testing için).
func (s *LocalStorage) GetBasePath() string {
	return s.basePath
}
