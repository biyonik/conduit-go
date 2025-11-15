# Go Schema Validator

[![Go Report Card](https://goreportcard.com/badge/github.com/biyonik/go-schema)](https://goreportcard.com/report/github.com/biyonik/go-schema)
[![Go.Dev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/biyonik/go-schema)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A type-safe, chainable, Zod-inspired validation library for Go, built with zero external dependencies.
<br>
Go için Zod'dan ilham alan, tip-güvenli, zincirlenebilir, sıfır dış bağımlılığa sahip bir doğrulama kütüphanesi.

---

## 🇬🇧 English

### Features

* **Fluent API:** Chain methods just like you would in Zod or Joi.
* **Type-Safe:** Provides distinct types for `String`, `Number`, `Boolean`, `Date`, `Object`, and `Array`.
* **Rich Rules:** Built-in support for complex rules like `Password`, `Email`, `IP`, `Phone`, `IBAN`, and `UUID`.
* **Sanitization & Transformation:** Cleanse data *before* validation using methods like `Trim()`, `StripTags()`, `EscapeHTML()`, and `SanitizeFilename()`.
* **Conditional Validation:** Apply rules dynamically with the powerful `.When()` method.
* **Cross-Field Validation:** Validate fields against each other using `.CrossValidate()`.
* **Zero Dependencies:** Built using only the Go standard library.

### Installation

```bash
go get [github.com/biyonik/conduit-go/pkg/validation](https://github.com/biyonik/conduit-go/pkg/validation)
```


### Basic Usage

Define your schema, validate your data, and get clean, type-safe results.

```go
import (
    v "[github.com/biyonik/conduit-go/pkg/validation](https://github.com/biyonik/conduit-go/pkg/validation)"
    "[github.com/biyonik/conduit-go/pkg/validation/types](https://github.com/biyonik/conduit-go/pkg/validation/types)"
    "fmt"
)

func main() {
    // 1. Define the schema
    userSchema := v.Make().Shape(map[string]v.Type{
        "name": types.String().Required().Min(3).Label("Full Name").Trim(),
        "email": types.String().Required().Email().Label("Email Address").Trim(),
        "age": types.Number().Min(18).Integer().Label("Age"),
        "role": types.String().OneOf("user", "admin", "editor").Default("user"),
    })

    // 2. Prepare your raw data
    data := map[string]any{
        "name":  "  Ahmet Altun ", // Will be trimmed
        "email": "ahmet@example.com",
        "age":   30,
    }

    // 3. Validate
    result := userSchema.Validate(data)

    // 4. Check results
    if result.HasErrors() {
        fmt.Println("Validation failed:")
        for field, errors := range result.Errors() {
            for _, err := range errors {
                fmt.Printf("- %s: %s\n", field, err)
            }
        }
    } else {
        fmt.Println("Validation successful!")
        // Get the sanitized and validated data
        validData := result.ValidData()
        fmt.Printf("Welcome, %s! Your role is: %s\n", validData["name"], validData["role"])
        // Output: Welcome, Ahmet Altun! Your role is: user
    }
}
```

### Available Types

#### `types.String()`
```go
types.String().
    Required().
    Min(5).
    Max(100).
    Email().
    URL().
    OneOf("admin", "user").
    Password(
        rules.WithMinLength(10),
        rules.WithRequireUppercase(true),
        rules.WithRequireNumeric(true),
    ).
    IP(4) // Requires IPv4
    Phone("TR") // Requires Turkish phone number
```

#### `types.Number()`
```go
types.Number().
    Required().
    Min(0).
    Max(100).
    Integer() // Must be a whole number
```

#### `types.Boolean()`
```go
types.Boolean().
    Required().
    Default(false)
```

#### `types.Date()`
Uses Go's standard time layout `2006-01-02` as default.
```go
types.Date().
    Required().
    Format("02/01/2006"). // Custom format
    Min("01/01/2020").
    Max("31/12/2025")
```

#### `types.Array()`
```go
types.Array().
    Required().
    Min(1). // Minimum 1 element
    Max(5). // Maximum 5 elements
    Elements( // Validate each element in the array
        types.String().Required().Email(),
    )
```

#### `types.Object()`
```go
types.Object().
    Required().
    Shape(map[string]v.Type{
        "street": types.String().Required(),
        "city":   types.String().Required(),
        "zip":    types.String().Required().Min(5).Max(5),
    })
```

### Specialized Types

```go
// UUID Validation
types.Uuid().
    Required().
    Version(4) // Require UUIDv4

// IBAN Validation
types.Iban().
    Required().
    Country("TR") // Validate for a specific country

// Credit Card Validation
types.CreditCard().
    Required().
    Type("visa") // "visa", "mastercard", etc.
```

### Advanced String (Sanitization)

`AdvancedString` inherits all `String` rules and adds powerful sanitization and filtering.

```go
types.AdvancedString().
    // Transformation (Sanitization)
    Trim().
    StripTags().
    EscapeHTML().
    SanitizeFilename().
    FilterEmoji(true). // Remove emojis

    // Validation (Filtering)
    CharSet("alphanumeric"). // Only allow a-z, A-Z, 0-9
    Domain(true).            // Must be a valid domain (subdomain allowed)
    TurkishChars(false)      // Must not contain TR chars
```

### Advanced Validation

#### Cross-Field Validation
Use `.CrossValidate()` to validate fields against each other. The callback runs *only* if all individual fields are already valid.

```go
passwordSchema := v.Make().Shape(map[string]v.Type{
    "password":        types.String().Min(8).Label("Password"),
    "passwordConfirm": types.String().Label("Confirm Password"),
}).CrossValidate(func(data map[string]any) error {
    pass, _ := data["password"].(string)
    confirm, _ := data["passwordConfirm"].(string)

    if pass != confirm {
        // This error is added to the "_cross_validation" field
        return fmt.Errorf("Passwords do not match")
    }
    return nil
})
```

#### Conditional Validation
Use `.When()` to apply schemas dynamically based on another field's value.

```go
paymentSchema := v.Make().Shape(map[string]v.Type{
    "paymentType": types.String().Required().OneOf("credit_card", "bank_transfer"),
    "cardNumber":  types.CreditCard(), // Not required by default
    "expiryDate":  types.Date().Format("01/06"), // Not required by default
    
}).When("paymentType", "credit_card", func() v.Schema {
    // This schema is ONLY applied if paymentType == "credit_card"
    return v.Make().Shape(map[string]v.Type{
        "cardNumber": types.CreditCard().Required().Label("Card Number"),
        "expiryDate": types.Date().Format("01/06").Required().Label("Expiry Date"),
    })
})
```

---

## 🇹🇷 Türkçe

### Özellikler

* **Akıcı API:** Zod veya Joi'de olduğu gibi metotları zincirleyin.
* **Tip Güvenliği:** `String`, `Number`, `Boolean`, `Date`, `Object`, ve `Array` için ayrı tipler sağlar.
* **Zengin Kurallar:** `Password`, `Email`, `IP`, `Phone`, `IBAN`, ve `UUID` gibi karmaşık kurallar için yerleşik destek.
* **Temizleme (Sanitization) & Dönüşüm:** `Trim()`, `StripTags()`, `EscapeHTML()` ve `SanitizeFilename()` gibi metotlarla veriyi doğrulamadan *önce* temizleyin.
* **Koşullu Doğrulama:** Güçlü `.When()` metodu ile kuralları dinamik olarak uygulayın.
* **Çapraz Alan Doğrulaması:** `.CrossValidate()` kullanarak alanları birbirine karşı doğrulayın.
* **Sıfır Bağımlılık:** Yalnızca Go standart kütüphanesi kullanılarak oluşturulmuştur.

### Kurulum

```bash
go get [github.com/biyonik/conduit-go/pkg/validation](https://github.com/biyonik/conduit-go/pkg/validation)
```

### Temel Kullanım

Şemanızı tanımlayın, verinizi doğrulayın ve temiz, tip-güvenli sonuçlar alın.

```go
import (
    v "[github.com/biyonik/conduit-go/pkg/validation](https://github.com/biyonik/conduit-go/pkg/validation)"
    "[github.com/biyonik/conduit-go/pkg/validation/types](https://github.com/biyonik/conduit-go/pkg/validation/types)"
    "fmt"
)

func main() {
    // 1. Şemayı tanımla
    kullaniciSemasi := v.Make().Shape(map[string]v.Type{
        "name": types.String().Required().Min(3).Label("Ad Soyad").Trim(),
        "email": types.String().Required().Email().Label("E-posta Adresi").Trim(),
        "age": types.Number().Min(18).Integer().Label("Yaş"),
        "role": types.String().OneOf("user", "admin", "editor").Default("user"),
    })

    // 2. Ham veriyi hazırla
    data := map[string]any{
        "name":  "  Ahmet Altun ", // Trim() ile temizlenecek
        "email": "ahmet@example.com",
        "age":   30,
    }

    // 3. Doğrula
    result := kullaniciSemasi.Validate(data)

    // 4. Sonuçları kontrol et
    if result.HasErrors() {
        fmt.Println("Doğrulama başarısız:")
        for field, errors := range result.Errors() {
            for _, err := range errors {
                fmt.Printf("- %s: %s\n", field, err)
            }
        }
    } else {
        fmt.Println("Doğrulama başarılı!")
        // Temizlenmiş ve doğrulanmış veriyi al
        validData := result.ValidData()
        fmt.Printf("Hoş geldin, %s! Rolün: %s\n", validData["name"], validData["role"])
        // Çıktı: Hoş geldin, Ahmet Altun! Rolün: user
    }
}
```

### Kullanılabilir Tipler

#### `types.String()`
```go
types.String().
    Required().
    Min(5).
    Max(100).
    Email().
    URL().
    OneOf("admin", "user").
    Password( // Şifre kuralları
        rules.WithMinLength(10),
        rules.WithRequireUppercase(true),
        rules.WithRequireNumeric(true),
    ).
    IP(4) // IPv4 gerektirir
    Phone("TR") // TR telefon numarası gerektirir
```

#### `types.Number()`
```go
types.Number().
    Required().
    Min(0).
    Max(100).
    Integer() // Tamsayı olmalı
```

#### `types.Boolean()`
```go
types.Boolean().
    Required().
    Default(false)
```

#### `types.Date()`
Varsayılan olarak Go'nun standart `2006-01-02` formatını kullanır.
```go
types.Date().
    Required().
    Format("02/01/2006"). // Özel format (dd/mm/yyyy)
    Min("01/01/2020").
    Max("31/12/2025")
```

#### `types.Array()`
```go
types.Array().
    Required().
    Min(1). // Minimum 1 eleman
    Max(5). // Maksimum 5 eleman
    Elements( // Dizideki her elemanı doğrula
        types.String().Required().Email(),
    )
```

#### `types.Object()`
```go
types.Object().
    Required().
    Shape(map[string]v.Type{
        "street": types.String().Required(),
        "city":   types.String().Required(),
        "zip":    types.String().Required().Min(5).Max(5),
    })
```

### Özelleşmiş Tipler

```go
// UUID Doğrulaması
types.Uuid().
    Required().
    Version(4) // UUIDv4 gerektirir

// IBAN Doğrulaması
types.Iban().
    Required().
    Country("TR") // Belirli bir ülke için doğrula

// Kredi Kartı Doğrulaması
types.CreditCard().
    Required().
    Type("visa") // "visa", "mastercard", vb.
```

### Gelişmiş String (Temizleme)

`AdvancedString`, `String` tipinin tüm kurallarını miras alır ve güçlü temizleme (sanitization) ve filtreleme özellikleri ekler.

```go
types.AdvancedString().
    // Dönüşüm (Temizleme)
    Trim().
    StripTags().
    EscapeHTML().
    SanitizeFilename().
    FilterEmoji(true). // Emoji'leri kaldır

    // Doğrulama (Filtreleme)
    CharSet("alphanumeric"). // Sadece a-z, A-Z, 0-9
    Domain(true).            // Geçerli bir domain olmalı (alt-domain dahil)
    TurkishChars(false)      // TR karakter içermemeli
```

### Gelişmiş Doğrulama

#### Çapraz Alan Doğrulaması
Alanları birbirine karşı doğrulamak için `.CrossValidate()` kullanın. Bu fonksiyon, *sadece* tüm bireysel alanlar zaten geçerliyse çalışır.

```go
sifreSemasi := v.Make().Shape(map[string]v.Type{
    "password":        types.String().Min(8).Label("Şifre"),
    "passwordConfirm": types.String().Label("Şifre Tekrar"),
}).CrossValidate(func(data map[string]any) error {
    pass, _ := data["password"].(string)
    confirm, _ := data["passwordConfirm"].(string)

    if pass != confirm {
        // Bu hata "_cross_validation" alanına eklenir
        return fmt.Errorf("Şifreler uyuşmuyor")
    }
    return nil
})
```

#### Koşullu Doğrulama
Başka bir alanın değerine göre dinamik olarak şema uygulamak için `.When()` kullanın.

```go
odemeSemasi := v.Make().Shape(map[string]v.Type{
    "paymentType": types.String().Required().OneOf("credit_card", "bank_transfer"),
    "cardNumber":  types.CreditCard(), // Varsayılan olarak zorunlu değil
    "expiryDate":  types.Date().Format("01/06"), // Varsayılan olarak zorunlu değil
    
}).When("paymentType", "credit_card", func() v.Schema {
    // Bu şema, SADECE paymentType == "credit_card" ise uygulanır
    return v.Make().Shape(map[string]v.Type{
        "cardNumber": types.CreditCard().Required().Label("Kart Numarası"),
        "expiryDate": types.Date().Format("01/06").Required().Label("Son Kul. Tarihi"),
    })
})
```