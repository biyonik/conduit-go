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
	"log"
	"net/http"
	"reflect"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/pkg/container"
)

// %s handles requests for the %s resource.
type %s struct {
	Logger *log.Logger
	// TODO: Add additional dependencies here (e.g., repositories, services)
}

// New%s creates a new %s instance using dependency injection.
func New%s(c *container.Container) (*%s, error) {
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)

	return &%s{
		Logger: logger,
		// TODO: Initialize additional dependencies from container
	}, nil
}

// Handle is a sample handler method.
//
// Example usage:
//   router.GET("/path", controller.Handle)
func (c *%s) Handle(w http.ResponseWriter, r *conduitReq.Request) {
	c.Logger.Println("📝 Handling request...")

	// TODO: Implement handler logic

	response := map[string]interface{}{
		"message": "success",
	}

	conduitRes.Success(w, 200, response, nil)
}
`, name, name, name, name, name, name, name, name, name)
}

func generateResourceController(name string, api bool) string {
	return fmt.Sprintf(`package controllers

import (
	"log"
	"net/http"
	"reflect"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/pkg/container"
)

// %s handles CRUD operations for the resource.
type %s struct {
	Logger *log.Logger
	// TODO: Add repositories and services (e.g., ResourceRepository)
}

// New%s creates a new %s instance using dependency injection.
func New%s(c *container.Container) (*%s, error) {
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)

	return &%s{
		Logger: logger,
		// TODO: Initialize repositories from container
	}, nil
}

// Index displays a listing of the resource.
//
// HTTP Method: GET
// Route: /resource
func (c *%s) Index(w http.ResponseWriter, r *conduitReq.Request) {
	c.Logger.Println("📋 Fetching all resources...")

	// TODO: Fetch all resources from repository
	// resources, err := c.ResourceRepository.GetAll(page, perPage)

	response := map[string]interface{}{
		"data": []interface{}{},
	}

	conduitRes.Success(w, 200, response, nil)
}

// Show displays the specified resource.
//
// HTTP Method: GET
// Route: /resource/{id}
func (c *%s) Show(w http.ResponseWriter, r *conduitReq.Request) {
	// TODO: Get ID from route parameters
	// id := r.RouteParam("id")

	c.Logger.Println("🔍 Fetching resource by ID...")

	// TODO: Fetch resource by ID
	// resource, err := c.ResourceRepository.FindByID(id)
	// if err == sql.ErrNoRows {
	//     conduitRes.Error(w, 404, "Resource not found")
	//     return
	// }

	response := map[string]interface{}{
		"data": nil,
	}

	conduitRes.Success(w, 200, response, nil)
}

// Store stores a newly created resource.
//
// HTTP Method: POST
// Route: /resource
func (c *%s) Store(w http.ResponseWriter, r *conduitReq.Request) {
	c.Logger.Println("➕ Creating new resource...")

	// TODO: Parse request body
	// var reqData struct { ... }
	// if err := r.ParseJSON(&reqData); err != nil {
	//     conduitRes.Error(w, 400, "Invalid JSON format")
	//     return
	// }

	// TODO: Validate request
	// TODO: Create resource in repository

	response := map[string]interface{}{
		"message": "Resource created successfully",
	}

	conduitRes.Success(w, 201, response, nil)
}

// Update updates the specified resource.
//
// HTTP Method: PUT/PATCH
// Route: /resource/{id}
func (c *%s) Update(w http.ResponseWriter, r *conduitReq.Request) {
	// TODO: Get ID from route parameters
	// id := r.RouteParam("id")

	c.Logger.Println("✏️  Updating resource...")

	// TODO: Parse request body
	// TODO: Validate request
	// TODO: Update resource in repository

	response := map[string]interface{}{
		"message": "Resource updated successfully",
	}

	conduitRes.Success(w, 200, response, nil)
}

// Destroy removes the specified resource.
//
// HTTP Method: DELETE
// Route: /resource/{id}
func (c *%s) Destroy(w http.ResponseWriter, r *conduitReq.Request) {
	// TODO: Get ID from route parameters
	// id := r.RouteParam("id")

	c.Logger.Println("🗑️  Deleting resource...")

	// TODO: Delete resource from repository (soft delete)
	// err := c.ResourceRepository.Delete(id)

	response := map[string]interface{}{
		"message": "Resource deleted successfully",
	}

	conduitRes.Success(w, 200, response, nil)
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
	"database/sql"

	"github.com/biyonik/conduit-go/pkg/database"
)

// %s model represents a %s record.
type %s struct {
	BaseModel
	// TODO: Add model fields here
	// Example:
	// Name  string ` + "`json:\"name\" db:\"name\"`" + `
	// Email string ` + "`json:\"email\" db:\"email\"`" + `
}

// %sRepository handles database operations for %s.
type %sRepository struct {
	db      *sql.DB
	grammar database.Grammar
}

// New%sRepository creates a new %sRepository instance.
func New%sRepository(db *sql.DB, grammar database.Grammar) *%sRepository {
	return &%sRepository{
		db:      db,
		grammar: grammar,
	}
}

// newBuilder creates a new query builder for this repository.
func (r *%sRepository) newBuilder() *database.QueryBuilder {
	return database.NewBuilder(r.db, r.grammar)
}

// FindByID finds a %s by ID.
func (r *%sRepository) FindByID(id int64) (*%s, error) {
	var record %s
	err := r.newBuilder().
		Table("%s").
		Where("id", "=", id).
		Where("deleted_at", "IS", nil). // Soft delete check
		First(&record)

	if err != nil {
		return nil, err
	}

	return &record, nil
}

// GetAll retrieves all %s records with pagination.
func (r *%sRepository) GetAll(page, perPage int) ([]%s, error) {
	var records []%s

	offset := (page - 1) * perPage

	err := r.newBuilder().
		Table("%s").
		Where("deleted_at", "IS", nil).
		OrderBy("created_at", "DESC").
		Limit(perPage).
		Offset(offset).
		Get(&records)

	if err != nil {
		return nil, err
	}

	return records, nil
}

// Create creates a new %s record.
func (r *%sRepository) Create(record *%s) (int64, error) {
	record.Initialize() // Sets CreatedAt and UpdatedAt

	result, err := r.newBuilder().ExecInsert(map[string]interface{}{
		// TODO: Add fields to insert
		// "name":       record.Name,
		"created_at": record.CreatedAt,
		"updated_at": record.UpdatedAt,
	})

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Update updates an existing %s record.
func (r *%sRepository) Update(record *%s) error {
	record.Touch() // Updates UpdatedAt

	data := map[string]interface{}{
		// TODO: Add fields to update
		// "name":       record.Name,
		"updated_at": record.UpdatedAt,
	}

	_, err := r.newBuilder().
		Table("%s").
		Where("id", "=", record.ID).
		ExecUpdate(data)

	return err
}

// Delete soft deletes a %s record.
func (r *%sRepository) Delete(id int64) error {
	_, err := r.newBuilder().
		Table("%s").
		Where("id", "=", id).
		ExecUpdate(map[string]interface{}{
			"deleted_at": r.newBuilder().Now(),
		})

	return err
}
`,
		name, name, name,
		name, name, name,
		name, name, name, name, name,
		name,
		name, name, name, name, toSnakeCase(pluralize(name)),
		pluralize(name), name, name, name, toSnakeCase(pluralize(name)),
		name, name, name,
		name, name, name,
		toSnakeCase(pluralize(name)),
		name, name, name, toSnakeCase(pluralize(name)))

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

	middlewareName := strings.TrimSuffix(name, "Middleware")

	content := fmt.Sprintf(`package middleware

import (
	"net/http"

	"github.com/biyonik/conduit-go/internal/http/response"
)

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// %s is a middleware that...
// TODO: Describe what this middleware does
//
// Example usage:
//   r.Use(middleware.%s())
//   r.GET("/path", handler).Middleware(middleware.%s())
func %s() Middleware {
	return %sWithConfig(nil)
}

// %sConfig holds configuration for %s middleware.
type %sConfig struct {
	// TODO: Add configuration fields here
	// Example:
	// Enabled bool
	// Timeout time.Duration
}

// Default%sConfig returns the default configuration.
func Default%sConfig() *%sConfig {
	return &%sConfig{
		// TODO: Set default values
	}
}

// %sWithConfig returns the middleware with custom configuration.
func %sWithConfig(config *%sConfig) Middleware {
	if config == nil {
		config = Default%sConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement middleware logic before request

			// Example: Check some condition
			// if !someCondition {
			//     response.Error(w, http.StatusForbidden, "Access denied")
			//     return
			// }

			// Call next handler
			next.ServeHTTP(w, r)

			// TODO: Implement middleware logic after request (if needed)
		})
	}
}
`, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName, middlewareName)

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
	"encoding/json"
	"log"

	"github.com/biyonik/conduit-go/pkg/queue"
)

// %s represents a queued job.
type %s struct {
	queue.BaseJob
	// TODO: Add job properties
	// Example:
	// UserID int64  ` + "`json:\"user_id\"`" + `
	// Email  string ` + "`json:\"email\"`" + `

	// Dependencies (not serialized - inject when executing)
	// Mailer mail.Mailer ` + "`json:\"-\"`" + `
}

// New%s creates a new %s instance.
func New%s() *%s {
	return &%s{
		BaseJob: queue.BaseJob{
			MaxAttempts: 3, // Retry up to 3 times on failure
		},
	}
}

// Handle executes the job.
func (j *%s) Handle() error {
	log.Printf("⚙️  Executing %s job...")

	// TODO: Implement job logic
	// Example: Send email, process image, update database, etc.

	log.Printf("✅ %s job completed successfully")
	return nil
}

// Failed is called when the job fails after all retry attempts.
func (j *%s) Failed(err error) error {
	log.Printf("❌ %s job failed: %%s (error: %%v)", j.ID, err)

	// TODO: Handle job failure
	// Examples:
	// - Log to database
	// - Send notification to admin
	// - Update status in monitoring system

	return nil
}

// GetPayload serializes the job to JSON.
func (j *%s) GetPayload() ([]byte, error) {
	return json.Marshal(j)
}

// SetPayload deserializes the job from JSON.
func (j *%s) SetPayload(data []byte) error {
	return json.Unmarshal(data, j)
}
`, name, name, name, name, name, name, name, name, name, name, name, name, name, name)

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
