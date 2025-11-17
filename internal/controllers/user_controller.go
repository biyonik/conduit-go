// -----------------------------------------------------------------------------
// Authentication Controller
// -----------------------------------------------------------------------------
// Bu controller, kullanıcı authentication işlemlerini yönetir:
// - Register (Kayıt)
// - Login (Giriş)
// - Logout (Çıkış)
// - Refresh Token (Token yenileme)
// - Profile (Profil bilgisi)
//
// Laravel'deki AuthController'a benzer bir yapı sağlar.
// -----------------------------------------------------------------------------

package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"reflect"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/models"
	"github.com/biyonik/conduit-go/pkg/auth"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/database"
	"github.com/biyonik/conduit-go/pkg/validation"
	"github.com/biyonik/conduit-go/pkg/validation/types"
)

// AuthController, authentication işlemlerini yönetir.
type AuthController struct {
	Logger         *log.Logger
	UserRepository *models.UserRepository
	JWTConfig      *auth.JWTConfig
}

// NewAuthController, DI Container için factory function.
func NewAuthController(c *container.Container) (*AuthController, error) {
	logger := c.MustGet(reflect.TypeOf((*log.Logger)(nil))).(*log.Logger)
	db := c.MustGet(reflect.TypeOf((*sql.DB)(nil))).(*sql.DB)
	grammarType := reflect.TypeOf((*database.Grammar)(nil)).Elem()
	grammar := c.MustGet(grammarType).(database.Grammar)

	return &AuthController{
		Logger:         logger,
		UserRepository: models.NewUserRepository(db, grammar),
		JWTConfig:      auth.DefaultJWTConfig(),
	}, nil
}

// RegisterRequest, registration validation için schema.
type RegisterRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

// Register, yeni kullanıcı kaydı yapar.
//
// POST /api/auth/register
//
// Request Body:
//
//	{
//	  "name": "John Doe",
//	  "email": "john@example.com",
//	  "password": "Secret123!",
//	  "password_confirm": "Secret123!"
//	}
//
// Response (201 Created):
//
//	{
//	  "success": true,
//	  "data": {
//	    "user": {
//	      "id": 123,
//	      "name": "John Doe",
//	      "email": "john@example.com",
//	      "status": "active"
//	    },
//	    "access_token": "eyJhbGc...",
//	    "refresh_token": "eyJhbGc...",
//	    "expires_in": 3600
//	  }
//	}
//
// Response (422 Validation Error):
//
//	{
//	  "success": false,
//	  "error": "Doğrulama hatası",
//	  "data": {
//	    "email": ["Email zaten kullanımda"]
//	  }
//	}
func (ac *AuthController) Register(w http.ResponseWriter, r *conduitReq.Request) {
	ac.Logger.Println("📝 User registration attempt...")

	// 1. Request body'yi parse et
	var reqData RegisterRequest
	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	// 2. Validation schema oluştur
	schema := validation.Make().Shape(map[string]validation.Type{
		"name": types.String().
			Required().
			Min(2).
			Max(255).
			Label("Ad Soyad"),

		"email": types.String().
			Required().
			Email().
			Max(255).
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
		// Şifrelerin eşleşip eşleşmediğini kontrol et
		password, _ := data["password"].(string)
		confirm, _ := data["password_confirm"].(string)
		if password != confirm {
			return validation.NewFieldError("password_confirm", "Şifreler eşleşmiyor")
		}
		return nil
	})

	// 3. Validation yap
	dataMap := map[string]any{
		"name":             reqData.Name,
		"email":            reqData.Email,
		"password":         reqData.Password,
		"password_confirm": reqData.PasswordConfirm,
	}

	result := schema.Validate(dataMap)
	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return
	}

	validData := result.ValidData()

	// 4. Email'in unique olup olmadığını kontrol et
	exists, err := ac.UserRepository.ExistsByEmail(validData["email"].(string))
	if err != nil {
		ac.Logger.Printf("❌ Database error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	if exists {
		conduitRes.Error(w, 422, map[string][]string{
			"email": {"Bu email adresi zaten kullanımda"},
		})
		return
	}

	// 5. Şifreyi hash'le
	hashedPassword, err := auth.Hash(validData["password"].(string))
	if err != nil {
		ac.Logger.Printf("❌ Password hashing error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	// 6. User oluştur
	user := &models.User{
		Name:     validData["name"].(string),
		Email:    validData["email"].(string),
		Password: hashedPassword,
		Status:   "active",
	}

	userID, err := ac.UserRepository.Create(user)
	if err != nil {
		ac.Logger.Printf("❌ User creation error: %v", err)
		conduitRes.Error(w, 500, "Kullanıcı oluşturulamadı")
		return
	}

	user.ID = userID

	// 7. JWT token'lar oluştur
	accessToken, err := auth.GenerateToken(user.ID, user.Email, user.GetRole(), ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("❌ Token generation error: %v", err)
		conduitRes.Error(w, 500, "Token oluşturulamadı")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("❌ Refresh token generation error: %v", err)
		conduitRes.Error(w, 500, "Token oluşturulamadı")
		return
	}

	// 8. Response hazırla
	ac.Logger.Printf("✅ User registered successfully: %s (ID: %d)", user.Email, user.ID)

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"status":     user.Status,
			"created_at": user.CreatedAt,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(ac.JWTConfig.ExpirationTime.Seconds()),
	}

	conduitRes.Success(w, 201, response, nil)
}

// LoginRequest, login validation için schema.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login, kullanıcı girişi yapar.
//
// POST /api/auth/login
//
// Request Body:
//
//	{
//	  "email": "john@example.com",
//	  "password": "Secret123!"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "user": {...},
//	    "access_token": "eyJhbGc...",
//	    "refresh_token": "eyJhbGc...",
//	    "expires_in": 3600
//	  }
//	}
//
// Response (401 Unauthorized):
//
//	{
//	  "success": false,
//	  "error": "Email veya şifre hatalı"
//	}
func (ac *AuthController) Login(w http.ResponseWriter, r *conduitReq.Request) {
	ac.Logger.Println("🔐 Login attempt...")

	// 1. Request body'yi parse et
	var reqData LoginRequest
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

		"password": types.String().
			Required().
			Min(1).
			Label("Şifre"),
	})

	dataMap := map[string]any{
		"email":    reqData.Email,
		"password": reqData.Password,
	}

	result := schema.Validate(dataMap)
	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return
	}

	validData := result.ValidData()

	// 3. Kullanıcıyı email ile bul
	user, err := ac.UserRepository.FindByEmail(validData["email"].(string))
	if err == sql.ErrNoRows {
		// Güvenlik: Email var mı yok mu belli etme (timing attack koruması)
		ac.Logger.Printf("⚠️  Login failed: User not found (%s)", validData["email"])
		conduitRes.Error(w, 401, "Email veya şifre hatalı")
		return
	}

	if err != nil {
		ac.Logger.Printf("❌ Database error: %v", err)
		conduitRes.Error(w, 500, "Sunucu hatası")
		return
	}

	// 4. Şifreyi kontrol et
	if !user.CheckPassword(validData["password"].(string)) {
		ac.Logger.Printf("⚠️  Login failed: Invalid password (%s)", user.Email)
		conduitRes.Error(w, 401, "Email veya şifre hatalı")
		return
	}

	// 5. Kullanıcı aktif mi kontrol et
	if !user.IsActive() {
		ac.Logger.Printf("⚠️  Login failed: User inactive (%s)", user.Email)
		conduitRes.Error(w, 403, "Hesabınız aktif değil. Lütfen yönetici ile iletişime geçin.")
		return
	}

	// 6. Şifre hash'i güncellenmeye ihtiyaç duyuyor mu kontrol et
	// (Güvenlik: Zaman içinde hash cost artırılabilir)
	if auth.NeedsRehash(user.Password) {
		newHash, _ := auth.Hash(validData["password"].(string))
		if newHash != "" {
			user.Password = newHash
			ac.UserRepository.Update(user)
			ac.Logger.Printf("🔄 Password hash updated for user: %s", user.Email)
		}
	}

	// 7. JWT token'lar oluştur
	accessToken, err := auth.GenerateToken(user.ID, user.Email, user.GetRole(), ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("❌ Token generation error: %v", err)
		conduitRes.Error(w, 500, "Token oluşturulamadı")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("❌ Refresh token generation error: %v", err)
		conduitRes.Error(w, 500, "Token oluşturulamadı")
		return
	}

	// 8. Response hazırla
	ac.Logger.Printf("✅ User logged in successfully: %s (ID: %d)", user.Email, user.ID)

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":                user.ID,
			"name":              user.Name,
			"email":             user.Email,
			"status":            user.Status,
			"role":              user.GetRole(),
			"email_verified_at": user.EmailVerifiedAt,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(ac.JWTConfig.ExpirationTime.Seconds()),
	}

	conduitRes.Success(w, 200, response, nil)
}

// Logout, kullanıcıyı çıkış yapar.
//
// POST /api/auth/logout
// Authorization: Bearer {token}
//
// JWT stateless olduğu için server tarafında bir şey yapmaya gerek yok.
// Client token'ı silmeli. İleride token blacklist eklenebilir.
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "message": "Çıkış başarılı"
//	  }
//	}
func (ac *AuthController) Logout(w http.ResponseWriter, r *conduitReq.Request) {
	// Context'ten user bilgisini al (middleware tarafından set edilmiş)
	user := r.Context().Value("user")
	if user != nil {
		if authUser, ok := user.(auth.User); ok {
			ac.Logger.Printf("👋 User logged out: %s", authUser.GetEmail())
		}
	}

	// TODO (Phase 3): Token blacklist'e ekle (Redis)
	// tokenBlacklist.Add(token, expirationTime)

	response := map[string]string{
		"message": "Çıkış başarılı",
	}

	conduitRes.Success(w, 200, response, nil)
}

// RefreshToken, access token'ı yeniler.
//
// POST /api/auth/refresh
//
// Request Body:
//
//	{
//	  "refresh_token": "eyJhbGc..."
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "access_token": "eyJhbGc...",
//	    "refresh_token": "eyJhbGc...", // Yeni refresh token (rotation)
//	    "expires_in": 3600
//	  }
//	}
//
// Güvenlik: Refresh Token Rotation
// Her refresh token kullanıldığında yeni bir refresh token oluşturulur.
// Bu sayede çalınan token'ların kullanımı minimize edilir.
func (ac *AuthController) RefreshToken(w http.ResponseWriter, r *conduitReq.Request) {
	ac.Logger.Println("🔄 Token refresh attempt...")

	// 1. Request body'den refresh token'ı al
	var reqData struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	if reqData.RefreshToken == "" {
		conduitRes.Error(w, 400, "Refresh token gerekli")
		return
	}

	// 2. Refresh token'ı parse et
	claims, err := auth.ParseToken(reqData.RefreshToken, ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("⚠️  Invalid refresh token: %v", err)
		conduitRes.Error(w, 401, "Geçersiz veya süresi dolmuş refresh token")
		return
	}

	// 3. Token'ın refresh token olduğunu kontrol et
	if claims.Role != "refresh" {
		ac.Logger.Printf("⚠️  Token is not a refresh token")
		conduitRes.Error(w, 401, "Geçersiz token tipi")
		return
	}

	// 4. Kullanıcıyı database'den al (token'da user bilgisi olabilir ama güncel olmayabilir)
	user, err := ac.UserRepository.FindByID(claims.UserID)
	if err != nil {
		ac.Logger.Printf("⚠️  User not found: %v", err)
		conduitRes.Error(w, 401, "Kullanıcı bulunamadı")
		return
	}

	// 5. Kullanıcı aktif mi kontrol et
	if !user.IsActive() {
		conduitRes.Error(w, 403, "Hesabınız aktif değil")
		return
	}

	// 6. Yeni token'lar oluştur (hem access hem refresh - rotation)
	newAccessToken, err := auth.GenerateToken(user.ID, user.Email, user.GetRole(), ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("❌ Token generation error: %v", err)
		conduitRes.Error(w, 500, "Token oluşturulamadı")
		return
	}

	newRefreshToken, err := auth.GenerateRefreshToken(user.ID, user.Email, ac.JWTConfig)
	if err != nil {
		ac.Logger.Printf("❌ Refresh token generation error: %v", err)
		conduitRes.Error(w, 500, "Token oluşturulamadı")
		return
	}

	// 7. Response hazırla
	ac.Logger.Printf("✅ Token refreshed for user: %s (ID: %d)", user.Email, user.ID)

	// TODO (Phase 3): Eski refresh token'ı blacklist'e ekle

	response := map[string]interface{}{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(ac.JWTConfig.ExpirationTime.Seconds()),
	}

	conduitRes.Success(w, 200, response, nil)
}

// Profile, authenticated user'ın profil bilgilerini döndürür.
//
// GET /api/auth/profile
// Authorization: Bearer {token}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "id": 123,
//	    "name": "John Doe",
//	    "email": "john@example.com",
//	    "role": "user",
//	    "status": "active",
//	    "email_verified_at": "2024-01-15T10:30:00Z",
//	    "created_at": "2024-01-01T10:00:00Z"
//	  }
//	}
func (ac *AuthController) Profile(w http.ResponseWriter, r *conduitReq.Request) {
	// Context'ten user'ı al (Auth middleware tarafından set edilmiş)
	contextUser := r.Context().Value("user")
	if contextUser == nil {
		conduitRes.Error(w, 401, "Unauthorized")
		return
	}

	authUser, ok := contextUser.(auth.User)
	if !ok {
		conduitRes.Error(w, 401, "Unauthorized")
		return
	}

	// Database'den tam user bilgisini çek (context'teki minimal bilgi)
	user, err := ac.UserRepository.FindByID(authUser.GetID())
	if err != nil {
		ac.Logger.Printf("❌ User not found: %v", err)
		conduitRes.Error(w, 404, "Kullanıcı bulunamadı")
		return
	}

	response := map[string]interface{}{
		"id":                user.ID,
		"name":              user.Name,
		"email":             user.Email,
		"role":              user.GetRole(),
		"status":            user.Status,
		"email_verified_at": user.EmailVerifiedAt,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
	}

	conduitRes.Success(w, 200, response, nil)
}

// UpdateProfile, authenticated user'ın profil bilgilerini günceller.
//
// PUT /api/auth/profile
// Authorization: Bearer {token}
//
// Request Body:
//
//	{
//	  "name": "Jane Doe"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "message": "Profil güncellendi",
//	    "user": {...}
//	  }
//	}
func (ac *AuthController) UpdateProfile(w http.ResponseWriter, r *conduitReq.Request) {
	// Context'ten user'ı al
	contextUser := r.Context().Value("user")
	if contextUser == nil {
		conduitRes.Error(w, 401, "Unauthorized")
		return
	}

	authUser, ok := contextUser.(auth.User)
	if !ok {
		conduitRes.Error(w, 401, "Unauthorized")
		return
	}

	// 1. Request body'yi parse et
	var reqData struct {
		Name string `json:"name"`
	}

	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	// 2. Validation
	schema := validation.Make().Shape(map[string]validation.Type{
		"name": types.String().
			Required().
			Min(2).
			Max(255).
			Label("Ad Soyad"),
	})

	result := schema.Validate(map[string]any{
		"name": reqData.Name,
	})

	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return
	}

	// 3. User'ı database'den çek
	user, err := ac.UserRepository.FindByID(authUser.GetID())
	if err != nil {
		conduitRes.Error(w, 404, "Kullanıcı bulunamadı")
		return
	}

	// 4. Güncelle
	user.Name = result.ValidData()["name"].(string)
	if err := ac.UserRepository.Update(user); err != nil {
		ac.Logger.Printf("❌ Profile update error: %v", err)
		conduitRes.Error(w, 500, "Profil güncellenemedi")
		return
	}

	ac.Logger.Printf("✅ Profile updated: %s (ID: %d)", user.Email, user.ID)

	response := map[string]interface{}{
		"message": "Profil başarıyla güncellendi",
		"user": map[string]interface{}{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"updated_at": user.UpdatedAt,
		},
	}

	conduitRes.Success(w, 200, response, nil)
}

// ChangePassword, authenticated user'ın şifresini değiştirir.
//
// PUT /api/auth/password
// Authorization: Bearer {token}
//
// Request Body:
//
//	{
//	  "current_password": "OldSecret123!",
//	  "new_password": "NewSecret456!",
//	  "new_password_confirm": "NewSecret456!"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "message": "Şifre değiştirildi"
//	  }
//	}
func (ac *AuthController) ChangePassword(w http.ResponseWriter, r *conduitReq.Request) {
	// Context'ten user'ı al
	contextUser := r.Context().Value("user")
	if contextUser == nil {
		conduitRes.Error(w, 401, "Unauthorized")
		return
	}

	authUser, ok := contextUser.(auth.User)
	if !ok {
		conduitRes.Error(w, 401, "Unauthorized")
		return
	}

	// 1. Request body'yi parse et
	var reqData struct {
		CurrentPassword    string `json:"current_password"`
		NewPassword        string `json:"new_password"`
		NewPasswordConfirm string `json:"new_password_confirm"`
	}

	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	// 2. Validation
	schema := validation.Make().Shape(map[string]validation.Type{
		"current_password": types.String().
			Required().
			Label("Mevcut Şifre"),

		"new_password": types.String().
			Required().
			Password(
				types.WithMinLength(8),
				types.WithRequireUppercase(true),
				types.WithRequireLowercase(true),
				types.WithRequireNumeric(true),
				types.WithRequireSpecial(true),
			).
			Label("Yeni Şifre"),

		"new_password_confirm": types.String().
			Required().
			Label("Yeni Şifre Tekrar"),
	}).CrossValidate(func(data map[string]any) error {
		newPass, _ := data["new_password"].(string)
		confirm, _ := data["new_password_confirm"].(string)
		if newPass != confirm {
			return validation.NewFieldError("new_password_confirm", "Şifreler eşleşmiyor")
		}
		return nil
	})

	result := schema.Validate(map[string]any{
		"current_password":     reqData.CurrentPassword,
		"new_password":         reqData.NewPassword,
		"new_password_confirm": reqData.NewPasswordConfirm,
	})

	if result.HasErrors() {
		conduitRes.Error(w, 422, result.Errors())
		return
	}

	validData := result.ValidData()

	// 3. User'ı database'den çek
	user, err := ac.UserRepository.FindByID(authUser.GetID())
	if err != nil {
		conduitRes.Error(w, 404, "Kullanıcı bulunamadı")
		return
	}

	// 4. Mevcut şifreyi kontrol et
	if !user.CheckPassword(validData["current_password"].(string)) {
		conduitRes.Error(w, 401, "Mevcut şifre hatalı")
		return
	}

	// 5. Yeni şifreyi güncelle
	if err := ac.UserRepository.UpdatePassword(user.ID, validData["new_password"].(string)); err != nil {
		ac.Logger.Printf("❌ Password update error: %v", err)
		conduitRes.Error(w, 500, "Şifre güncellenemedi")
		return
	}

	ac.Logger.Printf("✅ Password changed: %s (ID: %d)", user.Email, user.ID)

	response := map[string]string{
		"message": "Şifre başarıyla değiştirildi",
	}

	conduitRes.Success(w, 200, response, nil)
}
