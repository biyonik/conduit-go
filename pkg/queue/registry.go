// -----------------------------------------------------------------------------
// Job Registry
// -----------------------------------------------------------------------------
// Job type'ları register etmek için registry.
//
// Her job tipi register edilmeli ki worker deserialize edebilsin.
//
// Kullanım:
//   queue.RegisterJob("*jobs.SendEmailJob", func() queue.Job {
//       return &jobs.SendEmailJob{}
//   })
// -----------------------------------------------------------------------------

package queue

import (
	"fmt"
	"sync"
)

// JobFactory, job instance oluşturan fonksiyon tipi.
type JobFactory func() Job

// Registry, job type'larını saklayan yapı.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]JobFactory
}

// JobRegistry, global job registry.
var JobRegistry = &Registry{
	factories: make(map[string]JobFactory),
}

// Register, yeni bir job tipi register eder.
//
// Parametreler:
//   - jobType: Job type string'i (örn: "*jobs.SendEmailJob")
//   - factory: Job instance oluşturan fonksiyon
//
// Örnek:
//
//	queue.RegisterJob("*jobs.SendEmailJob", func() queue.Job {
//	    return &jobs.SendEmailJob{}
//	})
func (r *Registry) Register(jobType string, factory JobFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[jobType] = factory
}

// Create, job type'ına göre yeni bir instance oluşturur.
//
// Parametreler:
//   - jobType: Job type string'i
//
// Döndürür:
//   - Job: Oluşturulan job instance
//   - error: Job tipi register edilmemişse hata
func (r *Registry) Create(jobType string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[jobType]
	if !exists {
		return nil, fmt.Errorf("job tipi register edilmemiş: %s", jobType)
	}

	return factory(), nil
}

// RegisterJob, global registry'ye job register eder.
//
// Parametreler:
//   - jobType: Job type string'i
//   - factory: Job instance oluşturan fonksiyon
//
// Örnek:
//
//	queue.RegisterJob("*jobs.SendEmailJob", func() queue.Job {
//	    return &jobs.SendEmailJob{}
//	})
func RegisterJob(jobType string, factory JobFactory) {
	JobRegistry.Register(jobType, factory)
}
