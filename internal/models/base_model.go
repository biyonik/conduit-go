// internal/models/base_model.go
//
// Bu dosya, tüm modellerin kalıtım yoluyla devraldığı temel alanları
// (ID, CreatedAt, UpdatedAt) ve davranışları içerir.
//
// Laravel'deki `Model.php` dosyasının sade ve Go’ya uyarlanmış
// karşılığı olarak düşünülebilir.
//
// BaseModel’in amacı:
//
// - Tekrarlayan alanların merkezi bir yerde tanımlanması,
// - Modellerin ortak davranışlara sahip olması,
// - ORM tarafında standart bir base yapının oluşması,
// - Genişletilebilir bir üst sınıf mimarisinin sağlanmasıdır.
//
// Kullanım:
//    type User struct {
//        models.BaseModel
//        Name string
//        Email string
//    }
//
// Bu sayede User modeli otomatik olarak ID, CreatedAt, UpdatedAt alanlarına sahip olur.

package models

import "time"

// BaseModel
//
// Tüm modellerin gövdesini oluşturur. Timestamp yönetimi dahildir.
//
// Alanlar:
//   - ID:        int64  → birincil anahtar
//   - CreatedAt: time   → oluşturulma zamanı
//   - UpdatedAt: time   → güncellenme zamanı
type BaseModel struct {
	ID        int64     `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Initialize
//
// CreatedAt ve UpdatedAt alanlarını şu anki zamana ayarlar.
// Yeni bir kayıt oluşturulmadan önce çağrılır.
func (m *BaseModel) Initialize() {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
}

// Touch
//
// UpdatedAt alanını şu anki zamana günceller.
// Genelde kaydın değiştirilmesi durumunda otomatik tetiklenir.
func (m *BaseModel) Touch() {
	m.UpdatedAt = time.Now()
}
