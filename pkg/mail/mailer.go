// -----------------------------------------------------------------------------
// Mail Package - Laravel-Inspired Email System
// -----------------------------------------------------------------------------
// Bu package, email gönderimi için Laravel Mail Facade'ine benzer bir
// interface sağlar.
//
// Özellikler:
// - SMTP driver desteği
// - Fluent message builder
// - Template support
// - Queue integration
// - Multiple driver support (SMTP, Mailhog, Log, vb.)
//
// Kullanım:
//
//	mailer := mail.NewSMTPMailer(config, logger)
//	message := mail.NewMessage().
//	    From("noreply@conduit.com", "Conduit").
//	    To("user@example.com").
//	    Subject("Welcome!").
//	    Body("Welcome to Conduit!")
//	err := mailer.Send(message)
// -----------------------------------------------------------------------------

package mail

import (
	"fmt"
)

// Mailer, email gönderim interface'i.
//
// Farklı driver'lar (SMTP, Mailgun, SendGrid, SES, vb.) bu
// interface'i implement ederek sistemle entegre olabilir.
type Mailer interface {
	// Send, bir email mesajı gönderir.
	//
	// Parametre:
	//   - message: Gönderilecek mesaj
	//
	// Döndürür:
	//   - error: Gönderim başarısızsa hata
	Send(message *Message) error

	// SendAsync, email'i queue'ya ekleyerek asenkron gönderir.
	// Queue sistemi varsa kullanılır, yoksa senkron Send() çağrılır.
	//
	// Parametre:
	//   - message: Gönderilecek mesaj
	//
	// Döndürür:
	//   - error: Queue'ya ekleme başarısızsa hata
	SendAsync(message *Message) error
}

// Logger interface - dependency injection için
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// BaseMailer, tüm mailer implementasyonları için temel yapı.
//
// Bu yapı ortak fonksiyonları sağlar, her driver bu yapıyı embed eder.
type BaseMailer struct {
	logger Logger
}

// NewBaseMailer, yeni bir BaseMailer oluşturur.
func NewBaseMailer(logger Logger) *BaseMailer {
	return &BaseMailer{
		logger: logger,
	}
}

// ValidateMessage, mesajı validate eder.
func (m *BaseMailer) ValidateMessage(message *Message) error {
	return message.Validate()
}

// LogSending, gönderim işlemini loglar.
func (m *BaseMailer) LogSending(message *Message) {
	m.logger.Printf("📧 Sending email to: %s", message.GetTo()[0].Email)
	m.logger.Printf("   Subject: %s", message.GetSubject())
	m.logger.Printf("   From: %s", message.GetFrom().String())
}

// LogSuccess, başarılı gönderimi loglar.
func (m *BaseMailer) LogSuccess(message *Message) {
	m.logger.Printf("✅ Email sent successfully to: %s", message.GetTo()[0].Email)
}

// LogError, hata oluştuğunda loglar.
func (m *BaseMailer) LogError(message *Message, err error) {
	m.logger.Printf("❌ Email send failed to: %s - Error: %v", message.GetTo()[0].Email, err)
}

// -----------------------------------------------------------------------------
// Log Mailer (Development/Testing için)
// -----------------------------------------------------------------------------

// LogMailer, email'leri göndermek yerine loglara yazan mailer.
//
// Development ve test ortamında kullanışlıdır.
// Gerçek email gönderilmez, sadece log'a yazılır.
//
// Kullanım:
//
//	mailer := mail.NewLogMailer(logger)
//	err := mailer.Send(message)
type LogMailer struct {
	*BaseMailer
}

// NewLogMailer, yeni bir LogMailer oluşturur.
//
// Parametre:
//   - logger: Log yazımı için logger
//
// Döndürür:
//   - *LogMailer: Yeni LogMailer instance
//
// Örnek:
//
//	mailer := mail.NewLogMailer(log.Default())
func NewLogMailer(logger Logger) *LogMailer {
	return &LogMailer{
		BaseMailer: NewBaseMailer(logger),
	}
}

// Send, email'i loglara yazar (gerçek gönderim yapmaz).
func (m *LogMailer) Send(message *Message) error {
	// Validate
	if err := m.ValidateMessage(message); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Log email details
	m.logger.Println("\n" + "=".repeat(70))
	m.logger.Println("📧 EMAIL (LOG DRIVER - NOT ACTUALLY SENT)")
	m.logger.Println("=".repeat(70))
	m.logger.Printf("From: %s", message.GetFrom().String())

	for _, to := range message.GetTo() {
		m.logger.Printf("To: %s", to.String())
	}

	if len(message.GetCc()) > 0 {
		for _, cc := range message.GetCc() {
			m.logger.Printf("Cc: %s", cc.String())
		}
	}

	m.logger.Printf("Subject: %s", message.GetSubject())
	m.logger.Println("---")

	if message.GetBody() != "" {
		m.logger.Println("Body (Plain Text):")
		m.logger.Println(message.GetBody())
	}

	if message.GetHtmlBody() != "" {
		m.logger.Println("Body (HTML):")
		m.logger.Println(message.GetHtmlBody())
	}

	if len(message.GetAttachments()) > 0 {
		m.logger.Println("Attachments:")
		for _, att := range message.GetAttachments() {
			m.logger.Printf("  - %s", att)
		}
	}

	m.logger.Println("=".repeat(70) + "\n")

	return nil
}

// SendAsync, log driver için Send() ile aynıdır.
func (m *LogMailer) SendAsync(message *Message) error {
	return m.Send(message)
}

// -----------------------------------------------------------------------------
// String Helper
// -----------------------------------------------------------------------------

type repeatableString string

func (s repeatableString) repeat(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += string(s)
	}
	return result
}
