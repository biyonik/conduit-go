// -----------------------------------------------------------------------------
// Queue Interface
// -----------------------------------------------------------------------------
// Laravel-style job queue interface.
//
// Bu interface tüm queue driver'ların implement etmesi gereken metodları tanımlar.
// Driver'lar: Redis, Database, Sync
//
// Özellikler:
// - Push/Later (immediate/delayed dispatch)
// - Pop (job fetch)
// - Failed job handling
// - Retry mechanism
// -----------------------------------------------------------------------------

package queue

import (
	"time"
)

// Queue, tüm queue driver'ların implement etmesi gereken interface.
//
// Bu interface Laravel Queue pattern'ini takip eder.
// Her driver (Redis, Database, Sync) bu interface'i implement eder.
type Queue interface {
	// Push, job'ı hemen kuyruğa ekler.
	//
	// Parametreler:
	//   - job: Kuyruğa eklenecek job
	//   - queue: Kuyruk adı (örn: "emails", "default")
	//
	// Döndürür:
	//   - error: Push hatası
	//
	// Örnek:
	//   err := queue.Push(emailJob, "emails")
	Push(job Job, queue string) error

	// Later, job'ı belirli bir gecikme ile kuyruğa ekler.
	//
	// Parametreler:
	//   - delay: Gecikme süresi
	//   - job: Kuyruğa eklenecek job
	//   - queue: Kuyruk adı
	//
	// Döndürür:
	//   - error: Push hatası
	//
	// Örnek:
	//   err := queue.Later(5*time.Minute, emailJob, "emails")
	Later(delay time.Duration, job Job, queue string) error

	// Pop, kuyruktan bir job çeker.
	//
	// Parametreler:
	//   - queue: Kuyruk adı
	//
	// Döndürür:
	//   - Job: Çekilen job (nil ise kuyruk boş)
	//   - error: Pop hatası
	//
	// Örnek:
	//   job, err := queue.Pop("emails")
	Pop(queue string) (Job, error)

	// Delete, job'ı kuyruktan siler.
	//
	// Job başarıyla işlendiğinde çağrılır.
	//
	// Parametreler:
	//   - queue: Kuyruk adı
	//   - job: Silinecek job
	//
	// Döndürür:
	//   - error: Silme hatası
	Delete(queue string, job Job) error

	// Release, job'ı tekrar kuyruğa ekler.
	//
	// Job başarısız olduğunda retry için kullanılır.
	//
	// Parametreler:
	//   - queue: Kuyruk adı
	//   - job: Tekrar kuyruğa eklenecek job
	//   - delay: Gecikme süresi
	//
	// Döndürür:
	//   - error: Release hatası
	Release(queue string, job Job, delay time.Duration) error

	// Size, kuyruktaki job sayısını döndürür.
	//
	// Parametreler:
	//   - queue: Kuyruk adı
	//
	// Döndürür:
	//   - int64: Job sayısı
	//   - error: Hata
	Size(queue string) (int64, error)
}
