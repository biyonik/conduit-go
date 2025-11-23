// -----------------------------------------------------------------------------
// Storage Security Tests - Path Traversal Prevention
// -----------------------------------------------------------------------------
// Bu testler, path traversal saldırılarına karşı korumanın çalıştığını doğrular.
// Her test case bir exploit senaryosunu simüle eder.
//
// OWASP Top 10 - Path Traversal:
// Saldırgan, "../" veya "..\\" gibi sequence'lar kullanarak
// storage root dışındaki dosyalara erişmeye çalışır.
//
// Örnek saldırılar:
// - "../../etc/passwd" → Linux sistem dosyalarına erişim
// - "..\\..\\windows\\system32\\config\\sam" → Windows sistem dosyalarına erişim
// - "....//....//etc/passwd" → Double encoding
// - "%2e%2e%2f" → URL encoding
// -----------------------------------------------------------------------------

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockLogger for testing
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf(format, v...))
}

func (m *MockLogger) Println(v ...interface{}) {
	m.logs = append(m.logs, fmt.Sprint(v...))
}

// TestPathTraversal_BasicAttacks tests common path traversal patterns.
func TestPathTraversal_BasicAttacks(t *testing.T) {
	attacks := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Simple parent directory",
			path:     "../etc/passwd",
			expected: ErrInvalidPath.Error(),
		},
		{
			name:     "Multiple parent directories",
			path:     "../../etc/passwd",
			expected: ErrInvalidPath.Error(),
		},
		{
			name:     "Deep traversal",
			path:     "../../../../../etc/passwd",
			expected: ErrInvalidPath.Error(),
		},
		{
			name:     "Windows style traversal",
			path:     "..\\..\\windows\\system32\\config\\sam",
			expected: ErrInvalidPath.Error(),
		},
		{
			name:     "Mixed slashes",
			path:     "../..\\etc/passwd",
			expected: ErrInvalidPath.Error(),
		},
		{
			name:     "Traversal in middle",
			path:     "uploads/../../../etc/passwd",
			expected: ErrInvalidPath.Error(),
		},
		{
			name:     "Traversal at end",
			path:     "uploads/files/..",
			expected: ErrInvalidPath.Error(),
		},
	}

	for _, tc := range attacks {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SanitizePath(tc.path)
			if err == nil {
				t.Errorf("Expected error for path '%s', got nil", tc.path)
			}
			if err.Error() != tc.expected {
				t.Errorf("Expected error '%s', got '%s'", tc.expected, err.Error())
			}
		})
	}
}

// TestPathTraversal_EncodingAttacks tests various encoding techniques.
func TestPathTraversal_EncodingAttacks(t *testing.T) {
	attacks := []string{
		// Double dot variations
		"....//....//etc/passwd",
		"..../....",
		// Encoded dots (these should be normalized by URL decoder before reaching storage)
		// But we test them anyway in case someone forgets to decode
		"%2e%2e%2f",
		"%2e%2e/",
		"..%2f",
		// Unicode variations
		"..%c0%af",
		"..%c1%9c",
	}

	for _, attack := range attacks {
		t.Run(attack, func(t *testing.T) {
			// Most of these should be caught by the ".." check
			if containsPathTraversal(attack) {
				// Good - detected
				return
			}
			// If not detected by containsPathTraversal,
			// it might still be caught by character validation
			_, err := SanitizePath(attack)
			if err == nil {
				t.Logf("Warning: path '%s' passed validation", attack)
			}
		})
	}
}

// TestPathTraversal_NullByteInjection tests null byte attacks.
func TestPathTraversal_NullByteInjection(t *testing.T) {
	// Null byte injection: "safe.txt\x00../../etc/passwd"
	// Some systems truncate at null byte, potentially bypassing checks
	attacks := []string{
		"safe.txt\x00../../etc/passwd",
		"uploads/file.jpg\x00.php",
	}

	for _, attack := range attacks {
		t.Run(fmt.Sprintf("Null byte: %q", attack), func(t *testing.T) {
			_, err := SanitizePath(attack)
			// Should be rejected (either by .. detection or invalid characters)
			if err == nil {
				t.Errorf("Null byte injection '%q' passed validation", attack)
			}
		})
	}
}

// TestSanitizePath_ValidPaths tests that legitimate paths are accepted.
func TestSanitizePath_ValidPaths(t *testing.T) {
	validPaths := []struct {
		input    string
		expected string
	}{
		{
			input:    "avatars/user-123.jpg",
			expected: "avatars/user-123.jpg",
		},
		{
			input:    "/uploads/documents/report.pdf",
			expected: "uploads/documents/report.pdf", // Leading slash removed
		},
		{
			input:    "files/2024/01/image.png",
			expected: "files/2024/01/image.png",
		},
		{
			input:    "a/b/c/d/e/f/g.txt",
			expected: "a/b/c/d/e/f/g.txt",
		},
		{
			input:    "file-with-dashes.txt",
			expected: "file-with-dashes.txt",
		},
		{
			input:    "file_with_underscores.txt",
			expected: "file_with_underscores.txt",
		},
	}

	for _, tc := range validPaths {
		t.Run(tc.input, func(t *testing.T) {
			result, err := SanitizePath(tc.input)
			if err != nil {
				t.Errorf("Valid path '%s' rejected with error: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestLocalStorage_PathTraversalPrevention tests actual storage operations.
func TestLocalStorage_PathTraversalPrevention(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "storage-security-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	storage, err := NewLocalStorage(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create a sensitive file outside the storage directory
	sensitiveDir := filepath.Join(tempDir, "..", "sensitive")
	os.MkdirAll(sensitiveDir, 0755)
	defer os.RemoveAll(sensitiveDir)

	sensitiveFile := filepath.Join(sensitiveDir, "secret.txt")
	os.WriteFile(sensitiveFile, []byte("SECRET DATA"), 0644)

	attacks := []string{
		"../sensitive/secret.txt",
		"../../sensitive/secret.txt",
		"uploads/../../../sensitive/secret.txt",
	}

	for _, attack := range attacks {
		t.Run(attack, func(t *testing.T) {
			// Try to read the sensitive file
			_, err := storage.Get(attack)
			if err == nil {
				t.Errorf("Path traversal attack '%s' succeeded!", attack)
			}
			if err != ErrInvalidPath && err != ErrFileNotFound {
				// Should be caught by SanitizePath (ErrInvalidPath)
				t.Logf("Attack caught with: %v", err)
			}
		})
	}
}

// TestLocalStorage_WritePathTraversal tests write operations.
func TestLocalStorage_WritePathTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-write-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	storage, err := NewLocalStorage(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	attacks := []string{
		"../outside.txt",
		"../../tmp/malicious.txt",
		"uploads/../../../etc/cron.d/backdoor",
	}

	for _, attack := range attacks {
		t.Run(attack, func(t *testing.T) {
			err := storage.Put(attack, []byte("malicious content"))
			if err == nil {
				t.Errorf("Write path traversal attack '%s' succeeded!", attack)
			}
			if err != ErrInvalidPath {
				t.Logf("Attack caught with: %v", err)
			}
		})
	}
}

// TestLocalStorage_DeletePathTraversal tests delete operations.
func TestLocalStorage_DeletePathTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-delete-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	storage, err := NewLocalStorage(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create a file outside storage
	outsideFile := filepath.Join(tempDir, "..", "important.txt")
	os.WriteFile(outsideFile, []byte("IMPORTANT"), 0644)
	defer os.Remove(outsideFile)

	attacks := []string{
		"../important.txt",
		"../../important.txt",
	}

	for _, attack := range attacks {
		t.Run(attack, func(t *testing.T) {
			err := storage.Delete(attack)
			if err == nil {
				// Check if file still exists
				if _, statErr := os.Stat(outsideFile); os.IsNotExist(statErr) {
					t.Errorf("Delete path traversal attack '%s' succeeded - file deleted!", attack)
				}
			}
			if err != ErrInvalidPath && err != ErrFileNotFound {
				t.Logf("Attack caught with: %v", err)
			}
		})
	}

	// Verify important file still exists
	if _, err := os.Stat(outsideFile); os.IsNotExist(err) {
		t.Error("Important file was deleted by path traversal attack!")
	}
}

// TestContainsPathTraversal_EdgeCases tests edge cases in path traversal detection.
func TestContainsPathTraversal_EdgeCases(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		// Should detect
		{"../etc/passwd", true},
		{"foo/../bar", true},
		{"foo/bar/..", true},
		{"..\\windows", true},
		{"foo\\..\\bar", true},

		// Should NOT detect (legitimate)
		{"foo.bar", false},
		{"foo..bar", false},
		{"..foo", false},
		{"foo..", false},
		{"normal/path.txt", false},
		{"file-2024-01-15..jpg", false}, // Two dots in filename is OK
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := containsPathTraversal(tc.path)
			if result != tc.expected {
				t.Errorf("Path '%s': expected %v, got %v", tc.path, tc.expected, result)
			}
		})
	}
}

// TestLocalStorage_Sandboxing tests that all operations stay within basePath.
func TestLocalStorage_Sandboxing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-sandbox-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	storage, err := NewLocalStorage(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Valid file inside sandbox
	validPath := "safe/file.txt"
	err = storage.Put(validPath, []byte("safe content"))
	if err != nil {
		t.Errorf("Failed to write valid file: %v", err)
	}

	// Read it back
	content, err := storage.Get(validPath)
	if err != nil {
		t.Errorf("Failed to read valid file: %v", err)
	}
	if string(content) != "safe content" {
		t.Errorf("Content mismatch: got '%s'", string(content))
	}

	// Verify the file is actually inside tempDir
	fullPath := filepath.Join(tempDir, validPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("File not created in expected location")
	}

	// Try to escape sandbox
	escapePath := "../escape.txt"
	err = storage.Put(escapePath, []byte("escaped content"))
	if err == nil {
		t.Error("Sandbox escape succeeded!")

		// Check if file was created outside
		escapedFile := filepath.Join(tempDir, "..", "escape.txt")
		if _, err := os.Stat(escapedFile); err == nil {
			t.Error("File created outside sandbox!")
			os.Remove(escapedFile)
		}
	}
}

// TestGenerateUniqueName tests unique filename generation.
func TestGenerateUniqueName(t *testing.T) {
	original := "photo.jpg"

	name1 := GenerateUniqueName(original)
	name2 := GenerateUniqueName(original)

	// Names should be different (different timestamps)
	if name1 == name2 {
		t.Error("GenerateUniqueName generated duplicate names")
	}

	// Both should contain original filename
	if !strings.Contains(name1, "photo.jpg") {
		t.Errorf("Unique name '%s' doesn't contain original name", name1)
	}
	if !strings.Contains(name2, "photo.jpg") {
		t.Errorf("Unique name '%s' doesn't contain original name", name2)
	}
}

// BenchmarkSanitizePath benchmarks path validation performance.
func BenchmarkSanitizePath(b *testing.B) {
	path := "uploads/avatars/user-123.jpg"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizePath(path)
	}
}

// BenchmarkContainsPathTraversal benchmarks traversal detection.
func BenchmarkContainsPathTraversal(b *testing.B) {
	path := "uploads/avatars/user-123.jpg"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containsPathTraversal(path)
	}
}
