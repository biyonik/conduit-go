// -----------------------------------------------------------------------------
// Testing Helpers - Laravel-Inspired Testing Utilities
// -----------------------------------------------------------------------------
// Bu package, test yazımını kolaylaştıran helper fonksiyonlar sağlar.
//
// Özellikler:
// - HTTP testing helpers (request/response assertions)
// - Database testing traits (RefreshDatabase, DatabaseTransactions)
// - Mock helpers (cache, queue, mail, events, storage)
// - Factory pattern for test data
// - Custom assertions
//
// Kullanım:
//
//	func TestUserCreation(t *testing.T) {
//	    // Setup
//	    db := testing.SetupTestDatabase(t)
//	    defer testing.TeardownTestDatabase(t, db)
//
//	    // Test
//	    user := testing.CreateUser(db, "test@example.com")
//
//	    // Assert
//	    testing.AssertNotNil(t, user)
//	    testing.AssertEquals(t, "test@example.com", user.Email)
//	}
// -----------------------------------------------------------------------------

package testing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// HTTP Testing Helpers
// -----------------------------------------------------------------------------

// TestRequest represents an HTTP test request builder.
type TestRequest struct {
	method  string
	url     string
	body    io.Reader
	headers map[string]string
}

// NewTestRequest creates a new test request builder.
func NewTestRequest(method, url string) *TestRequest {
	return &TestRequest{
		method:  method,
		url:     url,
		headers: make(map[string]string),
	}
}

// WithJSON sets the request body as JSON.
func (r *TestRequest) WithJSON(data interface{}) *TestRequest {
	jsonData, _ := json.Marshal(data)
	r.body = strings.NewReader(string(jsonData))
	r.headers["Content-Type"] = "application/json"
	return r
}

// WithHeader adds a header to the request.
func (r *TestRequest) WithHeader(key, value string) *TestRequest {
	r.headers[key] = value
	return r
}

// Send executes the test request.
func (r *TestRequest) Send(handler http.Handler) *TestResponse {
	req := httptest.NewRequest(r.method, r.url, r.body)
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return &TestResponse{
		recorder: w,
	}
}

// TestResponse represents an HTTP test response.
type TestResponse struct {
	recorder *httptest.ResponseRecorder
}

// AssertStatus asserts the response status code.
func (r *TestResponse) AssertStatus(t *testing.T, expectedStatus int) *TestResponse {
	if r.recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, r.recorder.Code)
	}
	return r
}

// AssertJSON asserts the response contains JSON.
func (r *TestResponse) AssertJSON(t *testing.T) *TestResponse {
	contentType := r.recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON response, got %s", contentType)
	}
	return r
}

// AssertJSONPath asserts a value at a JSON path.
func (r *TestResponse) AssertJSONPath(t *testing.T, path string, expected interface{}) *TestResponse {
	var data map[string]interface{}
	if err := json.Unmarshal(r.recorder.Body.Bytes(), &data); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
		return r
	}

	// Simple path traversal (only supports single-level for now)
	actual, ok := data[path]
	if !ok {
		t.Errorf("JSON path '%s' not found", path)
		return r
	}

	if actual != expected {
		t.Errorf("Expected '%v' at path '%s', got '%v'", expected, path, actual)
	}

	return r
}

// GetJSON parses the response body as JSON.
func (r *TestResponse) GetJSON(t *testing.T) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal(r.recorder.Body.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	return data
}

// GetBody returns the response body as string.
func (r *TestResponse) GetBody() string {
	return r.recorder.Body.String()
}

// -----------------------------------------------------------------------------
// Database Testing Helpers
// -----------------------------------------------------------------------------

// RefreshDatabase tears down and rebuilds the database for each test.
//
// Kullanım:
//
//	func TestWithFreshDatabase(t *testing.T) {
//	    db := RefreshDatabase(t)
//	    defer db.Close()
//
//	    // Test with clean database
//	}
func RefreshDatabase(t *testing.T) *sql.DB {
	// TODO: Implement database refresh logic
	// 1. Drop all tables
	// 2. Run migrations
	// 3. Return DB connection

	t.Log("RefreshDatabase: Database refreshed (placeholder)")
	return nil
}

// DatabaseTransaction runs a test inside a transaction and rolls back.
//
// Kullanım:
//
//	func TestInTransaction(t *testing.T) {
//	    DatabaseTransaction(t, func(tx *sql.Tx) {
//	        // Test code here
//	        // Automatically rolled back after test
//	    })
//	}
func DatabaseTransaction(t *testing.T, callback func(*sql.Tx)) {
	// TODO: Implement transaction wrapper
	// 1. Begin transaction
	// 2. Run callback
	// 3. Rollback transaction

	t.Log("DatabaseTransaction: Running test in transaction (placeholder)")
}

// -----------------------------------------------------------------------------
// Mock Helpers
// -----------------------------------------------------------------------------

// MockCache represents a mock cache for testing.
type MockCache struct {
	data map[string]interface{}
}

// NewMockCache creates a new mock cache.
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
	}
}

// Set sets a value in the mock cache.
func (m *MockCache) Set(key string, value interface{}) {
	m.data[key] = value
}

// Get gets a value from the mock cache.
func (m *MockCache) Get(key string) (interface{}, bool) {
	value, ok := m.data[key]
	return value, ok
}

// Clear clears the mock cache.
func (m *MockCache) Clear() {
	m.data = make(map[string]interface{})
}

// MockQueue represents a mock queue for testing.
type MockQueue struct {
	jobs []interface{}
}

// NewMockQueue creates a new mock queue.
func NewMockQueue() *MockQueue {
	return &MockQueue{
		jobs: make([]interface{}, 0),
	}
}

// Push adds a job to the mock queue.
func (m *MockQueue) Push(job interface{}) {
	m.jobs = append(m.jobs, job)
}

// AssertPushed asserts that a job was pushed to the queue.
func (m *MockQueue) AssertPushed(t *testing.T, jobType interface{}) {
	for _, job := range m.jobs {
		if fmt.Sprintf("%T", job) == fmt.Sprintf("%T", jobType) {
			return // Found
		}
	}
	t.Errorf("Job of type %T was not pushed to queue", jobType)
}

// Count returns the number of jobs in the queue.
func (m *MockQueue) Count() int {
	return len(m.jobs)
}

// -----------------------------------------------------------------------------
// Assertion Helpers
// -----------------------------------------------------------------------------

// AssertEquals asserts two values are equal.
func AssertEquals(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEquals asserts two values are not equal.
func AssertNotEquals(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected == actual {
		t.Errorf("Expected values to be different, both are %v", expected)
	}
}

// AssertNil asserts a value is nil.
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Errorf("Expected nil, got %v", value)
	}
}

// AssertNotNil asserts a value is not nil.
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Error("Expected non-nil value, got nil")
	}
}

// AssertTrue asserts a condition is true.
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Errorf("Assertion failed: %s", message)
	}
}

// AssertFalse asserts a condition is false.
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Errorf("Assertion failed: %s", message)
	}
}

// AssertContains asserts a string contains a substring.
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected '%s' to contain '%s'", haystack, needle)
	}
}

// AssertNotContains asserts a string does not contain a substring.
func AssertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("Expected '%s' to NOT contain '%s'", haystack, needle)
	}
}

// -----------------------------------------------------------------------------
// Factory Pattern Helpers
// -----------------------------------------------------------------------------

// Factory represents a test data factory.
type Factory struct {
	defaults map[string]interface{}
}

// NewFactory creates a new factory with default values.
func NewFactory(defaults map[string]interface{}) *Factory {
	return &Factory{
		defaults: defaults,
	}
}

// Make creates a new instance with optional overrides.
func (f *Factory) Make(overrides map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy defaults
	for k, v := range f.defaults {
		result[k] = v
	}

	// Apply overrides
	for k, v := range overrides {
		result[k] = v
	}

	return result
}

// UserFactory creates a user factory with default values.
func UserFactory() *Factory {
	return NewFactory(map[string]interface{}{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "password123",
		"active":   true,
	})
}

// PostFactory creates a post factory with default values.
func PostFactory() *Factory {
	return NewFactory(map[string]interface{}{
		"title":   "Test Post",
		"content": "This is a test post content.",
		"status":  "published",
	})
}
