// -----------------------------------------------------------------------------
// Container Helper Functions
// -----------------------------------------------------------------------------
// This file provides convenient helper functions to reduce boilerplate code
// when retrieving common dependencies from the DI container.
//
// These helpers eliminate repetitive reflection code like:
//   c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
//
// And replace it with simple calls like:
//   container.GetLogger(c)
// -----------------------------------------------------------------------------

package container

import (
	"database/sql"
	"log"
	"reflect"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/pkg/cache"
	"github.com/biyonik/conduit-go/pkg/database"
	"github.com/biyonik/conduit-go/pkg/queue"
)

// GetLogger retrieves the logger from the container.
//
// Example:
//
//	logger := container.GetLogger(c)
func GetLogger(c *Container) *log.Logger {
	return c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
}

// GetDatabase retrieves the database connection from the container.
//
// Example:
//
//	db := container.GetDatabase(c)
func GetDatabase(c *Container) *sql.DB {
	return c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
}

// GetGrammar retrieves the SQL grammar from the container.
//
// Example:
//
//	grammar := container.GetGrammar(c)
func GetGrammar(c *Container) database.Grammar {
	grammarType := reflect.TypeOf((*database.Grammar)(nil)).Elem()
	return c.MustGet(grammarType).(database.Grammar)
}

// GetConfig retrieves the application config from the container.
//
// Example:
//
//	cfg := container.GetConfig(c)
//	env := cfg.App.Env
func GetConfig(c *Container) *config.Config {
	return c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
}

// GetCache retrieves the cache driver from the container.
//
// Example:
//
//	cache := container.GetCache(c)
//	cache.Set("key", "value", 5*time.Minute)
func GetCache(c *Container) cache.Cache {
	cacheType := reflect.TypeOf((*cache.Cache)(nil)).Elem()
	return c.MustGet(cacheType).(cache.Cache)
}

// GetQueue retrieves the queue driver from the container.
//
// Example:
//
//	q := container.GetQueue(c)
//	q.Push(job, "default")
func GetQueue(c *Container) queue.Queue {
	queueType := reflect.TypeOf((*queue.Queue)(nil)).Elem()
	return c.MustGet(queueType).(queue.Queue)
}

// GetDatabaseAndGrammar is a convenience function that retrieves both
// the database connection and grammar in a single call.
//
// This is useful for initializing repositories which typically need both.
//
// Example:
//
//	db, grammar := container.GetDatabaseAndGrammar(c)
//	userRepo := models.NewUserRepository(db, grammar)
func GetDatabaseAndGrammar(c *Container) (*sql.DB, database.Grammar) {
	db := GetDatabase(c)
	grammar := GetGrammar(c)
	return db, grammar
}
