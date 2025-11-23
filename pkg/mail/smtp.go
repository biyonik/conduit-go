// -----------------------------------------------------------------------------
// SMTP Mailer Driver
// -----------------------------------------------------------------------------
// Bu dosya, SMTP protokolü üzerinden email gönderimi yapar.
//
// Desteklenen SMTP Servisleri:
// - Gmail (smtp.gmail.com:587)
// - Mailhog (localhost:1025 - development)
// - SendGrid (smtp.sendgrid.net:587)
// - AWS SES (email-smtp.region.amazonaws.com:587)
// - Mailgun (smtp.mailgun.org:587)
// - Custom SMTP servers
//
// Kullanım:
//
//	config := &mail.SMTPConfig{
//	    Host:     "smtp.gmail.com",
//	    Port:     587,
//	    Username: "your@gmail.com",
//	    Password: "app-password",
//	    From:     mail.Address{Email: "noreply@conduit.com", Name: "Conduit"},
//	}
//	mailer := mail.NewSMTPMailer(config, logger)
// -----------------------------------------------------------------------------

package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SMTPConfig, SMTP bağlantı ayarlarını içerir.
type SMTPConfig struct {
	Host     string  // SMTP sunucu adresi (örn: smtp.gmail.com)
	Port     int     // SMTP port (25, 587, 465)
	Username string  // SMTP kullanıcı adı
	Password string  // SMTP şifre
	From     Address // Varsayılan gönderici adresi
	UseTLS   bool    // TLS kullanılsın mı (587 port için true)
	Timeout  time.Duration
}

// SMTPMailer, SMTP ile email gönderen mailer.
type SMTPMailer struct {
	*BaseMailer
	config *SMTPConfig
}

// NewSMTPMailer, yeni bir SMTP mailer oluşturur.
//
// Parametreler:
//   - config: SMTP konfigürasyonu
//   - logger: Logger instance
//
// Döndürür:
//   - *SMTPMailer: Yeni SMTP mailer
//
// Örnek (Gmail):
//
//	config := &mail.SMTPConfig{
//	    Host:     "smtp.gmail.com",
//	    Port:     587,
//	    Username: "your@gmail.com",
//	    Password: "app-password",
//	    UseTLS:   true,
//	}
//	mailer := mail.NewSMTPMailer(config, logger)
//
// Örnek (Mailhog - Development):
//
//	config := &mail.SMTPConfig{
//	    Host:     "localhost",
//	    Port:     1025,
//	    From:     mail.Address{Email: "dev@conduit.local"},
//	}
//	mailer := mail.NewSMTPMailer(config, logger)
func NewSMTPMailer(config *SMTPConfig, logger Logger) *SMTPMailer {
	// Varsayılan timeout
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SMTPMailer{
		BaseMailer: NewBaseMailer(logger),
		config:     config,
	}
}

// Send, email'i SMTP üzerinden gönderir.
func (m *SMTPMailer) Send(message *Message) error {
	// Validate message
	if err := m.ValidateMessage(message); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Log sending
	m.LogSending(message)

	// From adresi yoksa config'den al
	if message.GetFrom().Email == "" {
		message.From(m.config.From.Email, m.config.From.Name)
	}

	// SMTP server adresi
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	// Auth (eğer username/password varsa)
	var auth smtp.Auth
	if m.config.Username != "" && m.config.Password != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	// Alıcıları topla
	recipients := m.collectRecipients(message)

	// Email içeriğini oluştur
	emailBody, err := m.buildEmail(message)
	if err != nil {
		m.LogError(message, err)
		return fmt.Errorf("failed to build email: %w", err)
	}

	// Email gönder
	err = smtp.SendMail(addr, auth, message.GetFrom().Email, recipients, emailBody)
	if err != nil {
		m.LogError(message, err)
		return fmt.Errorf("smtp send failed: %w", err)
	}

	// Log success
	m.LogSuccess(message)

	return nil
}

// SendAsync, queue yoksa senkron gönderir.
// Queue integration yapıldığında bu metod güncellenecek.
func (m *SMTPMailer) SendAsync(message *Message) error {
	// TODO (Phase 3): Queue sistemi ile entegrasyon
	// Şimdilik senkron gönder
	return m.Send(message)
}

// collectRecipients, tüm alıcıları (To, Cc, Bcc) toplar.
func (m *SMTPMailer) collectRecipients(message *Message) []string {
	recipients := make([]string, 0)

	for _, to := range message.GetTo() {
		recipients = append(recipients, to.Email)
	}

	for _, cc := range message.GetCc() {
		recipients = append(recipients, cc.Email)
	}

	for _, bcc := range message.GetBcc() {
		recipients = append(recipients, bcc.Email)
	}

	return recipients
}

// buildEmail, email içeriğini MIME formatında oluşturur.
func (m *SMTPMailer) buildEmail(message *Message) ([]byte, error) {
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", message.GetFrom().String()))

	// To
	toAddrs := make([]string, len(message.GetTo()))
	for i, to := range message.GetTo() {
		toAddrs[i] = to.String()
	}
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toAddrs, ", ")))

	// Cc
	if len(message.GetCc()) > 0 {
		ccAddrs := make([]string, len(message.GetCc()))
		for i, cc := range message.GetCc() {
			ccAddrs[i] = cc.String()
		}
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccAddrs, ", ")))
	}

	// Reply-To
	if message.GetReplyTo() != nil {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", message.GetReplyTo().String()))
	}

	// Subject
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", message.GetSubject()))

	// Date
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", message.GetDate().Format(time.RFC1123Z)))

	// Priority
	if message.GetPriority() != PriorityNormal {
		buf.WriteString(fmt.Sprintf("X-Priority: %d\r\n", message.GetPriority()))
	}

	// Custom headers
	for key, value := range message.GetHeaders() {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// MIME headers
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Attachment varsa multipart/mixed, yoksa multipart/alternative
	if len(message.GetAttachments()) > 0 {
		return m.buildMultipartWithAttachments(&buf, message)
	}

	return m.buildMultipartAlternative(&buf, message)
}

// buildMultipartAlternative, plain text ve HTML içeriği olan email oluşturur.
func (m *SMTPMailer) buildMultipartAlternative(buf *bytes.Buffer, message *Message) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary))

	// Plain text part
	if message.GetBody() != "" {
		part, _ := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type": []string{"text/plain; charset=UTF-8"},
		})
		part.Write([]byte(message.GetBody()))
	}

	// HTML part
	if message.GetHtmlBody() != "" {
		part, _ := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type": []string{"text/html; charset=UTF-8"},
		})
		part.Write([]byte(message.GetHtmlBody()))
	}

	writer.Close()

	return buf.Bytes(), nil
}

// buildMultipartWithAttachments, ek dosyalı email oluşturur.
func (m *SMTPMailer) buildMultipartWithAttachments(buf *bytes.Buffer, message *Message) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))

	// Text/HTML content part
	if message.GetBody() != "" || message.GetHtmlBody() != "" {
		contentWriter := multipart.NewWriter(buf)
		contentBoundary := contentWriter.Boundary()

		part, _ := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type": []string{fmt.Sprintf("multipart/alternative; boundary=%s", contentBoundary)},
		})

		// Plain text
		if message.GetBody() != "" {
			textPart, _ := contentWriter.CreatePart(textproto.MIMEHeader{
				"Content-Type": []string{"text/plain; charset=UTF-8"},
			})
			textPart.Write([]byte(message.GetBody()))
		}

		// HTML
		if message.GetHtmlBody() != "" {
			htmlPart, _ := contentWriter.CreatePart(textproto.MIMEHeader{
				"Content-Type": []string{"text/html; charset=UTF-8"},
			})
			htmlPart.Write([]byte(message.GetHtmlBody()))
		}

		contentWriter.Close()
		part.Write([]byte("\r\n"))
	}

	// Attachments
	for _, filePath := range message.GetAttachments() {
		if err := m.addAttachment(writer, filePath); err != nil {
			return nil, fmt.Errorf("failed to attach file %s: %w", filePath, err)
		}
	}

	writer.Close()

	return buf.Bytes(), nil
}

// addAttachment, dosyayı ek olarak ekler.
func (m *SMTPMailer) addAttachment(writer *multipart.Writer, filePath string) error {
	// Dosyayı aç
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Dosya içeriğini oku
	fileData, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// MIME header oluştur
	fileName := filepath.Base(filePath)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              []string{"application/octet-stream"},
		"Content-Disposition":       []string{fmt.Sprintf("attachment; filename=\"%s\"", fileName)},
		"Content-Transfer-Encoding": []string{"base64"},
	})
	if err != nil {
		return err
	}

	// Base64 encode et ve yaz
	encoded := base64.StdEncoding.EncodeToString(fileData)

	// 76 karakterde satır sonu ekle (RFC 2045)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		part.Write([]byte(encoded[i:end] + "\r\n"))
	}

	return nil
}
