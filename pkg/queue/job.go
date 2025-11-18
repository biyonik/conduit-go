// -----------------------------------------------------------------------------
// Job Interface
// -----------------------------------------------------------------------------
// Queue sistemindeki job'ların implement etmesi gereken interface.
//
// Her job:
// - Handle(): İşi yapar
// - Failed(): Başarısız olursa çağrılır
// - GetPayload/SetPayload: Serialization
// - Metadata: ID, attempts, queue name
// -----------------------------------------------------------------------------

package queue

import (
	"encoding/json"
	"time"
)

// Job, queue sistemindeki tüm job'ların implement etmesi gereken interface.
type Job interface {
	// Handle, job'ın asıl işini yapar.
	//
	// Döndürür:
	//   - error: İşlem hatası (retry için)
	//
	// Örnek:
	//   func (j *SendEmailJob) Handle() error {
	//       return sendEmail(j.To, j.Subject, j.Body)
	//   }
	Handle() error

	// Failed, job başarısız olduğunda çağrılır.
	//
	// Parametreler:
	//   - err: Hata nedeni
	//
	// Döndürür:
	//   - error: Failed handler hatası
	//
	// Örnek:
	//   func (j *SendEmailJob) Failed(err error) error {
	//       log.Printf("Email failed: %v", err)
	//       return nil
	//   }
	Failed(err error) error

	// GetPayload, job'ı JSON'a serialize eder.
	//
	// Döndürür:
	//   - []byte: JSON payload
	//   - error: Serialization hatası
	GetPayload() ([]byte, error)

	// SetPayload, JSON'dan job'ı deserialize eder.
	//
	// Parametreler:
	//   - data: JSON payload
	//
	// Döndürür:
	//   - error: Deserialization hatası
	SetPayload(data []byte) error

	// GetID, job'ın unique ID'sini döndürür.
	GetID() string

	// SetID, job'ın ID'sini set eder.
	SetID(id string)

	// GetAttempts, job'ın kaç kere denendiğini döndürür.
	GetAttempts() int

	// SetAttempts, attempt sayısını set eder.
	SetAttempts(attempts int)

	// GetQueue, job'ın hangi kuyrukta olduğunu döndürür.
	GetQueue() string

	// SetQueue, job'ın kuyruğunu set eder.
	SetQueue(queue string)

	// GetMaxAttempts, maksimum deneme sayısını döndürür.
	GetMaxAttempts() int
}

// BaseJob, tüm job'ların gömebileceği temel yapı.
//
// Bu struct, Job interface'inin metadata metodlarını implement eder.
// Gerçek job'lar sadece Handle() ve Failed() metodlarını implement etmeli.
type BaseJob struct {
	ID          string `json:"id"`
	Queue       string `json:"queue"`
	Attempts    int    `json:"attempts"`
	MaxAttempts int    `json:"max_attempts"`
}

// GetID, job ID'sini döndürür.
func (b *BaseJob) GetID() string {
	return b.ID
}

// SetID, job ID'sini set eder.
func (b *BaseJob) SetID(id string) {
	b.ID = id
}

// GetAttempts, attempt sayısını döndürür.
func (b *BaseJob) GetAttempts() int {
	return b.Attempts
}

// SetAttempts, attempt sayısını set eder.
func (b *BaseJob) SetAttempts(attempts int) {
	b.Attempts = attempts
}

// GetQueue, queue adını döndürür.
func (b *BaseJob) GetQueue() string {
	return b.Queue
}

// SetQueue, queue adını set eder.
func (b *BaseJob) SetQueue(queue string) {
	b.Queue = queue
}

// GetMaxAttempts, maksimum deneme sayısını döndürür.
func (b *BaseJob) GetMaxAttempts() int {
	if b.MaxAttempts == 0 {
		return 3 // Varsayılan
	}
	return b.MaxAttempts
}

// JobPayload, queue'da saklanan job wrapper'ı.
//
// Bu struct job'ı metadata ile birlikte saklar.
type JobPayload struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`         // Job tipi (SendEmailJob, ProcessUploadJob, vb.)
	Queue       string          `json:"queue"`        // Kuyruk adı
	Payload     json.RawMessage `json:"payload"`      // Gerçek job data
	Attempts    int             `json:"attempts"`     // Deneme sayısı
	MaxAttempts int             `json:"max_attempts"` // Maksimum deneme
	CreatedAt   time.Time       `json:"created_at"`   // Oluşturulma zamanı
	AvailableAt time.Time       `json:"available_at"` // İşlenebilir olacağı zaman (delayed jobs için)
}
