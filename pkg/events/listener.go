// -----------------------------------------------------------------------------
// Event Listeners
// -----------------------------------------------------------------------------
// Bu dosya, event listener interface'ini ve yardımcı tipleri içerir.
//
// Listener Nedir?
// Listener, bir event gerçekleştiğinde çalışacak kod bloğudur.
// Event dispatch edildiğinde, o event'e kayıtlı tüm listener'lar çalıştırılır.
//
// Örnek:
//
//	// Listener tanımla
//	type SendWelcomeEmail struct {}
//
//	func (l *SendWelcomeEmail) Handle(event events.Event) error {
//	    user := event.Payload().(*models.User)
//	    return mailer.Send(user.Email, "Welcome!")
//	}
//
//	// Listener'ı kaydet
//	dispatcher.Listen("user.registered", &SendWelcomeEmail{})
// -----------------------------------------------------------------------------

package events

// Listener, event'leri dinleyen ve işleyen interface.
//
// Her listener, Handle() metodunu implement etmelidir.
// Handle metodu, event gerçekleştiğinde çağrılır.
type Listener interface {
	// Handle, event'i işler.
	//
	// Parametre:
	//   - event: Gerçekleşen event
	//
	// Döndürür:
	//   - error: İşlem başarısızsa hata döner
	//
	// Hata Yönetimi:
	// Handle metodu error dönerse, dispatcher bu hatayı loglar
	// ancak diğer listener'ların çalışmasını engellemez.
	Handle(event Event) error
}

// ListenerFunc, fonksiyonları Listener interface'ine çevirir.
//
// Bu adapter pattern sayesinde, struct tanımlamadan
// fonksiyon olarak listener yazabilirsiniz:
//
//	dispatcher.Listen("user.registered", events.ListenerFunc(func(e events.Event) error {
//	    user := e.Payload().(*models.User)
//	    log.Println("User registered:", user.Email)
//	    return nil
//	}))
type ListenerFunc func(Event) error

// Handle, ListenerFunc'ı Listener interface'ine uyumlu hale getirir.
func (f ListenerFunc) Handle(event Event) error {
	return f(event)
}

// -----------------------------------------------------------------------------
// Async Listener Wrapper
// -----------------------------------------------------------------------------

// AsyncListener, listener'ı goroutine'de çalıştıran wrapper.
//
// Kullanım:
//
//	// Senkron listener
//	slowListener := &SendEmailListener{}
//
//	// Asenkron hale getir
//	asyncListener := events.NewAsyncListener(slowListener, logger)
//	dispatcher.Listen("user.registered", asyncListener)
//
// Avantajlar:
// - Event dispatch hızlı döner (blocking olmaz)
// - Yavaş işlemler (email, API call) arka planda çalışır
// - Main flow bloke olmaz
type AsyncListener struct {
	listener Listener
	logger   Logger // Error logging için
}

// Logger, log interface'i (dependency injection için).
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// NewAsyncListener, yeni bir AsyncListener oluşturur.
//
// Parametreler:
//   - listener: Wrap edilecek listener
//   - logger: Hata loglamak için logger
//
// Döndürür:
//   - *AsyncListener: Async listener wrapper
func NewAsyncListener(listener Listener, logger Logger) *AsyncListener {
	return &AsyncListener{
		listener: listener,
		logger:   logger,
	}
}

// Handle, listener'ı goroutine'de çalıştırır.
//
// NOT: Goroutine'de çalıştığı için error dönmez (nil döner).
// Hatalar logger'a yazılır.
func (a *AsyncListener) Handle(event Event) error {
	// Goroutine'de çalıştır
	go func() {
		if err := a.listener.Handle(event); err != nil {
			a.logger.Printf("❌ Async listener error for event '%s': %v", event.Name(), err)
		}
	}()

	// Goroutine başlatıldı, hemen dön
	return nil
}

// -----------------------------------------------------------------------------
// Conditional Listener
// -----------------------------------------------------------------------------

// ConditionalListener, sadece belirli koşullarda çalışan listener.
//
// Kullanım:
//
//	listener := &SendEmailListener{}
//	condition := func(e events.Event) bool {
//	    user := e.Payload().(*models.User)
//	    return user.EmailVerified // Sadece verified user'lar için
//	}
//	conditionalListener := events.NewConditionalListener(listener, condition)
type ConditionalListener struct {
	listener  Listener
	condition func(Event) bool
}

// NewConditionalListener, yeni bir ConditionalListener oluşturur.
func NewConditionalListener(listener Listener, condition func(Event) bool) *ConditionalListener {
	return &ConditionalListener{
		listener:  listener,
		condition: condition,
	}
}

// Handle, koşul sağlanıyorsa listener'ı çalıştırır.
func (c *ConditionalListener) Handle(event Event) error {
	if c.condition(event) {
		return c.listener.Handle(event)
	}
	return nil // Koşul sağlanmadı, skip
}
