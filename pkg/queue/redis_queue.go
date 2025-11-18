// -----------------------------------------------------------------------------
// Redis Queue Driver
// -----------------------------------------------------------------------------
// Redis-based queue implementation (production-ready).
//
// Özellikler:
// - Atomic operations (RPUSH, BLPOP)
// - Delayed jobs (sorted sets)
// - Failed job tracking
// - Multiple queue support
//
// Redis Data Structures:
// - queues:{name} - List (FIFO)
// - queues:{name}:delayed - Sorted Set (timestamp score)
// - queues:{name}:reserved - Set (processing jobs)
// - queues:failed - List (failed jobs)
// -----------------------------------------------------------------------------

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisQueue, Redis-based queue implementation.
type RedisQueue struct {
	client *redis.Client
	logger *log.Logger
	prefix string // Key prefix (namespace)
}

// NewRedisQueue, yeni bir Redis queue instance oluşturur.
//
// Parametreler:
//   - client: Redis client
//   - logger: Log instance
//   - prefix: Key prefix (örn: "conduit:")
//
// Döndürür:
//   - *RedisQueue: Queue instance
//
// Örnek:
//
//	queue := NewRedisQueue(redisClient, logger, "conduit:")
//	queue.Push(emailJob, "emails")
func NewRedisQueue(client *redis.Client, logger *log.Logger, prefix string) *RedisQueue {
	return &RedisQueue{
		client: client,
		logger: logger,
		prefix: prefix,
	}
}

// queueKey, queue name'den Redis key oluşturur.
func (r *RedisQueue) queueKey(queue string) string {
	return r.prefix + "queues:" + queue
}

// delayedKey, delayed jobs için Redis key oluşturur.
func (r *RedisQueue) delayedKey(queue string) string {
	return r.prefix + "queues:" + queue + ":delayed"
}

// reservedKey, reserved jobs için Redis key oluşturur.
func (r *RedisQueue) reservedKey(queue string) string {
	return r.prefix + "queues:" + queue + ":reserved"
}

// failedKey, failed jobs için Redis key oluşturur.
func (r *RedisQueue) failedKey() string {
	return r.prefix + "queues:failed"
}

// Push, job'ı hemen kuyruğa ekler.
func (r *RedisQueue) Push(job Job, queue string) error {
	return r.Later(0, job, queue)
}

// Later, job'ı belirli bir gecikme ile kuyruğa ekler.
func (r *RedisQueue) Later(delay time.Duration, job Job, queue string) error {
	ctx := context.Background()

	// Job metadata set et
	if job.GetID() == "" {
		job.SetID(uuid.New().String())
	}
	job.SetQueue(queue)

	// Job payload oluştur
	payload, err := r.createPayload(job, delay)
	if err != nil {
		r.logger.Printf("❌ Payload oluşturma hatası: %v", err)
		return fmt.Errorf("payload oluşturulamadı: %w", err)
	}

	// Serialize et
	data, err := json.Marshal(payload)
	if err != nil {
		r.logger.Printf("❌ JSON encode hatası: %v", err)
		return fmt.Errorf("json encode hatası: %w", err)
	}

	// Delayed job ise sorted set'e ekle
	if delay > 0 {
		availableAt := time.Now().Add(delay).Unix()
		err = r.client.ZAdd(ctx, r.delayedKey(queue), redis.Z{
			Score:  float64(availableAt),
			Member: data,
		}).Err()

		if err != nil {
			r.logger.Printf("❌ Delayed job push hatası [%s]: %v", queue, err)
			return fmt.Errorf("delayed job push hatası: %w", err)
		}

		r.logger.Printf("✅ Delayed job pushed: %s (queue: %s, delay: %v)", job.GetID(), queue, delay)
		return nil
	}

	// Normal job ise list'e ekle
	err = r.client.RPush(ctx, r.queueKey(queue), data).Err()
	if err != nil {
		r.logger.Printf("❌ Job push hatası [%s]: %v", queue, err)
		return fmt.Errorf("job push hatası: %w", err)
	}

	r.logger.Printf("✅ Job pushed: %s (queue: %s)", job.GetID(), queue)
	return nil
}

// Pop, kuyruktan bir job çeker.
func (r *RedisQueue) Pop(queue string) (Job, error) {
	ctx := context.Background()

	// Önce delayed jobs'ları kontrol et ve taşı
	r.migrateDelayedJobs(queue)

	// BLPOP ile job çek (5 saniye timeout)
	result, err := r.client.BLPop(ctx, 5*time.Second, r.queueKey(queue)).Result()
	if err != nil {
		if err == redis.Nil {
			// Queue boş
			return nil, nil
		}
		r.logger.Printf("❌ Job pop hatası [%s]: %v", queue, err)
		return nil, fmt.Errorf("job pop hatası: %w", err)
	}

	// result[0] = key, result[1] = value
	data := result[1]

	// Deserialize et
	var payload JobPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		r.logger.Printf("❌ JSON decode hatası: %v", err)
		return nil, fmt.Errorf("json decode hatası: %w", err)
	}

	// Job instance oluştur (tip registry'den)
	job, err := r.createJobInstance(&payload)
	if err != nil {
		r.logger.Printf("❌ Job instance oluşturma hatası: %v", err)
		return nil, fmt.Errorf("job instance oluşturulamadı: %w", err)
	}

	// Job'ı reserved set'e ekle
	r.client.SAdd(ctx, r.reservedKey(queue), data)

	r.logger.Printf("🔄 Job popped: %s (queue: %s, attempts: %d)", job.GetID(), queue, job.GetAttempts())
	return job, nil
}

// Delete, job'ı kuyruktan siler.
func (r *RedisQueue) Delete(queue string, job Job) error {
	ctx := context.Background()

	// Job payload oluştur
	payload, err := r.createPayload(job, 0)
	if err != nil {
		return err
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Reserved set'ten sil
	err = r.client.SRem(ctx, r.reservedKey(queue), data).Err()
	if err != nil {
		r.logger.Printf("❌ Job delete hatası [%s]: %v", queue, err)
		return fmt.Errorf("job delete hatası: %w", err)
	}

	r.logger.Printf("✅ Job deleted: %s (queue: %s)", job.GetID(), queue)
	return nil
}

// Release, job'ı tekrar kuyruğa ekler.
func (r *RedisQueue) Release(queue string, job Job, delay time.Duration) error {
	ctx := context.Background()

	// Attempt sayısını artır
	job.SetAttempts(job.GetAttempts() + 1)

	// Job payload oluştur
	payload, err := r.createPayload(job, delay)
	if err != nil {
		return err
	}

	// Serialize et
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Reserved set'ten sil
	r.client.SRem(ctx, r.reservedKey(queue), data)

	// Max attempts aşıldı mı?
	if job.GetAttempts() >= job.GetMaxAttempts() {
		// Failed jobs'a ekle
		r.client.RPush(ctx, r.failedKey(), data)
		r.logger.Printf("⚠️  Job failed (max attempts): %s (queue: %s, attempts: %d)", job.GetID(), queue, job.GetAttempts())
		return nil
	}

	// Tekrar kuyruğa ekle
	return r.Later(delay, job, queue)
}

// Size, kuyruktaki job sayısını döndürür.
func (r *RedisQueue) Size(queue string) (int64, error) {
	ctx := context.Background()

	// Normal queue size
	normalSize, err := r.client.LLen(ctx, r.queueKey(queue)).Result()
	if err != nil {
		return 0, err
	}

	// Delayed queue size
	delayedSize, err := r.client.ZCard(ctx, r.delayedKey(queue)).Result()
	if err != nil {
		return 0, err
	}

	return normalSize + delayedSize, nil
}

// migrateDelayedJobs, delayed jobs'ları kontrol eder ve zamanı gelenleri taşır.
func (r *RedisQueue) migrateDelayedJobs(queue string) {
	ctx := context.Background()
	now := float64(time.Now().Unix())

	// Zamanı gelen job'ları bul
	jobs, err := r.client.ZRangeByScore(ctx, r.delayedKey(queue), &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()

	if err != nil || len(jobs) == 0 {
		return
	}

	// Her job'ı normal queue'ya taşı
	for _, jobData := range jobs {
		// Normal queue'ya ekle
		r.client.RPush(ctx, r.queueKey(queue), jobData)
		// Delayed queue'dan sil
		r.client.ZRem(ctx, r.delayedKey(queue), jobData)
	}

	r.logger.Printf("🔄 Migrated %d delayed jobs (queue: %s)", len(jobs), queue)
}

// createPayload, job'dan JobPayload oluşturur.
func (r *RedisQueue) createPayload(job Job, delay time.Duration) (*JobPayload, error) {
	// Job'ı serialize et
	jobData, err := job.GetPayload()
	if err != nil {
		return nil, err
	}

	// Job type belirle (reflection ile)
	jobType := fmt.Sprintf("%T", job)

	// Payload oluştur
	availableAt := time.Now()
	if delay > 0 {
		availableAt = availableAt.Add(delay)
	}

	payload := &JobPayload{
		ID:          job.GetID(),
		Type:        jobType,
		Queue:       job.GetQueue(),
		Payload:     jobData,
		Attempts:    job.GetAttempts(),
		MaxAttempts: job.GetMaxAttempts(),
		CreatedAt:   time.Now(),
		AvailableAt: availableAt,
	}

	return payload, nil
}

// createJobInstance, JobPayload'dan Job instance oluşturur.
//
// NOT: Bu fonksiyon job type registry kullanır.
// Her job tipi register edilmelidir.
func (r *RedisQueue) createJobInstance(payload *JobPayload) (Job, error) {
	// Job type registry'den instance oluştur
	job, err := JobRegistry.Create(payload.Type)
	if err != nil {
		return nil, err
	}

	// Metadata set et
	job.SetID(payload.ID)
	job.SetQueue(payload.Queue)
	job.SetAttempts(payload.Attempts)

	// Payload set et
	if err := job.SetPayload(payload.Payload); err != nil {
		return nil, err
	}

	return job, nil
}
