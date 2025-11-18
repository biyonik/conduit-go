// -----------------------------------------------------------------------------
// Sync Queue Driver
// -----------------------------------------------------------------------------
// Synchronous queue implementation (testing/development).
//
// Bu driver job'ları hemen çalıştırır, queue'lamaz.
// Test ve development ortamları için uygundur.
// -----------------------------------------------------------------------------

package queue

import (
	"log"
	"time"
)

// SyncQueue, synchronous queue implementation.
type SyncQueue struct {
	logger *log.Logger
}

// NewSyncQueue, yeni bir Sync queue instance oluşturur.
func NewSyncQueue(logger *log.Logger) *SyncQueue {
	return &SyncQueue{
		logger: logger,
	}
}

// Push, job'ı hemen çalıştırır.
func (s *SyncQueue) Push(job Job, queue string) error {
	s.logger.Printf("⚡ Sync executing job: %s (queue: %s)", job.GetID(), queue)

	err := job.Handle()
	if err != nil {
		s.logger.Printf("❌ Job failed: %s (error: %v)", job.GetID(), err)
		job.Failed(err)
		return err
	}

	s.logger.Printf("✅ Job completed: %s", job.GetID())
	return nil
}

// Later, job'ı gecikme ile çalıştırır.
func (s *SyncQueue) Later(delay time.Duration, job Job, queue string) error {
	if delay > 0 {
		s.logger.Printf("⏱️  Waiting %v before executing job: %s", delay, job.GetID())
		time.Sleep(delay)
	}
	return s.Push(job, queue)
}

// Pop, sync queue'da kullanılmaz.
func (s *SyncQueue) Pop(queue string) (Job, error) {
	return nil, nil
}

// Delete, sync queue'da kullanılmaz.
func (s *SyncQueue) Delete(queue string, job Job) error {
	return nil
}

// Release, sync queue'da kullanılmaz.
func (s *SyncQueue) Release(queue string, job Job, delay time.Duration) error {
	return nil
}

// Size, sync queue'da her zaman 0 döner.
func (s *SyncQueue) Size(queue string) (int64, error) {
	return 0, nil
}
