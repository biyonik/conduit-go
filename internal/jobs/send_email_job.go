// -----------------------------------------------------------------------------
// Send Email Job
// -----------------------------------------------------------------------------
// Email gönderme job'u.
//
// Bu job mail queue'sunda çalışır ve email gönderir.
// Başarısız olursa 3 kere denenir.
//
// Phase 3 Update:
// Artık gerçek mail sistemi kullanılıyor (pkg/mail).
// Mailer dependency injection ile sağlanır.
// -----------------------------------------------------------------------------

package jobs

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/biyonik/conduit-go/pkg/mail"
	"github.com/biyonik/conduit-go/pkg/queue"
)

// SendEmailJob, email gönderme job'u.
type SendEmailJob struct {
	queue.BaseJob
	To       string `json:"to"`
	ToName   string `json:"to_name"`   // Alıcı adı (opsiyonel)
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	HtmlBody string `json:"html_body"` // HTML içerik (opsiyonel)
	From     string `json:"from"`      // Gönderici email (opsiyonel)
	FromName string `json:"from_name"` // Gönderici adı (opsiyonel)

	// Dependency injection için (serialize edilmez)
	Mailer mail.Mailer `json:"-"`
}

// Handle, email gönderme işlemini yapar.
//
// Artık gerçek mail sistemi (pkg/mail) kullanılıyor.
// Mailer dependency'si job oluşturulurken inject edilmelidir.
func (j *SendEmailJob) Handle() error {
	log.Printf("📧 Sending email to: %s", j.To)
	log.Printf("   Subject: %s", j.Subject)

	// Mailer yoksa fallback (backward compatibility)
	if j.Mailer == nil {
		log.Printf("⚠️  No mailer configured, simulating email send")
		log.Printf("✅ Email simulated successfully to: %s", j.To)
		return nil
	}

	// Email mesajı oluştur
	message := mail.NewMessage()

	// Gönderici (varsa)
	if j.From != "" {
		message.From(j.From, j.FromName)
	}

	// Alıcı
	message.To(j.To, j.ToName)

	// Konu
	message.Subject(j.Subject)

	// İçerik
	if j.Body != "" {
		message.Body(j.Body)
	}

	if j.HtmlBody != "" {
		message.Html(j.HtmlBody)
	}

	// Email gönder
	if err := j.Mailer.Send(message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

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
//   - body: Email içeriği (plain text)
//   - mailer: Mail driver (dependency injection)
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
//	    mailer,
//	)
//	queue.Push(job, "emails")
//
// HTML Email Örneği:
//
//	job := jobs.NewSendEmailJob("user@example.com", "Welcome", "", mailer)
//	job.HtmlBody = "<h1>Welcome!</h1><p>Thank you for joining.</p>"
//	queue.Push(job, "emails")
func NewSendEmailJob(to, subject, body string, mailer mail.Mailer) *SendEmailJob {
	return &SendEmailJob{
		BaseJob: queue.BaseJob{
			MaxAttempts: 3,
		},
		To:      to,
		Subject: subject,
		Body:    body,
		Mailer:  mailer,
	}
}

// NewSendHtmlEmailJob, HTML email job'u oluşturur.
//
// Parametreler:
//   - to: Alıcı email adresi
//   - toName: Alıcı adı
//   - subject: Email konusu
//   - htmlBody: HTML içerik
//   - mailer: Mail driver
//
// Döndürür:
//   - *SendEmailJob: Job instance
//
// Örnek:
//
//	job := jobs.NewSendHtmlEmailJob(
//	    "user@example.com",
//	    "John Doe",
//	    "Welcome!",
//	    "<h1>Welcome to Conduit!</h1>",
//	    mailer,
//	)
func NewSendHtmlEmailJob(to, toName, subject, htmlBody string, mailer mail.Mailer) *SendEmailJob {
	return &SendEmailJob{
		BaseJob: queue.BaseJob{
			MaxAttempts: 3,
		},
		To:       to,
		ToName:   toName,
		Subject:  subject,
		HtmlBody: htmlBody,
		Mailer:   mailer,
	}
}
