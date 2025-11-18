// -----------------------------------------------------------------------------
// Process Upload Job
// -----------------------------------------------------------------------------
// Dosya upload işleme job'u.
//
// Bu job upload queue'sunda çalışır ve dosya işler:
// - Thumbnail oluşturma
// - Format dönüşümü
// - Virus scan
// - Storage'a kaydetme
// -----------------------------------------------------------------------------

package jobs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/biyonik/conduit-go/pkg/queue"
)

// ProcessUploadJob, dosya upload işleme job'u.
type ProcessUploadJob struct {
	queue.BaseJob
	FilePath string `json:"file_path"`
	UserID   int64  `json:"user_id"`
	FileType string `json:"file_type"` // image, video, document
}

// Handle, dosya işleme işlemini yapar.
func (j *ProcessUploadJob) Handle() error {
	log.Printf("📁 Processing upload: %s (user: %d, type: %s)", j.FilePath, j.UserID, j.FileType)

	// Simulated processing steps
	steps := []string{
		"Validating file...",
		"Scanning for viruses...",
		"Generating thumbnail...",
		"Optimizing file...",
		"Uploading to storage...",
	}

	for i, step := range steps {
		log.Printf("   [%d/%d] %s", i+1, len(steps), step)
		time.Sleep(500 * time.Millisecond) // Simulate work

		// Hata simülasyonu (test için):
		// if i == 2 && j.FilePath == "fail.jpg" {
		//     return fmt.Errorf("thumbnail generation failed")
		// }
	}

	log.Printf("✅ Upload processed successfully: %s", j.FilePath)

	// TODO: Database'e file record ekle
	// TODO: User'a notification gönder

	return nil
}

// Failed, job başarısız olduğunda çağrılır.
func (j *ProcessUploadJob) Failed(err error) error {
	log.Printf("❌ Upload processing failed: %s (file: %s, error: %v)", j.ID, j.FilePath, err)

	// TODO: Temp file'ı sil
	// TODO: User'a hata notification gönder
	// TODO: Admin'e alert gönder

	return nil
}

// GetPayload, job'ı serialize eder.
func (j *ProcessUploadJob) GetPayload() ([]byte, error) {
	return json.Marshal(j)
}

// SetPayload, job'ı deserialize eder.
func (j *ProcessUploadJob) SetPayload(data []byte) error {
	return json.Unmarshal(data, j)
}

// NewProcessUploadJob, yeni bir ProcessUploadJob oluşturur.
//
// Parametreler:
//   - filePath: Dosya yolu
//   - userID: Kullanıcı ID'si
//   - fileType: Dosya tipi (image, video, document)
//
// Döndürür:
//   - *ProcessUploadJob: Job instance
//
// Örnek:
//
//	job := jobs.NewProcessUploadJob(
//	    "/tmp/upload_abc123.jpg",
//	    42,
//	    "image",
//	)
//	queue.Push(job, "uploads")
func NewProcessUploadJob(filePath string, userID int64, fileType string) *ProcessUploadJob {
	return &ProcessUploadJob{
		BaseJob: queue.BaseJob{
			MaxAttempts: 5, // Upload job'lar için daha fazla retry
		},
		FilePath: filePath,
		UserID:   userID,
		FileType: fileType,
	}
}
