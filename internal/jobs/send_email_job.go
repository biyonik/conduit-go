// -----------------------------------------------------------------------------
// Send Email Job
// -----------------------------------------------------------------------------
// Email gönderme job'u.
//
// Bu job mail queue'sunda çalışır ve email gönderir.
// Başarısız olursa 3 kere denenir.
// -----------------------------------------------------------------------------

package jobs

import (
	"encoding/json"
	"log"

	"github.com/biyonik/conduit-go/pkg/queue"
)

// SendEmailJob, email gönderme job'u.
type SendEmailJob struct {
	queue.BaseJob
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// Handle, email gönderme işlemini yapar.
func (j *SendEmailJob) Handle() error {
	log.Printf("📧 Sending email to: %s", j.To)
	log.Printf("   Subject: %s", j.Subject)

	// TODO (Phase 3): Gerçek mail sistemi entegrasyonu
	// mail.To(j.To).Subject(j.Subject).Body(j.Body).Send()

	// Şimdilik simulate ediyoruz
	// Hata simülasyonu (test için):
	// if j.To == "fail@example.com" {
	//     return fmt.Errorf("simulated email failure")
	// }

	log.Printf("✅ Email sent successfully to: %s", j.To)
	return nil
}

// Failed, job başarısız olduğunda çağrılır.
func (j *SendEmailJob) Failed(err error) error {
	log.Printf("❌ Email job failed: %s (to: %s, error: %v)", j.ID, j.To, err)

	// TODO: Failed job'ları database'e kaydet
	// TODO: Admin'e notification gönder

	return nil
}

// GetPayload, job'ı serialize eder.
func (j *SendEmailJob) GetPayload() ([]byte, error) {
	return json.Marshal(j)
}

// SetPayload, job'ı deserialize eder.
func (j *SendEmailJob) SetPayload(data []byte) error {
	return json.Unmarshal(data, j)
}

// NewSendEmailJob, yeni bir SendEmailJob oluşturur.
//
// Parametreler:
//   - to: Alıcı email adresi
//   - subject: Email konusu
//   - body: Email içeriği
//
// Döndürür:
//   - *SendEmailJob: Job instance
//
// Örnek:
//
//	job := jobs.NewSendEmailJob(
//	    "user@example.com",
//	    "Welcome to Conduit-Go",
//	    "Hello! Welcome to our platform.",
//	)
//	queue.Push(job, "emails")
func NewSendEmailJob(to, subject, body string) *SendEmailJob {
	return &SendEmailJob{
		BaseJob: queue.BaseJob{
			MaxAttempts: 3,
		},
		To:      to,
		Subject: subject,
		Body:    body,
	}
}
