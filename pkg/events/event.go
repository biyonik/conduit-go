// -----------------------------------------------------------------------------
// Event System - Core Interfaces
// -----------------------------------------------------------------------------
// Bu dosya, event-driven architecture için temel yapıları içerir.
//
// Event Nedir?
// Event (olay), sistemde meydana gelen önemli bir durumu temsil eder.
// Örnek: UserRegistered, OrderPlaced, EmailSent, PaymentCompleted
//
// Neden Event-Driven Architecture?
// - Loose coupling (bağımlılıkların azaltılması)
// - Separation of concerns (sorumlulukların ayrılması)
// - Scalability (ölçeklenebilirlik)
// - Testability (test edilebilirlik)
//
// Laravel Event Pattern:
// Event::dispatch(new UserRegistered($user))
// Event::listen(UserRegistered::class, SendWelcomeEmail::class)
// -----------------------------------------------------------------------------

package events

import (
	"time"
)

// Event, tüm event'lerin implement etmesi gereken interface.
//
// Her event şu bilgileri sağlamalıdır:
//   - Name: Event'in unique adı
//   - OccurredAt: Event'in gerçekleşme zamanı
//   - Payload: Event ile taşınan veri
type Event interface {
	// Name, event'in benzersiz adını döndürür.
	// Örnek: "user.registered", "order.placed"
	Name() string

	// OccurredAt, event'in gerçekleşme zamanını döndürür.
	OccurredAt() time.Time

	// Payload, event ile taşınan veriyi döndürür.
	// Generic interface{} olduğu için her türlü data taşınabilir.
	Payload() interface{}
}

// BaseEvent, tüm custom event'ler için temel yapıdır.
//
// Custom event oluştururken BaseEvent'i embed edin:
//
//	type UserRegistered struct {
//	    events.BaseEvent
//	    User *models.User
//	}
//
// Bu sayede Name() ve OccurredAt() metodlarını otomatik implement etmiş olursunuz.
type BaseEvent struct {
	name       string
	occurredAt time.Time
	payload    interface{}
}

// NewBaseEvent, yeni bir BaseEvent oluşturur.
//
// Parametreler:
//   - name: Event adı (örn: "user.registered")
//   - payload: Event verisi (optional, nil olabilir)
//
// Döndürür:
//   - *BaseEvent: BaseEvent instance
//
// Örnek:
//
//	event := events.NewBaseEvent("user.registered", user)
func NewBaseEvent(name string, payload interface{}) *BaseEvent {
	return &BaseEvent{
		name:       name,
		occurredAt: time.Now(),
		payload:    payload,
	}
}

// Name, event adını döndürür.
func (e *BaseEvent) Name() string {
	return e.name
}

// OccurredAt, event zamanını döndürür.
func (e *BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// Payload, event verisini döndürür.
func (e *BaseEvent) Payload() interface{} {
	return e.payload
}

// SetPayload, event verisini günceller.
// Bu metod BaseEvent'i embed eden struct'lar için kullanışlıdır.
func (e *BaseEvent) SetPayload(payload interface{}) {
	e.payload = payload
}

// -----------------------------------------------------------------------------
// Common Event Types (Yaygın Event Tipleri)
// -----------------------------------------------------------------------------
// Bazı yaygın event tipleri önceden tanımlanmıştır.
// Bunları doğrudan kullanabilir veya kendi event'lerinizi oluşturabilirsiniz.

const (
	// User Events
	EventUserRegistered      = "user.registered"
	EventUserUpdated         = "user.updated"
	EventUserDeleted         = "user.deleted"
	EventUserEmailVerified   = "user.email.verified"
	EventUserPasswordChanged = "user.password.changed"
	EventUserLoggedIn        = "user.logged.in"
	EventUserLoggedOut       = "user.logged.out"

	// Order Events (e-commerce için)
	EventOrderCreated   = "order.created"
	EventOrderUpdated   = "order.updated"
	EventOrderCancelled = "order.cancelled"
	EventOrderShipped   = "order.shipped"
	EventOrderDelivered = "order.delivered"

	// Payment Events
	EventPaymentReceived = "payment.received"
	EventPaymentFailed   = "payment.failed"
	EventPaymentRefunded = "payment.refunded"

	// Email Events
	EventEmailSent   = "email.sent"
	EventEmailFailed = "email.failed"

	// File Upload Events
	EventFileUploaded = "file.uploaded"
	EventFileDeleted  = "file.deleted"

	// Cache Events
	EventCacheHit      = "cache.hit"
	EventCacheMiss     = "cache.miss"
	EventCacheCleared  = "cache.cleared"
	EventCacheKeyFlush = "cache.key.flush"
)

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// NewUserRegisteredEvent, kullanıcı kaydı event'i oluşturur.
//
// Parametre:
//   - user: Kaydolan kullanıcı objesi
//
// Döndürür:
//   - Event: UserRegistered event'i
//
// Kullanım:
//
//	event := events.NewUserRegisteredEvent(user)
//	dispatcher.Dispatch(event)
func NewUserRegisteredEvent(user interface{}) Event {
	return NewBaseEvent(EventUserRegistered, user)
}

// NewUserLoggedInEvent, kullanıcı login event'i oluşturur.
func NewUserLoggedInEvent(user interface{}) Event {
	return NewBaseEvent(EventUserLoggedIn, user)
}

// NewEmailSentEvent, email gönderildi event'i oluşturur.
func NewEmailSentEvent(emailData interface{}) Event {
	return NewBaseEvent(EventEmailSent, emailData)
}

// NewPaymentReceivedEvent, ödeme alındı event'i oluşturur.
func NewPaymentReceivedEvent(payment interface{}) Event {
	return NewBaseEvent(EventPaymentReceived, payment)
}
