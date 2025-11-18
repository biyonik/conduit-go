// -----------------------------------------------------------------------------
// Queue Worker
// -----------------------------------------------------------------------------
// Job'ları kuyruktan çekip işleyen worker.
//
// Özellikler:
// - Multiple queue support
// - Graceful shutdown
// - Failed job handling
// - Retry mechanism
// - Concurrency control
//
// Kullanım:
//   worker := NewWorker(queue, logger)
//   worker.Work("emails", "notifications")
// -----------------------------------------------------------------------------

package queue

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Worker, queue job'larını işleyen yapı.
type Worker struct {
	queue      Queue
	logger     *log.Logger
	stopChan   chan struct{}
	wg         sync.WaitGroup
	maxRetries int
	retryDelay time.Duration
}

// NewWorker, yeni bir Worker instance oluşturur.
//
// Parametreler:
//   - queue: Queue driver (Redis, Database, vb.)
//   - logger: Log instance
//
// Döndürür:
//   - *Worker: Worker instance
//
// Örnek:
//
//	worker := NewWorker(redisQueue, logger)
//	worker.Work("emails")
func NewWorker(queue Queue, logger *log.Logger) *Worker {
	return &Worker{
		queue:      queue,
		logger:     logger,
		stopChan:   make(chan struct{}),
		maxRetries: 3,
		retryDelay: 90 * time.Second,
	}
}

// SetMaxRetries, maksimum retry sayısını ayarlar.
func (w *Worker) SetMaxRetries(max int) *Worker {
	w.maxRetries = max
	return w
}

// SetRetryDelay, retry gecikme süresini ayarlar.
func (w *Worker) SetRetryDelay(delay time.Duration) *Worker {
	w.retryDelay = delay
	return w
}

// Work, belirtilen queue'ları dinlemeye başlar.
//
// Bu fonksiyon blocking'dir, goroutine'de çalıştırılmalı.
//
// Parametreler:
//   - queues: Dinlenecek queue adları (variadic)
//
// Örnek:
//
//	go worker.Work("emails", "notifications", "default")
//
// Graceful Shutdown:
//
//	// SIGTERM/SIGINT ile graceful shutdown
//	quit := make(chan os.Signal, 1)
//	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
//	<-quit
//	worker.Stop()
func (w *Worker) Work(queues ...string) {
	if len(queues) == 0 {
		queues = []string{"default"}
	}

	w.logger.Println("\n" + strings.Repeat("=", 70))
	w.logger.Println("🚀 Queue Worker Started")
	w.logger.Println(strings.Repeat("=", 70))
	w.logger.Printf("📋 Queues: %v", queues)
	w.logger.Printf("🔄 Max Retries: %d", w.maxRetries)
	w.logger.Printf("⏱️  Retry Delay: %v", w.retryDelay)
	w.logger.Println(strings.Repeat("=", 70))

	// Her queue için bir worker goroutine başlat
	for _, queueName := range queues {
		w.wg.Add(1)
		go w.processQueue(queueName)
	}

	// Graceful shutdown signal handler
	w.handleShutdown()

	// Tüm worker'ların bitmesini bekle
	w.wg.Wait()
	w.logger.Println("✅ Queue Worker Stopped")
}

// processQueue, tek bir queue'yu işler.
func (w *Worker) processQueue(queueName string) {
	defer w.wg.Done()

	w.logger.Printf("✅ Worker started for queue: %s", queueName)

	for {
		select {
		case <-w.stopChan:
			w.logger.Printf("🛑 Worker stopping for queue: %s", queueName)
			return
		default:
			// Job çek
			job, err := w.queue.Pop(queueName)
			if err != nil {
				w.logger.Printf("❌ Job pop hatası [%s]: %v", queueName, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Queue boş
			if job == nil {
				continue
			}

			// Job'ı işle
			w.processJob(queueName, job)
		}
	}
}

// processJob, tek bir job'ı işler.
func (w *Worker) processJob(queueName string, job Job) {
	startTime := time.Now()

	w.logger.Printf("🔄 Processing job: %s (queue: %s, attempt: %d/%d)",
		job.GetID(), queueName, job.GetAttempts()+1, job.GetMaxAttempts())

	// Job'ı çalıştır
	err := job.Handle()

	// Başarılı
	if err == nil {
		elapsed := time.Since(startTime)
		w.logger.Printf("✅ Job completed: %s (queue: %s, duration: %v)",
			job.GetID(), queueName, elapsed)

		// Queue'dan sil
		if delErr := w.queue.Delete(queueName, job); delErr != nil {
			w.logger.Printf("⚠️  Job delete hatası: %v", delErr)
		}
		return
	}

	// Başarısız
	w.logger.Printf("❌ Job failed: %s (queue: %s, error: %v)",
		job.GetID(), queueName, err)

	// Max attempts kontrolü
	if job.GetAttempts()+1 >= job.GetMaxAttempts() {
		w.logger.Printf("⚠️  Job max attempts reached: %s (queue: %s)",
			job.GetID(), queueName)

		// Failed handler çağır
		if failErr := job.Failed(err); failErr != nil {
			w.logger.Printf("⚠️  Job failed handler hatası: %v", failErr)
		}

		// Queue'dan sil (failed queue'ya taşınacak)
		w.queue.Release(queueName, job, 0)
		return
	}

	// Retry için tekrar kuyruğa ekle
	w.logger.Printf("🔄 Job retrying: %s (queue: %s, next attempt: %d/%d)",
		job.GetID(), queueName, job.GetAttempts()+2, job.GetMaxAttempts())

	if relErr := w.queue.Release(queueName, job, w.retryDelay); relErr != nil {
		w.logger.Printf("❌ Job release hatası: %v", relErr)
	}
}

// Stop, worker'ı gracefully durdurur.
//
// Bu fonksiyon mevcut job'ların bitmesini bekler.
//
// Örnek:
//
//	worker.Stop()
func (w *Worker) Stop() {
	w.logger.Println("🛑 Stopping queue worker...")
	close(w.stopChan)
}

// handleShutdown, SIGTERM/SIGINT sinyallerini dinler.
func (w *Worker) handleShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		w.Stop()
	}()
}

// Stats, worker istatistiklerini döndürür.
func (w *Worker) Stats(queues ...string) map[string]interface{} {
	stats := make(map[string]interface{})

	for _, queueName := range queues {
		size, err := w.queue.Size(queueName)
		if err != nil {
			stats[queueName] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		stats[queueName] = map[string]interface{}{
			"size": size,
		}
	}

	return stats
}
