// pkg/container/container.go
package container

import (
	"fmt"
	"reflect"
	"sync"
)

// @author    Ahmet Altun
// @email     ahmet.altun60@gmail.com
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik

// Container, bağımlılıkları yöneten DI konteyneridir.
// Servisleri (hizmetleri) "tembel" (lazy) olarak yükler ve
// singleton (tekil) olarak saklar.
type Container struct {
	mu        sync.RWMutex
	factories map[reflect.Type]func(*Container) (any, error)
	instances map[reflect.Type]any
}

// New, yeni bir boş DI konteyneri oluşturur.
func New() *Container {
	return &Container{
		factories: make(map[reflect.Type]func(*Container) (any, error)),
		instances: make(map[reflect.Type]any),
	}
}

// Register, bir servisi konteynere kaydeder.
// Kayıt, bir "fabrika" (factory) fonksiyonu aracılığıyla yapılır.
// Bu fonksiyon, servis ilk kez 'Get' ile istendiğinde çalıştırılır.
//
// Örnek:
//
//	c.Register(func(c *Container) (*sql.DB, error) {
//	    cfg := c.Get(configType).(*Config)
//	    return database.Connect(cfg.DB.DSN)
//	})
func (c *Container) Register(provider any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Gelen 'provider'ın bir fonksiyon olduğunu doğrula
	providerType := reflect.TypeOf(provider)
	if providerType.Kind() != reflect.Func {
		panic(fmt.Sprintf("container: Register() parametresi bir fonksiyon olmalıdır, %T alındı", provider))
	}

	// Fonksiyonun bir 'Container' parametresi aldığını doğrula
	if providerType.NumIn() != 1 || providerType.In(0) != reflect.TypeOf(c) {
		panic("container: Register() fonksiyonu parametre olarak sadece *container.Container almalıdır")
	}

	// Fonksiyonun (any, error) döndüğünü doğrula
	if providerType.NumOut() != 2 || !providerType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("container: Register() fonksiyonu (any, error) döndürmelidir")
	}

	// Servisin tipini (ilk dönüş değeri) anahtar olarak kullan
	serviceType := providerType.Out(0)
	c.factories[serviceType] = provider.(func(*Container) (any, error))
}

// Get, bir servisi konteynerdan tipine göre çözer (resolve).
// Eğer servis daha önce çözüldüyse, mevcut (singleton) örnek döndürülür.
// Eğer çözülmediyse, fabrikası çalıştırılır, sonuç saklanır ve döndürülür.
func (c *Container) Get(serviceType reflect.Type) (any, error) {
	// Önce mevcut örnek var mı diye bak (hızlı yol)
	c.mu.RLock()
	instance, ok := c.instances[serviceType]
	c.mu.RUnlock()

	if ok {
		return instance, nil
	}

	// Mevcut örnek yok, yazma kilidi al
	c.mu.Lock()
	defer c.mu.Unlock()

	// Başka bir goroutine biz kilidi beklerken bu servisi oluşturmuş olabilir
	// (double-checked locking)
	instance, ok = c.instances[serviceType]
	if ok {
		return instance, nil
	}

	// Servis fabrikasını bul
	factory, ok := c.factories[serviceType]
	if !ok {
		return nil, fmt.Errorf("container: %s tipi için bir servis kaydı bulunamadı", serviceType)
	}

	// Fabrikayı çalıştırarak servisi oluştur
	instance, err := factory(c)
	if err != nil {
		return nil, fmt.Errorf("container: %s tipi oluşturulurken hata: %w", serviceType, err)
	}

	// Oluşturulan örneği (singleton) sakla
	c.instances[serviceType] = instance
	return instance, nil
}

// MustGet, 'Get' metodunu çağırır ama hata durumunda 'panic' yapar.
// Bu, uygulamanın başlatılması (bootstrap) sırasında, servislerin
// varlığından emin olduğumuzda kullanılır.
func (c *Container) MustGet(serviceType reflect.Type) any {
	instance, err := c.Get(serviceType)
	if err != nil {
		panic(err)
	}
	return instance
}
