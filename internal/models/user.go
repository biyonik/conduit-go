// -----------------------------------------------------------------------------
// User Model
// -----------------------------------------------------------------------------
// Bu dosya, User modelini ve ilgili database işlemlerini içerir.
// Laravel'deki Eloquent Model'e benzer bir yapı sağlar.
// -----------------------------------------------------------------------------

package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/biyonik/conduit-go/pkg/auth"
	"github.com/biyonik/conduit-go/pkg/database"
)

// User, users tablosunu temsil eden modeldir.
type User struct {
	BaseModel                  // ID, CreatedAt, UpdatedAt, DeletedAt
	Name            string     `json:"name" db:"name"`
	Email           string     `json:"email" db:"email"`
	Password        string     `json:"-" db:"password"` // json:"-" = API'ye göndermez
	Status          string     `json:"status" db:"status"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	RememberToken   *string    `json:"-" db:"remember_token"`
}

// UserRepository, User model için database işlemlerini yönetir.
// Bu pattern "Repository Pattern" olarak bilinir ve business logic'i
// database logic'ten ayırır.
type UserRepository struct {
	db      *sql.DB
	grammar database.Grammar
}

// NewUserRepository, yeni bir UserRepository oluşturur.
func NewUserRepository(db *sql.DB, grammar database.Grammar) *UserRepository {
	return &UserRepository{
		db:      db,
		grammar: grammar,
	}
}

// newBuilder, repository için yeni bir QueryBuilder oluşturur.
func (r *UserRepository) newBuilder() *database.QueryBuilder {
	return database.NewBuilder(r.db, r.grammar)
}

// FindByID, ID'ye göre user bulur.
//
// Parametre:
//   - id: Kullanıcı ID'si
//
// Döndürür:
//   - *User: Bulunan kullanıcı
//   - error: Kullanıcı bulunamazsa sql.ErrNoRows, diğer hatalar için error
//
// Örnek:
//
//	user, err := userRepo.FindByID(123)
//	if err == sql.ErrNoRows {
//	    return errors.New("user not found")
//	}
func (r *UserRepository) FindByID(id int64) (*User, error) {
	var user User
	err := r.newBuilder().
		Table("users").
		Where("id", "=", id).
		Where("deleted_at", "IS", nil). // Soft delete check
		First(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// FindByEmail, email'e göre user bulur.
//
// Parametre:
//   - email: Kullanıcı email'i
//
// Döndürür:
//   - *User: Bulunan kullanıcı
//   - error: Kullanıcı bulunamazsa sql.ErrNoRows
//
// Kullanım:
// Login işleminde kullanıcıyı email ile bulmak için.
func (r *UserRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.newBuilder().
		Table("users").
		Where("email", "=", email).
		Where("deleted_at", "IS", nil).
		First(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAll, tüm kullanıcıları döndürür (pagination ile).
//
// Parametreler:
//   - page: Sayfa numarası (1'den başlar)
//   - perPage: Sayfa başına kayıt sayısı
//
// Döndürür:
//   - []User: Kullanıcı listesi
//   - error: Hata varsa
func (r *UserRepository) GetAll(page, perPage int) ([]User, error) {
	var users []User

	offset := (page - 1) * perPage

	err := r.newBuilder().
		Table("users").
		Where("deleted_at", "IS", nil).
		OrderBy("created_at", "DESC").
		Limit(perPage).
		Offset(offset).
		Get(&users)

	if err != nil {
		return nil, err
	}

	return users, nil
}

// Create, yeni bir kullanıcı oluşturur.
//
// Parametre:
//   - user: Oluşturulacak kullanıcı (Password hash'lenmeli!)
//
// Döndürür:
//   - int64: Oluşturulan kullanıcının ID'si
//   - error: Hata varsa
//
// Örnek:
//
//	user := &User{
//	    Name: "John Doe",
//	    Email: "john@example.com",
//	    Password: auth.MustHash("secret123"),
//	    Status: "active",
//	}
//	userID, err := userRepo.Create(user)
//
// Güvenlik Notu:
// Password mutlaka hash'lenmiş olmalıdır! Bu metod hash'leme yapmaz.
func (r *UserRepository) Create(user *User) (int64, error) {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	result, err := r.newBuilder().ExecInsert(map[string]interface{}{
		"name":       user.Name,
		"email":      user.Email,
		"password":   user.Password,
		"status":     user.Status,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Update, mevcut kullanıcıyı günceller.
//
// Parametre:
//   - user: Güncellenecek kullanıcı (ID dolu olmalı)
//
// Döndürür:
//   - error: Hata varsa
//
// Örnek:
//
//	user.Name = "Jane Doe"
//	err := userRepo.Update(user)
func (r *UserRepository) Update(user *User) error {
	user.UpdatedAt = time.Now()

	data := map[string]interface{}{
		"name":       user.Name,
		"email":      user.Email,
		"status":     user.Status,
		"updated_at": user.UpdatedAt,
	}

	// Password sadece değiştiyse güncelle
	if user.Password != "" {
		data["password"] = user.Password
	}

	// EmailVerifiedAt güncellendiyse
	if user.EmailVerifiedAt != nil {
		data["email_verified_at"] = user.EmailVerifiedAt
	}

	_, err := r.newBuilder().
		Table("users").
		Where("id", "=", user.ID).
		ExecUpdate(data)

	return err
}

// Delete, kullanıcıyı soft delete yapar.
//
// Soft Delete nedir?
// Kullanıcı fiziksel olarak silinmez, sadece deleted_at alanı set edilir.
// Bu sayede:
// - Audit trail korunur
// - İlişkili kayıtlar bozulmaz
// - Gerekirse kullanıcı geri yüklenebilir
//
// Parametre:
//   - id: Silinecek kullanıcının ID'si
//
// Döndürür:
//   - error: Hata varsa
func (r *UserRepository) Delete(id int64) error {
	_, err := r.newBuilder().
		Table("users").
		Where("id", "=", id).
		ExecUpdate(map[string]interface{}{
			"deleted_at": time.Now(),
		})

	return err
}

// ForceDelete, kullanıcıyı kalıcı olarak siler (hard delete).
//
// UYARI: Bu işlem geri alınamaz!
// Sadece şu durumlarda kullanılmalıdır:
// - GDPR/KVKK gereği kullanıcı verisini tamamen silmek gerekiyorsa
// - Test ortamında temizlik yapılıyorsa
func (r *UserRepository) ForceDelete(id int64) error {
	_, err := r.newBuilder().
		Table("users").
		Where("id", "=", id).
		ExecDelete()

	return err
}

// UpdatePassword, kullanıcının şifresini günceller.
//
// Parametreler:
//   - id: Kullanıcı ID'si
//   - newPassword: Yeni şifre (düz metin, bu metod hash'ler)
//
// Döndürür:
//   - error: Hata varsa
//
// Örnek:
//
//	err := userRepo.UpdatePassword(user.ID, "newSecret123")
func (r *UserRepository) UpdatePassword(id int64, newPassword string) error {
	// Şifreyi hash'le
	hashedPassword, err := auth.Hash(newPassword)
	if err != nil {
		return err
	}

	_, err = r.newBuilder().
		Table("users").
		Where("id", "=", id).
		ExecUpdate(map[string]interface{}{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		})

	return err
}

// VerifyEmail, kullanıcının email'ini doğrulanmış olarak işaretler.
//
// Parametre:
//   - id: Kullanıcı ID'si
//
// Döndürür:
//   - error: Hata varsa
func (r *UserRepository) VerifyEmail(id int64) error {
	now := time.Now()
	_, err := r.newBuilder().
		Table("users").
		Where("id", "=", id).
		ExecUpdate(map[string]interface{}{
			"email_verified_at": now,
			"updated_at":        now,
		})

	return err
}

// ExistsByEmail, verilen email'e sahip bir kullanıcı var mı kontrol eder.
//
// Parametre:
//   - email: Kontrol edilecek email
//
// Döndürür:
//   - bool: Email mevcutsa true
//   - error: Database hatası varsa
//
// Kullanım:
// Registration sırasında email'in unique olup olmadığını kontrol etmek için.
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	// var count int
	// TODO: Count() metodu eklendiğinde bu implementasyon güncellenecek

	// Geçici çözüm: FindByEmail ile kontrol et
	_, err := r.FindByEmail(email)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetID, auth.User interface implementasyonu için.
func (u *User) GetID() int64 {
	return u.ID
}

// GetEmail, auth.User interface implementasyonu için.
func (u *User) GetEmail() string {
	return u.Email
}

// GetRole, auth.User interface implementasyonu için.
// (Şu an basit implementasyon, ileride roles tablosu eklenecek)
func (u *User) GetRole() string {
	// TODO: Database'den user'ın role'ünü çek
	// Şimdilik status'a göre basit bir mapping
	if u.Email == "admin@conduit-go.local" {
		return "admin"
	}
	return "user"
}

// IsActive, kullanıcının aktif olup olmadığını kontrol eder.
func (u *User) IsActive() bool {
	return u.Status == "active"
}

// IsEmailVerified, kullanıcının email'ini doğruladı mı kontrol eder.
func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// CheckPassword, verilen şifreyi kullanıcının şifresi ile karşılaştırır.
func (u *User) CheckPassword(password string) bool {
	return auth.Check(password, u.Password)
}
