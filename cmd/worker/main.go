// -----------------------------------------------------------------------------
// Queue Worker Command
// -----------------------------------------------------------------------------
// Queue worker'ı başlatan CLI command.
//
// Kullanım:
//   go run cmd/worker/main.go
//   go run cmd/worker/main.go emails notifications
// -----------------------------------------------------------------------------

package main

import (
	"database/sql"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/biyonik/conduit-go/internal/config"
	"github.com/biyonik/conduit-go/internal/jobs"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
	"github.com/biyonik/conduit-go/pkg/queue"
)

func main() {
	// =========================================================================
	// 1. DEPENDENCY INJECTION CONTAINER
	// =========================================================================
	c := container.New()

	// Config servisi
	c.Register(func(c *container.Container) (*config.Config, error) {
		return config.Load(), nil
	})

	// Logger servisi
	c.Register(func(c *container.Container) (*log.Logger, error) {
		return log.New(os.Stdout, "[Worker] ", log.Ldate|log.Ltime|log.Lshortfile), nil
	})

	// Database bağlantısı
	c.Register(func(c *container.Container) (*sql.DB, error) {
		cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)
		db, err := database.Connect(cfg.DB.DSN)
		if err != nil {
			return nil, err
		}
		return db, nil
	})

	// =========================================================================
	// 2. REDIS CLIENT (Queue için)
	// =========================================================================
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	cfg := c.MustGet(reflect.TypeOf((*config.Config)(nil))).(*config.Config)

	var queueDriver queue.Queue

	if cfg.Queue.Driver == "redis" {
		redisConfig := &database.RedisConfig{
			Host:         cfg.Redis.Host,
			Port:         cfg.Redis.Port,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     10,
			MinIdleConns: 2,
			MaxRetries:   3,
		}

		redisClient, err := database.NewRedisClient(redisConfig, logger)
		if err != nil {
			logger.Printf("❌ Redis bağlantı hatası: %v", err)
			logger.Println("⚠️  Sync queue'e geçiliyor...")
			queueDriver = queue.NewSyncQueue(logger)
		} else {
			queueDriver = queue.NewRedisQueue(redisClient.Client(), logger, cfg.Cache.Prefix)
		}
	} else {
		queueDriver = queue.NewSyncQueue(logger)
	}

	// =========================================================================
	// 3. JOB REGISTRY
	// =========================================================================
	logger.Println("📋 Registering job types...")

	queue.RegisterJob("*jobs.SendEmailJob", func() queue.Job {
		return &jobs.SendEmailJob{}
	})

	queue.RegisterJob("*jobs.ProcessUploadJob", func() queue.Job {
		return &jobs.ProcessUploadJob{}
	})

	logger.Println("✅ Job types registered")

	// =========================================================================
	// 4. WORKER BAŞLAT
	// =========================================================================
	worker := queue.NewWorker(queueDriver, logger)
	worker.SetMaxRetries(cfg.Queue.MaxAttempts)
	worker.SetRetryDelay(time.Duration(cfg.Queue.RetryAfter) * time.Second)

	// CLI argument'lardan queue isimleri al
	queues := os.Args[1:]
	if len(queues) == 0 {
		queues = []string{cfg.Queue.Default}
	}

	// Worker'ı başlat (blocking)
	worker.Work(queues...)
}
