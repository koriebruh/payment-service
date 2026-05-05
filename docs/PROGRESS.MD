# PROGRESS & STATUS

Dokumen ini berisi rangkuman status pengembangan `payment-service` yang mengintegrasikan Midtrans dengan sistem berbasis arsitektur Hexagonal (Ports & Adapters).

---

## ✅ YANG SUDAH DIIMPLEMENTASIKAN

### 1. Arsitektur & Konfigurasi Dasar
- [x] Struktur folder berdasarkan **Clean Architecture / Hexagonal Architecture** (`cmd`, `internal/domain`, `internal/usecase`, `internal/adapter`, `pkg`, `config`).
- [x] Manajemen konfigurasi tersentralisasi (`config/config.go` & `config/loader.go`) dengan dukungan `.env` fallback.
- [x] Setup `cmd/main.go` yang melakukan Dependency Injection untuk semua layer secara *clean*.

### 2. Domain Layer
- [x] Entity Database murni tanpa library eksternal (`Transaction`, `WebhookLog`, `Customer`, `PaymentMethod`, `Refund`).
- [x] Domain Error Handling (`AppError` beserta Sentinel Errors seperti `ErrInvalidSignature`).
- [x] Definisi `EventEnvelope` dan Payload (e.g. `PaymentSettledPayload`) standar untuk pesan ke Kafka.

### 3. Usecase Layer (Business Logic)
- [x] **Inbound & Outbound Ports**: Definisi *contract interfaces* agar usecase tidak bergantung pada implementasi adapter.
- [x] **`ChargeTransactionUsecase`**:
  - Validasi & Persiapan data order
  - Panggil API Midtrans (via `PaymentGatewayPort`) untuk mendapatkan `SnapToken` dan `PaymentURL`.
  - Simpan transaksi berstatus `pending` ke database dalam 1 transaksi DB.
- [x] **`HandleMidtransWebhookUsecase`**:
  - Hitung dan cocokkan Signature Key dengan algoritma SHA-512 Midtrans (`order_id + status_code + gross_amount + server_key`).
  - Pengecekan *Idempotency* membaca data dari `webhook_logs` (mengabaikan notifikasi duplikat).
  - Merubah state transaksi dari `pending` menjadi `settlement`, `cancel`, `expire`, dll.
  - Jika `settlement`, maka memicu (trigger) *Event Publisher* agar service lain tau pembayaran sukses.

### 4. Adapter Layer
- [x] **HTTP Handler (Go Fiber)**: `payment_handler.go` & `webhook_handler.go` untuk menangani format JSON request.
- [x] **Middleware & Validator**: Binding JSON dan validasi data (`go-validator`).
- [x] **Database Repository (GORM)**: Implementasi *Transaction Repository* & *Webhook Log Repository* beserta `TxManager` untuk mengontrol *commit/rollback* dari Usecase.
- [x] **Midtrans Client (Gateway)**: Implementasi memanggil Midtrans menggunakan `midtrans-go/snap` resmi.
- [x] **Kafka Publisher**: Implementasi Event Publisher menggunakan `segmentio/kafka-go`.

### 5. Utilities & Infrastructure
- [x] **Structured Logging**: Menggunakan standar Go 1.21 `log/slog` berformat JSON (`pkg/logger`).
- [x] **API Response Factory**: Format balasan HTTP yang konsisten (terdapat field `success`, `code`, `message`, `data`, `request_id`).
- [x] **Database Migration**: File SQL `up.sql` dan `down.sql` berisi skema tabel PostgreSQL (`payment_methods`, `customers`, `transactions`, `refunds`, `webhook_logs`).

---

## ⏳ YANG BELUM DIIMPLEMENTASIKAN (NEXT STEPS)

### 1. Implementasi Outbox Pattern (WAJIB Arsitektur)
- **Status saat ini**: Jika pembayaran `settlement`, Usecase langsung memanggil `kafkaPublisher.Publish()` di dalam *transaction block*.
- **Harusnya**: Sesuai `ARCHITECTURE.md`, `Event publishing MUST be atomic with its state change — always use Outbox`. Artinya event seharusnya di-insert dulu ke tabel `outbox_events` di dalam PostgreSQL saat *commit*, kemudian sebuah *background worker* (`outbox_worker.go`) membaca tabel tersebut dan barulah mem-publish ke Kafka. 

### 2. Idempotency Store
- **Status saat ini**: `ChargeTransactionUsecase` memakai *mock* `memoryStore` yang menggunakan RAM aplikasi (bisa hilang jika server mati).
- **Harusnya**: Implementasikan `RedisIdempotencyStore` menggunakan library `go-redis` untuk melacak `Idempotency-Key` HTTP Request secara *distributed*.

### 3. Setup Observability (Tracing & Metrics)
- **Status saat ini**: Sistem logging (slog) sudah berjalan.
- **Harusnya**: Menambahkan OpenTelemetry (OTEL) untuk men-*generate* `trace_id` yang bisa dilihat di Jaeger, serta Prometheus metrics (`HTTP requests duration`, `Active Goroutines`, dsb) seperti instruksi di `CODING_GO_MICROSERVICE_STANDAR.md`.

### 4. Testing (Unit Test & Integration Test)
- **Status saat ini**: Belum ada *test files*.
- **Harusnya**: Menulis `charge_transaction_usecase_test.go` (dengan mock repo & midtrans) dan test khusus logika perhitungan Signature di webhook.

### 5. Deployment & Containerization
- **Status saat ini**: Bisa dijalankan lokal dengan `go run`.
- **Harusnya**: Setup `Dockerfile` khusus service ini, serta memastikan enviroment variables (*Midtrans Keys*, *Database URL*, *Kafka Brokers*) aman dikelola oleh Kubernetes (Helm) atau `docker-compose.yaml`.
