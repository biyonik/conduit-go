// -----------------------------------------------------------------------------
// Password Reset Controller
// -----------------------------------------------------------------------------
// Bu controller, şifre sıfırlama (forgot password) işlemlerini yönetir:
// - Forgot Password (Şifre sıfırlama isteği)
// - Reset Password (Yeni şifre belirleme)
//
// Akış:
// 1. Kullanıcı email girer (forgot password)
// 2. System email'e reset link gönderir (token içerir)
// 3. Kullanıcı linke tıklar
// 4. Yeni şifre girer (reset password)
// 5. Şifre güncellenir, token invalidate edilir
// -----------------------------------------------------------------------------

package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"reflect"
	"time"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/models"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
	"github.com/biyonik/conduit-go/pkg/validation"
	"github.com/biyonik/conduit-go/pkg/validation/types"
)

// PasswordResetToken, şifre sıfırlama token'larını temsil eder.
type PasswordResetToken struct {
	Email     string    `db:"email"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
}

// PasswordController, şifre yönetimi işlemlerini yönetir.
type PasswordController struct {
	Logger         *log.Logger
	DB             *sql.DB
	Grammar        database.Grammar
	UserRepository *models.UserRepository
}

// NewPasswordController, DI Container için factory function.
func NewPasswordController(c *container.Container) (*PasswordController, error) {
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	grammarType := reflect.TypeOf((*database.Grammar)(nil)).Elem()
	grammar := c.MustGet(grammarType).(database.Grammar)

	return &PasswordController{
		Logger:         logger,
		DB:             db,
		Grammar:        grammar,
		UserRepository: models.NewUserRepository(db, grammar),
	}, nil
}

// newBuilder, controller için yeni bir QueryBuilder oluşturur.
func (pc *PasswordController) newBuilder() *database.QueryBuilder {
	return database.NewBuilder(pc.DB, pc.Grammar)
}

// ForgotPassword, şifre sıfırlama isteği oluşturur.
//
// POST /api/auth/forgot-password
//
// Request Body:
//
//	{
//	  "email": "john@example.com"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "message": "Şifre sıfırlama linki email adresinize gönderildi"
//	  }
//	}
//
// Güvenlik Notu:
// Email bulunamasa bile aynı mesaj dönülür (user enumeration attack koruması).
// Kullanıcı hangi email'lerin sistemde olduğunu anlayamamalı.
func (pc *PasswordController) ForgotPassword(w http.ResponseWriter, r *conduitReq.Request) {
	pc.Logger.Println("🔑 Password reset request...")

	// 1. Request body'yi parse et
	var reqData struct {
		Email string `json:"email"`
	}

	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	// 2. Validation
	schema := validation.Make().Shape(map[string]validation.Type{
		"email": types.String().
			Required().
			Email().
			Label("Email").
			Trim(),
	})

	result := schema.Validate(map[string]any{
		"email": reqData.Email,
	})

	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return
	}

	email := result.ValidData()["email"].(string)

	// 3. Kullanıcıyı bul
	user, err := pc.UserRepository.FindByEmail(email)

	// GÜVENLIK: Email bulunamasa bile aynı mesajı dön
	if err == sql.ErrNoRows {
		pc.Logger.Printf("⚠️  Password reset requested for non-existent email: %s", email)
		// Yine de başarılı mesaj dön (user enumeration attack koruması)
		pc.sendSuccessResponse(w)
		return
	}

	if err != nil {
		pc.Logger.Printf("❌ Database error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	// 4. User aktif değilse işlem yapma
	if !user.IsActive() {
		pc.Logger.Printf("⚠️  Password reset requested for inactive user: %s", email)
		// Yine de başarılı mesaj dön
		pc.sendSuccessResponse(w)
		return
	}

	// 5. Reset token oluştur
	token, err := pc.generateResetToken()
	if err != nil {
		pc.Logger.Printf("❌ Token generation error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	// 6. Mevcut token'ları sil (aynı email için)
	_, _ = pc.newBuilder().
		Table("password_reset_tokens").
		Where("email", "=", email).
		ExecDelete()

	// 7. Yeni token'ı kaydet
	_, err = pc.newBuilder().ExecInsert(map[string]interface{}{
		"email":      email,
		"token":      pc.hashToken(token), // Token hash'lenmiş olarak saklanır
		"created_at": time.Now(),
	})

	if err != nil {
		pc.Logger.Printf("❌ Token save error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	// 8. Email gönder (Phase 3'te implement edilecek)
	pc.Logger.Printf("✅ Password reset token created for: %s", email)
	pc.Logger.Printf("🔗 Reset link: http://localhost:3000/reset-password?token=%s", token)

	// TODO (Phase 3): Mail sistemini kullanarak reset link gönder
	// mail.To(email).
	//     Subject("Şifre Sıfırlama").
	//     Template("password-reset", map[string]string{
	//         "name": user.Name,
	//         "link": "http://localhost:3000/reset-password?token=" + token,
	//     }).
	//     Send()

	pc.sendSuccessResponse(w)
}

// ResetPassword, şifre sıfırlama işlemini tamamlar.
//
// POST /api/auth/reset-password
//
// Request Body:
//
//	{
//	  "token": "abc123...",
//	  "email": "john@example.com",
//	  "password": "NewSecret123!",
//	  "password_confirm": "NewSecret123!"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "message": "Şifreniz başarıyla değiştirildi"
//	  }
//	}
//
// Response (422 Invalid Token):
//
//	{
//	  "success": false,
//	  "error": "Geçersiz veya süresi dolmuş token"
//	}
func (pc *PasswordController) ResetPassword(w http.ResponseWriter, r *conduitReq.Request) {
	pc.Logger.Println("🔄 Password reset attempt...")

	// 1. Request body'yi parse et
	var reqData struct {
		Token           string `json:"token"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		PasswordConfirm string `json:"password_confirm"`
	}

	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	// 2. Validation
	schema := validation.Make().Shape(map[string]validation.Type{
		"token": types.String().
			Required().
			Min(32).
			Label("Token"),

		"email": types.String().
			Required().
			Email().
			Label("Email").
			Trim(),

		"password": types.String().
			Required().
			Password(
				types.WithMinLength(8),
				types.WithRequireUppercase(true),
				types.WithRequireLowercase(true),
				types.WithRequireNumeric(true),
				types.WithRequireSpecial(true),
			).
			Label("Şifre"),

		"password_confirm": types.String().
			Required().
			Label("Şifre Tekrar"),
	}).CrossValidate(func(data map[string]any) error {
		password, _ := data["password"].(string)
		confirm, _ := data["password_confirm"].(string)
		if password != confirm {
			return validation.NewFieldError("password_confirm", "Şifreler eşleşmiyor")
		}
		return nil
	})

	result := schema.Validate(map[string]any{
		"token":            reqData.Token,
		"email":            reqData.Email,
		"password":         reqData.Password,
		"password_confirm": reqData.PasswordConfirm,
	})

	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return
	}

	validData := result.ValidData()

	// 3. Token'ı doğrula
	var resetToken PasswordResetToken
	err := pc.newBuilder().
		Table("password_reset_tokens").
		Where("email", "=", validData["email"]).
		Where("token", "=", pc.hashToken(validData["token"].(string))).
		First(&resetToken)

	if err == sql.ErrNoRows {
		pc.Logger.Printf("⚠️  Invalid reset token for email: %s", validData["email"])
		conduitRes.Error(w, 422, "Geçersiz veya süresi dolmuş token")
		return
	}

	if err != nil {
		pc.Logger.Printf("❌ Database error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	// 4. Token expire kontrolü (1 saat geçerli)
	if time.Since(resetToken.CreatedAt) > 1*time.Hour {
		pc.Logger.Printf("⚠️  Expired reset token for email: %s", validData["email"])
		conduitRes.Error(w, 422, "Token süresi dolmuş. Lütfen yeni bir şifre sıfırlama isteği oluşturun.")
		return
	}

	// 5. Kullanıcıyı bul
	user, err := pc.UserRepository.FindByEmail(validData["email"].(string))
	if err != nil {
		pc.Logger.Printf("❌ User not found: %v", err)
		conduitRes.Error(w, 404, "Kullanıcı bulunamadı")
		return
	}

	// 6. Şifreyi güncelle
	if err := pc.UserRepository.UpdatePassword(user.ID, validData["password"].(string)); err != nil {
		pc.Logger.Printf("❌ Password update error: %v", err)
		conduitRes.Error(w, 500, "Şifre güncellenemedi")
		return
	}

	// 7. Token'ı sil (tek kullanımlık)
	_, _ = pc.newBuilder().
		Table("password_reset_tokens").
		Where("email", "=", validData["email"]).
		ExecDelete()

	pc.Logger.Printf("✅ Password reset successful for: %s", user.Email)

	response := map[string]string{
		"message": "Şifreniz başarıyla değiştirildi. Artık yeni şifrenizle giriş yapabilirsiniz.",
	}

	conduitRes.Success(w, 200, response, nil)
}

// generateResetToken, güvenli bir reset token oluşturur.
func (pc *PasswordController) generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// hashToken, token'ı hash'ler (database'de plain text saklamayalım).
func (pc *PasswordController) hashToken(token string) string {
	// Basit bir hash (SHA256 kullanılabilir)
	// Şimdilik token'ı olduğu gibi dönüyoruz
	// TODO: crypto/sha256 kullanarak hash'le
	return token
}

// sendSuccessResponse, standart başarı mesajı döner.
func (pc *PasswordController) sendSuccessResponse(w http.ResponseWriter) {
	response := map[string]string{
		"message": "Eğer bu email adresi sistemimizde kayıtlıysa, şifre sıfırlama linki gönderildi.",
	}
	conduitRes.Success(w, 200, response, nil)
}
