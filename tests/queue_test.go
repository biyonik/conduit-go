// -----------------------------------------------------------------------------
// Queue System Tests
// -----------------------------------------------------------------------------
// Queue sisteminin test dosyası.
// -----------------------------------------------------------------------------

package tests

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/biyonik/conduit-go/internal/jobs"
	"github.com/biyonik/conduit-go/pkg/queue"
)

func TestSyncQueue(t *testing.T) {
	logger := log.New(os.Stdout, "[QueueTest] ", log.Ldate|log.Ltime)

	// Sync queue oluştur
	syncQueue := queue.NewSyncQueue(logger)

	// Email job oluştur
	emailJob := jobs.NewSendEmailJob(
		"test@example.com",
		"Test Email",
		"This is a test email from queue system",
	)

	// Job'ı register et
	queue.RegisterJob("*jobs.SendEmailJob", func() queue.Job {
		return &jobs.SendEmailJob{}
	})

	// Job'ı push et (hemen çalışmalı)
	err := syncQueue.Push(emailJob, "emails")
	if err != nil {
		t.Errorf("Push hatası: %v", err)
	}

	// Later testi (5 saniye gecikme)
	t.Log("Testing Later() with 2 second delay...")
	uploadJob := jobs.NewProcessUploadJob(
		"/tmp/test.jpg",
		123,
		"image",
	)

	queue.RegisterJob("*jobs.ProcessUploadJob", func() queue.Job {
		return &jobs.ProcessUploadJob{}
	})

	startTime := time.Now()
	err = syncQueue.Later(2*time.Second, uploadJob, "uploads")
	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("Later hatası: %v", err)
	}

	if elapsed < 2*time.Second {
		t.Errorf("Later gecikme çalışmadı: %v", elapsed)
	}

	t.Log("✅ Sync queue tests passed")
}

func TestJobSerialization(t *testing.T) {
	// Email job oluştur
	job := jobs.NewSendEmailJob(
		"user@example.com",
		"Welcome",
		"Welcome to Conduit-Go!",
	)

	// Serialize
	payload, err := job.GetPayload()
	if err != nil {
		t.Errorf("Serialization hatası: %v", err)
	}

	// Deserialize
	newJob := &jobs.SendEmailJob{}
	err = newJob.SetPayload(payload)
	if err != nil {
		t.Errorf("Deserialization hatası: %v", err)
	}

	// Karşılaştır
	if newJob.To != job.To {
		t.Errorf("To mismatch: %s != %s", newJob.To, job.To)
	}
	if newJob.Subject != job.Subject {
		t.Errorf("Subject mismatch: %s != %s", newJob.Subject, job.Subject)
	}
	if newJob.Body != job.Body {
		t.Errorf("Body mismatch: %s != %s", newJob.Body, job.Body)
	}

	t.Log("✅ Job serialization tests passed")
}

// Benchmark testi
func BenchmarkSyncQueue(b *testing.B) {
	logger := log.New(os.Stdout, "[Benchmark] ", log.Ldate|log.Ltime)
	syncQueue := queue.NewSyncQueue(logger)

	queue.RegisterJob("*jobs.SendEmailJob", func() queue.Job {
		return &jobs.SendEmailJob{}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := jobs.NewSendEmailJob(
			"bench@example.com",
			"Benchmark",
			"Benchmark test",
		)
		syncQueue.Push(job, "emails")
	}
}
