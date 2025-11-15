// Package response, web uygulamalarında JSON tabanlı çıktı üretimini
// tek bir merkezden kontrollü şekilde yapmayı amaçlayan küçük ama
// oldukça önemli bir yardımcı pakettir. Bu paket, özellikle API
// geliştirme süreçlerinde sıkça ihtiyaç duyulan; başarılı veya hatalı
// yanıtların standart bir formda istemciye iletilmesini sağlar.
//
// Laravel ve Symfony gibi büyük frameworklerin "Response Factory"
// mantığını örnek alarak sadeleştirilmiş bir yapı sunar. Projenin
// herhangi bir yerinde, HTTP yanıtlarını tek satırlık fonksiyonlarla
// oluşturabilmek, hem kod tekrarını azaltır hem de geliştiricinin daha
// tutarlı bir API üretmesine yardım eder. Paket; başarılı (success)
// yanıtlarını, hata (error) yanıtlarını ve meta bilgiler içeren
// kapsamlı JSON çıktılarını standart bir sözleşme hâline getirir.
//
// Aşağıdaki tüm yapı ve fonksiyon tanımları, profesyonel düzeyde bir
// API geliştirme deneyimi sağlamak amacıyla detaylı açıklama satırları
// ile desteklenmiştir.
package response

import (
	"encoding/json"
	"net/http"
)

// JSONResponse, tüm API yanıtlarının ortak veri sözleşmesini (contract)
// temsil eden bir modeldir. Modern API mimarilerinde, yanıtların
// standart bir formda olması; dokümantasyon süreçlerini
// kolaylaştırdığı gibi, frontend tarafında veri işleme mantığını da
// sadeleştirir.
//
// Alanlar:
//   - Success: İşlemin başarılı olup olmadığını belirtir. true/false.
//   - Data: İşlem başarılıysa döndürülen asli içerik burada taşınır.
//   - Error: İşlem başarısızsa hata mesajı buraya yazılır.
//   - Meta: Sayfalama, istatistik, toplam kayıt vb. ek bilgiler için
//     kullanılan, isteğe bağlı meta veri alanıdır.
type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// Send, HTTP yanıtını istenen statü kodu ve JSONResponse yapısı ile
// birlikte istemciye gönderen temel fonksiyondur. Bu fonksiyon, tüm
// diğer "Success" ve "Error" fonksiyonlarının arka planda çağırdığı
// ana merkez görevi görür.
//
// Parametreler:
//   - w: Yanıtın gönderileceği ResponseWriter arayüzü.
//   - status: HTTP durum kodu (200, 201, 400, 500 vs.).
//   - payload: JSON olarak kodlanıp gönderilecek olan veri yapısı.
//
// Fonksiyon Akışı:
//  1. Content-Type başlığı JSON olarak ayarlanır.
//  2. HTTP durum kodu yazılır.
//  3. Gönderilecek payload JSON'a çevrilerek çıktı akışına yazılır.
//  4. Encode sırasında bir hata oluşursa hata fonksiyona döndürülür.
func Send(w http.ResponseWriter, status int, payload JSONResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(payload)
	if err != nil {
		return err
	}

	return nil
}

// Success, başarılı bir işlem sonucunda standart bir JSON çıktı
// oluşturmak için kullanılan kolaylaştırıcı (helper) fonksiyondur.
// Bu fonksiyon, geliştiricinin sürekli olarak JSONResponse
// yapılandırmasıyla uğraşmasını engeller ve tek satırda temiz bir
// başarı yanıtı döndürmesine imkân sağlar.
//
// Parametreler:
//   - w: Yanıt yazıcısı.
//   - status: HTTP durum kodu (genellikle 200, 201 vs.).
//   - data: Başarılı işlemin istemciye iletilmek istenen ana içeriği.
//   - meta: Ek bilgiler (sayfalama, toplam kayıt gibi). İsteğe bağlıdır.
//
// Döndürür:
//   - error: JSON encode sırasında oluşabilecek bir hata.
func Success(w http.ResponseWriter, status int, data interface{}, meta interface{}) error {
	return Send(w, status, JSONResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error, başarısız bir işlem sonucunda istemciye hata mesajı
// döndürmek için kullanılan yardımcı fonksiyondur. Geliştiricinin her
// hata durumunda manuel olarak JSONResponse oluşturma yükünü ortadan
// kaldırır ve API genelinde standartlaşmış bir hata yapısı sağlar.
//
// Parametreler:
//   - w: Yanıt yazıcısı.
//   - status: HTTP durum kodu (400, 404, 422, 500 vs.).
//   - errMsg: Döndürülecek olan açıklayıcı hata mesajı.
//
// Döndürür:
//   - error: Gönderim veya encode sürecinde oluşan hata.
func Error(w http.ResponseWriter, status int, errData any) error {
	payload := JSONResponse{
		Success: false,
	}

	// Gelen hatanın tipine göre JSONResponse'u doldur
	switch e := errData.(type) {
	case string:
		payload.Error = e
	case error:
		payload.Error = e.Error()
	case map[string][]string:
		payload.Error = "Doğrulama hatası" // Genel mesaj
		payload.Data = e                   // Detaylı hataları 'data' alanına koy
	default:
		payload.Error = "Bilinmeyen bir sunucu hatası oluştu"
	}

	return Send(w, status, payload)
}
