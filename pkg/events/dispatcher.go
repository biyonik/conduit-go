// -----------------------------------------------------------------------------
// Event Dispatcher
// -----------------------------------------------------------------------------
// Bu dosya, event'leri dispatch eden ve listener'ları yöneten merkezi yapıdır.
//
// Dispatcher Pattern:
// Dispatcher, observer pattern'in bir implementasyonudur. Event'ler ve
// listener'lar arasında gevşek bağlantı (loose coupling) sağlar.
//
// Laravel'deki Event::dispatch() konseptine benzer şekilde çalışır.
//
// Kullanım:
//
//	// Dispatcher oluştur
//	dispatcher := events.NewDispatcher(logger)
//
//	// Listener kaydet
//	dispatcher.Listen("user.registered", &SendWelcomeEmail{})
//	dispatcher.Listen("user.registered", &UpdateUserStats{})
//
//	// Event dispatch et
//	event := events.NewUserRegisteredEvent(user)
//	dispatcher.Dispatch(event)
// -----------------------------------------------------------------------------

package events

import (
	"fmt"
	"sync"
)

// Dispatcher, event'leri yöneten merkezi yapıdır.
//
// Özellikler:
// - Thread-safe (concurrent kullanım için güvenli)
// - Multiple listeners per event
// - Wildcard listener desteği
// - Synchronous ve asynchronous dispatch
type Dispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
	logger    Logger
}

// NewDispatcher, yeni bir Dispatcher oluşturur.
//
// Parametre:
//   - logger: Log yazımı için logger instance
//
// Döndürür:
//   - *Dispatcher: Yeni dispatcher instance
//
// Örnek:
//
//	dispatcher := events.NewDispatcher(logger)
func NewDispatcher(logger Logger) *Dispatcher {
	return &Dispatcher{
		listeners: make(map[string][]Listener),
		logger:    logger,
	}
}

// Listen, belirtilen event'e bir listener kaydeder.
//
// Bir event'e birden fazla listener kayıt edilebilir.
// Tüm listener'lar sırayla çağrılır.
//
// Parametreler:
//   - eventName: Dinlenecek event adı (örn: "user.registered")
//   - listener: Event gerçekleştiğinde çalışacak listener
//
// Örnek:
//
//	dispatcher.Listen("user.registered", &SendWelcomeEmail{})
//	dispatcher.Listen("user.registered", &SendAdminNotification{})
//
// Fonksiyon Listener:
//
//	dispatcher.Listen("user.registered", events.ListenerFunc(func(e events.Event) error {
//	    log.Println("User registered!")
//	    return nil
//	}))
func (d *Dispatcher) Listen(eventName string, listener Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.listeners[eventName] = append(d.listeners[eventName], listener)
	d.logger.Printf("✅ Listener registered for event: %s", eventName)
}

// Dispatch, bir event'i tüm kayıtlı listener'lara gönderir.
//
// Tüm listener'lar sırayla (synchronously) çalıştırılır.
// Bir listener error dönerse, diğerleri yine de çalışmaya devam eder.
//
// Parametre:
//   - event: Dispatch edilecek event
//
// Döndürür:
//   - error: Listener'lardan herhangi biri hata dönerse, son hatayı döner
//
// Örnek:
//
//	event := events.NewUserRegisteredEvent(user)
//	if err := dispatcher.Dispatch(event); err != nil {
//	    log.Printf("Event dispatch error: %v", err)
//	}
//
// Hata Yönetimi:
// Bir listener hata dönerse, log'a yazılır ama diğer listener'lar
// çalışmaya devam eder. Bu sayede bir listener'ın hatası diğerlerini engellemez.
func (d *Dispatcher) Dispatch(event Event) error {
	d.mu.RLock()
	listeners := d.listeners[event.Name()]
	d.mu.RUnlock()

	if len(listeners) == 0 {
		d.logger.Printf("⚠️  No listeners for event: %s", event.Name())
		return nil
	}

	d.logger.Printf("📢 Dispatching event: %s (listeners: %d)", event.Name(), len(listeners))

	var lastError error

	for i, listener := range listeners {
		d.logger.Printf("   [%d/%d] Executing listener for: %s", i+1, len(listeners), event.Name())

		if err := listener.Handle(event); err != nil {
			lastError = err
			d.logger.Printf("❌ Listener error for '%s': %v", event.Name(), err)
			// Hataya rağmen diğer listener'ları çalıştırmaya devam et
		}
	}

	return lastError
}

// DispatchAsync, event'i asenkron olarak dispatch eder.
//
// Event dispatch işlemi goroutine'de çalışır, bu metod hemen döner.
// Hızlı response süresi için kullanışlıdır.
//
// Parametre:
//   - event: Dispatch edilecek event
//
// Örnek:
//
//	// Async dispatch (non-blocking)
//	dispatcher.DispatchAsync(event)
//	// Kod hemen devam eder, listener'lar arka planda çalışır
//
// Uyarı:
// Async dispatch edilen event'lerin hatalarını yakalayamazsınız.
// Hatalar sadece log'a yazılır.
func (d *Dispatcher) DispatchAsync(event Event) {
	go func() {
		if err := d.Dispatch(event); err != nil {
			d.logger.Printf("❌ Async dispatch error for '%s': %v", event.Name(), err)
		}
	}()
}

// Forget, belirtilen event için tüm listener'ları kaldırır.
//
// Parametre:
//   - eventName: Temizlenecek event adı
//
// Örnek:
//
//	dispatcher.Forget("user.registered")
func (d *Dispatcher) Forget(eventName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.listeners, eventName)
	d.logger.Printf("🗑️  All listeners removed for event: %s", eventName)
}

// GetListeners, belirtilen event'in listener sayısını döndürür.
//
// Parametre:
//   - eventName: Event adı
//
// Döndürür:
//   - int: Listener sayısı
//
// Örnek:
//
//	count := dispatcher.GetListeners("user.registered")
//	fmt.Printf("Listener count: %d\n", count)
func (d *Dispatcher) GetListeners(eventName string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.listeners[eventName])
}

// HasListeners, belirtilen event için listener olup olmadığını kontrol eder.
//
// Parametre:
//   - eventName: Event adı
//
// Döndürür:
//   - bool: Listener varsa true
func (d *Dispatcher) HasListeners(eventName string) bool {
	return d.GetListeners(eventName) > 0
}

// DispatchMultiple, birden fazla event'i sırayla dispatch eder.
//
// Parametreler:
//   - events: Dispatch edilecek event'ler
//
// Döndürür:
//   - []error: Her event için hata listesi (nil ise başarılı)
//
// Örnek:
//
//	events := []events.Event{
//	    events.NewUserRegisteredEvent(user),
//	    events.NewEmailSentEvent(emailData),
//	}
//	errors := dispatcher.DispatchMultiple(events)
func (d *Dispatcher) DispatchMultiple(events []Event) []error {
	errors := make([]error, len(events))

	for i, event := range events {
		errors[i] = d.Dispatch(event)
	}

	return errors
}

// -----------------------------------------------------------------------------
// Utility Functions
// -----------------------------------------------------------------------------

// Subscribe, bir listener'ı birden fazla event'e aynı anda kaydeder.
//
// Parametreler:
//   - events: Event adları listesi
//   - listener: Kayıt edilecek listener
//
// Örnek:
//
//	// Aynı listener'ı birden fazla event'e kaydet
//	dispatcher.Subscribe(
//	    []string{"user.created", "user.updated", "user.deleted"},
//	    &LogUserActivityListener{},
//	)
func (d *Dispatcher) Subscribe(eventNames []string, listener Listener) {
	for _, eventName := range eventNames {
		d.Listen(eventName, listener)
	}
}

// Clear, tüm listener'ları temizler.
//
// Test amaçlı kullanılır, production'da dikkatli kullanın!
//
// Örnek:
//
//	dispatcher.Clear() // Tüm listener'ları sil
func (d *Dispatcher) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.listeners = make(map[string][]Listener)
	d.logger.Println("🗑️  All event listeners cleared")
}

// Stats, dispatcher istatistiklerini döndürür.
//
// Döndürür:
//   - map[string]int: Event adı -> Listener sayısı
//
// Örnek:
//
//	stats := dispatcher.Stats()
//	for event, count := range stats {
//	    fmt.Printf("%s: %d listeners\n", event, count)
//	}
func (d *Dispatcher) Stats() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]int)
	for event, listeners := range d.listeners {
		stats[event] = len(listeners)
	}

	return stats
}

// PrintStats, dispatcher istatistiklerini konsola yazdırır.
func (d *Dispatcher) PrintStats() {
	stats := d.Stats()
	d.logger.Println("\n" + "=".repeat(70))
	d.logger.Println("📊 Event Dispatcher Stats")
	d.logger.Println("=".repeat(70))

	totalListeners := 0
	for event, count := range stats {
		d.logger.Printf("  %s: %d listener(s)", event, count)
		totalListeners += count
	}

	d.logger.Printf("\nTotal Events: %d", len(stats))
	d.logger.Printf("Total Listeners: %d", totalListeners)
	d.logger.Println("=".repeat(70))
}

// -----------------------------------------------------------------------------
// String Utility (Go doesn't have String.repeat)
// -----------------------------------------------------------------------------

type repeatableString string

func (s repeatableString) repeat(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += string(s)
	}
	return result
}
