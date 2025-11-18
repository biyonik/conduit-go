# -----------------------------------------------------------------------------
# Conduit-Go Makefile
# -----------------------------------------------------------------------------
# Bu Makefile, development sürecini hızlandırmak için yaygın komutları içerir.
#
# Kullanım:
#   make run          # Uygulamayı başlat
#   make test         # Testleri çalıştır
#   make build        # Binary oluştur
#   make clean        # Temizlik
#   make docker-up    # Docker container'ları başlat
# -----------------------------------------------------------------------------

.PHONY: help run build test clean docker-up docker-down migrate-up migrate-down

# Varsayılan hedef: help
.DEFAULT_GOAL := help

# Go binary yolu
BINARY_NAME=conduit-go
BINARY_PATH=./bin/$(BINARY_NAME)

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Tüm make komutlarını gösterir
help:
	@echo "Kullanılabilir komutlar:"
	@echo "  make run          - Uygulamayı başlat (hot-reload ile)"
	@echo "  make build        - Production binary oluştur"
	@echo "  make test         - Testleri çalıştır"
	@echo "  make clean        - Binary ve cache'i temizle"
	@echo "  make docker-up    - Docker container'ları başlat"
	@echo "  make docker-down  - Docker container'ları durdur"
	@echo "  make fmt          - Kodu formatla"
	@echo "  make lint         - Linter çalıştır"

## run: Uygulamayı development modda başlatır
run:
	@echo "🚀 Uygulama başlatılıyor..."
	@go run cmd/api/main.go

## build: Production binary oluşturur
build:
	@echo "🔨 Binary oluşturuluyor..."
	@mkdir -p ./bin
	@go build $(LDFLAGS) -o $(BINARY_PATH) cmd/api/main.go
	@echo "✅ Binary oluşturuldu: $(BINARY_PATH)"

## test: Tüm testleri çalıştırır
test:
	@echo "🧪 Testler çalıştırılıyor..."
	@go test -v ./...

## test-coverage: Test coverage raporu oluşturur
test-coverage:
	@echo "📊 Test coverage hesaplanıyor..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage raporu: coverage.html"

## clean: Binary ve cache dosyalarını temizler
clean:
	@echo "🧹 Temizlik yapılıyor..."
	@rm -rf ./bin
	@go clean
	@echo "✅ Temizlik tamamlandı"

## fmt: Go kod formatlama
fmt:
	@echo "💅 Kod formatlanıyor..."
	@go fmt ./...
	@echo "✅ Format tamamlandı"

## lint: golangci-lint ile kod analizi
lint:
	@echo "🔍 Linter çalıştırılıyor..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint yüklü değil. Yüklemek için:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## docker-up: Docker container'ları başlatır (MySQL, Redis, vb.)
docker-up:
	@echo "🐳 Docker container'lar başlatılıyor..."
	@docker-compose up -d
	@echo "✅ Container'lar başlatıldı"

## docker-down: Docker container'ları durdurur
docker-down:
	@echo "🐳 Docker container'lar durduruluyor..."
	@docker-compose down
	@echo "✅ Container'lar durduruldu"

## deps: Go dependency'lerini günceller
deps:
	@echo "📦 Dependency'ler güncelleniyor..."
	@go mod tidy
	@go mod download
	@echo "✅ Dependency'ler güncellendi"

## security: Go security checker (gosec) çalıştırır
security:
	@echo "🔒 Security check yapılıyor..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec yüklü değil. Yüklemek için:"; \
		echo "  go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# Queue Worker Commands
.PHONY: worker worker-all worker-emails

# Start queue worker for default queue
worker:
	@echo "Starting queue worker (default queue)..."
	@go run cmd/worker/main.go

# Start queue worker for all queues
worker-all:
	@echo "Starting queue worker (all queues)..."
	@go run cmd/worker/main.go default emails notifications uploads

# Start queue worker for emails queue only
worker-emails:
	@echo "Starting queue worker (emails queue)..."
	@go run cmd/worker/main.go emails

# Test queue system
test-queue:
	@echo "Running queue tests..."
	@go test -v ./tests -run Queue