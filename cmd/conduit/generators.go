// -----------------------------------------------------------------------------
// Code Generators - Laravel-Inspired Code Generation
// -----------------------------------------------------------------------------
// Bu dosya, çeşitli kod yapılarını (controller, model, middleware, vb.)
// otomatik olarak oluşturan generator fonksiyonlarını içerir.
// -----------------------------------------------------------------------------

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Controller Generator
// -----------------------------------------------------------------------------

func generateController(name string, resource bool, api bool) {
	// Ensure Controller suffix
	if !strings.HasSuffix(name, "Controller") {
		name = name + "Controller"
	}

	dir := "internal/controllers"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(dir, toSnakeCase(name)+".go")

	var content string
	if resource {
		content = generateResourceController(name, api)
	} else {
		content = generateBasicController(name)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Controller created: %s\n", filename)
}

func generateBasicController(name string) string {
	return fmt.Sprintf(`package controllers

import (
	"net/http"
)

// %s handles requests for the %s resource.
type %s struct {
	// Add dependencies here (e.g., services, repositories)
}

// New%s creates a new %s instance.
func New%s() *%s {
	return &%s{}
}

// Handle is a sample handler method.
//
// Örnek:
//   router.HandleFunc("/path", controller.Handle)
func (c *%s) Handle(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement handler logic
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"message\": \"success\"}"))
}
`, name, name, name, name, name, name, name, name, name)
}

func generateResourceController(name string, api bool) string {
	return fmt.Sprintf(`package controllers

import (
	"encoding/json"
	"net/http"
)

// %s handles CRUD operations for the resource.
type %s struct {
	// Add dependencies here (e.g., services, repositories)
}

// New%s creates a new %s instance.
func New%s() *%s {
	return &%s{}
}

// Index displays a listing of the resource.
//
// HTTP Method: GET
// Route: /resource
func (c *%s) Index(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch all resources
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": []interface{}{},
	})
}

// Show displays the specified resource.
//
// HTTP Method: GET
// Route: /resource/{id}
func (c *%s) Show(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch resource by ID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": nil,
	})
}

// Store stores a newly created resource.
//
// HTTP Method: POST
// Route: /resource
func (c *%s) Store(w http.ResponseWriter, r *http.Request) {
	// TODO: Validate and create resource
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Resource created successfully",
	})
}

// Update updates the specified resource.
//
// HTTP Method: PUT/PATCH
// Route: /resource/{id}
func (c *%s) Update(w http.ResponseWriter, r *http.Request) {
	// TODO: Validate and update resource
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Resource updated successfully",
	})
}

// Destroy removes the specified resource.
//
// HTTP Method: DELETE
// Route: /resource/{id}
func (c *%s) Destroy(w http.ResponseWriter, r *http.Request) {
	// TODO: Delete resource
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Resource deleted successfully",
	})
}
`, name, name, name, name, name, name, name, name, name, name, name, name)
}

// -----------------------------------------------------------------------------
// Model Generator
// -----------------------------------------------------------------------------

func generateModel(name string, withMigration bool) {
	dir := "internal/models"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(dir, toSnakeCase(name)+".go")

	content := fmt.Sprintf(`package models

import (
	"time"
)

// %s model represents a %s record.
type %s struct {
	ID        uint       ` + "`json:\"id\" db:\"id\"`" + `
	CreatedAt time.Time  ` + "`json:\"created_at\" db:\"created_at\"`" + `
	UpdatedAt time.Time  ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
	DeletedAt *time.Time ` + "`json:\"deleted_at,omitempty\" db:\"deleted_at\"`" + ` // Soft delete

	// TODO: Add model fields here
}

// TableName returns the table name for this model.
func (m *%s) TableName() string {
	return "%s"
}

// BeforeCreate is called before creating a new record.
func (m *%s) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate is called before updating a record.
func (m *%s) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
`, name, name, name, name, toSnakeCase(pluralize(name)), name, name)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Model created: %s\n", filename)

	if withMigration {
		generateMigration("create_"+toSnakeCase(pluralize(name))+"_table", "")
	}
}

// -----------------------------------------------------------------------------
// Middleware Generator
// -----------------------------------------------------------------------------

func generateMiddleware(name string) {
	// Ensure Middleware suffix
	if !strings.HasSuffix(name, "Middleware") {
		name = name + "Middleware"
	}

	dir := "internal/middleware"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(dir, toSnakeCase(name)+".go")

	content := fmt.Sprintf(`package middleware

import (
	"net/http"
)

// %s is a middleware that...
// TODO: Describe what this middleware does
type %s struct {
	// Add dependencies here
}

// New%s creates a new %s instance.
func New%s() *%s {
	return &%s{}
}

// Handle wraps an http.Handler and applies middleware logic.
func (m *%s) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement middleware logic before request

		// Call next handler
		next.ServeHTTP(w, r)

		// TODO: Implement middleware logic after request
	})
}

// Func wraps an http.HandlerFunc and applies middleware logic.
func (m *%s) Func(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement middleware logic before request

		// Call next handler
		next(w, r)

		// TODO: Implement middleware logic after request
	}
}
`, name, name, name, name, name, name, name, name, name)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Middleware created: %s\n", filename)
}

// -----------------------------------------------------------------------------
// Job Generator
// -----------------------------------------------------------------------------

func generateJob(name string) {
	// Ensure Job suffix
	if !strings.HasSuffix(name, "Job") {
		name = name + "Job"
	}

	dir := "internal/jobs"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(dir, toSnakeCase(name)+".go")

	content := fmt.Sprintf(`package jobs

import (
	"context"
	"fmt"
)

// %s represents a queued job.
type %s struct {
	// TODO: Add job properties
}

// New%s creates a new %s instance.
func New%s() *%s {
	return &%s{}
}

// Handle executes the job.
func (j *%s) Handle(ctx context.Context) error {
	// TODO: Implement job logic
	fmt.Println("Executing %s...")

	// Example: Send email, process image, etc.

	return nil
}

// Failed is called when the job fails.
func (j *%s) Failed(err error) {
	// TODO: Handle job failure (log, notify, etc.)
	fmt.Printf("❌ %s failed: %%v\n", err)
}

// MaxRetries returns the maximum number of retry attempts.
func (j *%s) MaxRetries() int {
	return 3
}

// Timeout returns the job execution timeout.
func (j *%s) Timeout() int {
	return 60 // seconds
}
`, name, name, name, name, name, name, name, name, name, name, name, name, name)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Job created: %s\n", filename)
}

// -----------------------------------------------------------------------------
// Event Generator
// -----------------------------------------------------------------------------

func generateEvent(name string) {
	dir := "internal/events"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(dir, toSnakeCase(name)+".go")

	content := fmt.Sprintf(`package events

import (
	"time"

	"github.com/biyonik/conduit-go/pkg/events"
)

// %s is an event that occurs when...
// TODO: Describe when this event is triggered
type %s struct {
	events.BaseEvent

	// TODO: Add event-specific data
	// Example:
	// UserID uint
	// Email  string
}

// New%s creates a new %s instance.
func New%s( /* TODO: Add parameters */ ) *%s {
	event := &%s{
		BaseEvent: *events.NewBaseEvent("%s", nil),
	}

	// TODO: Set event payload
	// event.SetPayload(yourData)

	return event
}
`, name, name, name, name, name, name, name, toSnakeCase(name))

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Event created: %s\n", filename)
}

// -----------------------------------------------------------------------------
// Listener Generator
// -----------------------------------------------------------------------------

func generateListener(name string, eventName string) {
	// Ensure Listener suffix
	if !strings.HasSuffix(name, "Listener") {
		name = name + "Listener"
	}

	dir := "internal/listeners"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(dir, toSnakeCase(name)+".go")

	eventImport := ""
	eventType := "events.Event"
	if eventName != "" {
		eventImport = fmt.Sprintf("\n\t\"github.com/biyonik/conduit-go/internal/events\"")
		eventType = fmt.Sprintf("*events.%s", eventName)
	}

	content := fmt.Sprintf(`package listeners

import (
	"fmt"%s
	pkgevents "github.com/biyonik/conduit-go/pkg/events"
)

// %s handles the %s event.
type %s struct {
	// TODO: Add dependencies (e.g., mailer, logger, repository)
}

// New%s creates a new %s instance.
func New%s() *%s {
	return &%s{}
}

// Handle processes the event.
func (l *%s) Handle(event pkgevents.Event) error {
	// TODO: Type assert to specific event if needed
	// e, ok := event.Payload().(%s)
	// if !ok {
	//     return fmt.Errorf("invalid event type")
	// }

	fmt.Printf("Handling event: %%s\n", event.Name())

	// TODO: Implement listener logic
	// Example: Send email, update database, trigger another event, etc.

	return nil
}
`, eventImport, name, eventName, name, name, name, name, name, name, name, eventType)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Listener created: %s\n", filename)
}

// -----------------------------------------------------------------------------
// Migration Generator
// -----------------------------------------------------------------------------

func generateMigration(name string, table string) string {
	dir := "database/migrations"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	// Generate timestamp-based filename
	timestamp := time.Now().Format("2006_01_02_150405")
	filename := filepath.Join(dir, fmt.Sprintf("%s_%s.go", timestamp, name))

	structName := toPascalCase(name)

	content := fmt.Sprintf(`package migrations

import (
	"github.com/biyonik/conduit-go/pkg/database/migration"
)

// %s migration
type %s struct{}

// Up runs the migration.
func (m *%s) Up(migrator *migration.Migrator) error {
	// TODO: Implement migration logic
	// Example:
	// return migrator.CreateTable("%s", func(t *migration.Blueprint) {
	//     t.ID()
	//     t.String("name", 255)
	//     t.String("email", 255).Unique()
	//     t.Timestamps()
	// })

	return nil
}

// Down reverses the migration.
func (m *%s) Down(migrator *migration.Migrator) error {
	// TODO: Implement rollback logic
	// Example:
	// return migrator.DropTable("%s")

	return nil
}
`, structName, structName, structName, table, structName, table)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create migration file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Migration created: %s\n", filename)
	return filename
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// toSnakeCase converts a string to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}

// pluralize adds an 's' to make a word plural (simple version).
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}
