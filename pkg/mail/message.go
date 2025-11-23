// -----------------------------------------------------------------------------
// Email Message Builder
// -----------------------------------------------------------------------------
// Bu dosya, email mesajlarını oluşturmak için fluent API sağlar.
//
// Laravel Mail Facade'ine benzer şekilde çalışır:
//
//	message := mail.NewMessage()
//	    .From("noreply@conduit.com", "Conduit")
//	    .To("user@example.com")
//	    .Subject("Welcome to Conduit!")
//	    .Body("Welcome to our platform!")
//	    .Html("<h1>Welcome!</h1>")
// -----------------------------------------------------------------------------

package mail

import (
	"fmt"
	"time"
)

// Address, email adresi ve opsiyonel isim içeren yapıdır.
type Address struct {
	Email string // Email adresi (zorunlu)
	Name  string // İsim (opsiyonel)
}

// String, Address'i "Name <email@example.com>" formatında döndürür.
func (a Address) String() string {
	if a.Name != "" {
		return fmt.Sprintf("%s <%s>", a.Name, a.Email)
	}
	return a.Email
}

// Message, email mesajını temsil eder.
//
// Fluent API ile zincirleme kullanım:
//
//	msg := mail.NewMessage().
//	    To("user@example.com").
//	    Subject("Hello").
//	    Body("Welcome!")
type Message struct {
	from        Address
	to          []Address
	cc          []Address
	bcc         []Address
	replyTo     *Address
	subject     string
	body        string   // Plain text body
	htmlBody    string   // HTML body
	attachments []string // File paths
	headers     map[string]string
	priority    Priority
	date        time.Time
}

// Priority, email öncelik seviyesi.
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityNormal Priority = 3
	PriorityHigh   Priority = 5
)

// NewMessage, yeni bir Message instance'ı oluşturur.
//
// Döndürür:
//   - *Message: Yeni message instance
//
// Örnek:
//
//	msg := mail.NewMessage()
func NewMessage() *Message {
	return &Message{
		headers:  make(map[string]string),
		priority: PriorityNormal,
		date:     time.Now(),
	}
}

// From, gönderici adresini ayarlar.
//
// Parametreler:
//   - email: Gönderici email adresi
//   - name: Gönderici adı (opsiyonel, boş string olabilir)
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.From("noreply@conduit.com", "Conduit Team")
func (m *Message) From(email string, name string) *Message {
	m.from = Address{Email: email, Name: name}
	return m
}

// To, alıcı adresini ekler.
//
// Parametreler:
//   - email: Alıcı email adresi
//   - name: Alıcı adı (opsiyonel)
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.To("user@example.com", "John Doe")
//	msg.To("admin@example.com", "") // İsim olmadan
func (m *Message) To(email string, name string) *Message {
	m.to = append(m.to, Address{Email: email, Name: name})
	return m
}

// Cc, CC (Carbon Copy) alıcısı ekler.
func (m *Message) Cc(email string, name string) *Message {
	m.cc = append(m.cc, Address{Email: email, Name: name})
	return m
}

// Bcc, BCC (Blind Carbon Copy) alıcısı ekler.
func (m *Message) Bcc(email string, name string) *Message {
	m.bcc = append(m.bcc, Address{Email: email, Name: name})
	return m
}

// ReplyTo, yanıt adresi ayarlar.
func (m *Message) ReplyTo(email string, name string) *Message {
	m.replyTo = &Address{Email: email, Name: name}
	return m
}

// Subject, email konusunu ayarlar.
//
// Parametre:
//   - subject: Email konusu
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.Subject("Welcome to Conduit!")
func (m *Message) Subject(subject string) *Message {
	m.subject = subject
	return m
}

// Body, plain text email gövdesini ayarlar.
//
// Parametre:
//   - body: Plain text içerik
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.Body("Welcome to our platform!\n\nThank you for joining.")
func (m *Message) Body(body string) *Message {
	m.body = body
	return m
}

// Html, HTML email gövdesini ayarlar.
//
// Parametre:
//   - html: HTML içerik
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.Html("<h1>Welcome!</h1><p>Thank you for joining.</p>")
//
// Güvenlik Notu:
// HTML içeriği için XSS koruması yapılmaz, güvenilir kaynaklardan
// HTML kullanın veya template engine kullanın.
func (m *Message) Html(html string) *Message {
	m.htmlBody = html
	return m
}

// Attach, dosya ekler.
//
// Parametre:
//   - filePath: Eklenecek dosyanın yolu
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.Attach("/path/to/document.pdf")
//	msg.Attach("/path/to/image.jpg")
func (m *Message) Attach(filePath string) *Message {
	m.attachments = append(m.attachments, filePath)
	return m
}

// Priority, email önceliğini ayarlar.
//
// Parametre:
//   - priority: Öncelik seviyesi (Low, Normal, High)
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.Priority(mail.PriorityHigh)
func (m *Message) Priority(priority Priority) *Message {
	m.priority = priority
	return m
}

// Header, özel header ekler.
//
// Parametreler:
//   - key: Header adı
//   - value: Header değeri
//
// Döndürür:
//   - *Message: Zincirleme için kendi instance'ını döner
//
// Örnek:
//
//	msg.Header("X-Custom-Header", "custom-value")
//	msg.Header("X-Campaign-ID", "summer-2024")
func (m *Message) Header(key, value string) *Message {
	m.headers[key] = value
	return m
}

// Validate, message'ın geçerli olup olmadığını kontrol eder.
//
// Döndürür:
//   - error: Geçersizse hata, geçerliyse nil
//
// Kontroller:
// - From adresi dolu olmalı
// - En az bir To adresi olmalı
// - Subject dolu olmalı
// - Body veya HtmlBody dolu olmalı
func (m *Message) Validate() error {
	if m.from.Email == "" {
		return fmt.Errorf("sender address is required")
	}

	if len(m.to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if m.subject == "" {
		return fmt.Errorf("subject is required")
	}

	if m.body == "" && m.htmlBody == "" {
		return fmt.Errorf("body or html body is required")
	}

	return nil
}

// GetFrom, gönderici adresini döndürür.
func (m *Message) GetFrom() Address {
	return m.from
}

// GetTo, alıcı adreslerini döndürür.
func (m *Message) GetTo() []Address {
	return m.to
}

// GetCc, CC adreslerini döndürür.
func (m *Message) GetCc() []Address {
	return m.cc
}

// GetBcc, BCC adreslerini döndürür.
func (m *Message) GetBcc() []Address {
	return m.bcc
}

// GetReplyTo, yanıt adresini döndürür.
func (m *Message) GetReplyTo() *Address {
	return m.replyTo
}

// GetSubject, konuyu döndürür.
func (m *Message) GetSubject() string {
	return m.subject
}

// GetBody, plain text gövdeyi döndürür.
func (m *Message) GetBody() string {
	return m.body
}

// GetHtmlBody, HTML gövdeyi döndürür.
func (m *Message) GetHtmlBody() string {
	return m.htmlBody
}

// GetAttachments, ekleri döndürür.
func (m *Message) GetAttachments() []string {
	return m.attachments
}

// GetHeaders, özel header'ları döndürür.
func (m *Message) GetHeaders() map[string]string {
	return m.headers
}

// GetPriority, önceliği döndürür.
func (m *Message) GetPriority() Priority {
	return m.priority
}

// GetDate, tarih döndürür.
func (m *Message) GetDate() time.Time {
	return m.date
}
