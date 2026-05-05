# PROGRESS & STATUS

Dokumen ini berisi rangkuman status pengembangan `payment-service` yang mengintegrasikan Midtrans dengan sistem berbasis arsitektur Hexagonal (Ports & Adapters).

---

## ✅ YANG SUDAH DIIMPLEMENTASIKAN (SELESAI 100%)

Seluruh fitur inti dan _advanced architecture_ telah selesai diimplementasikan sesuai dengan instruksi `ARCHITECTURE.md` dan `CODING_GO_MICROSERVICE_STANDAR.md`:

### 1. Arsitektur & Konfigurasi Dasar
- [x] Struktur folder berdasarkan **Clean Architecture / Hexagonal Architecture** (`cmd`, `internal/domain`, `internal/usecase`, `internal/adapter`, `pkg`, `config`).
- [x] Manajemen konfigurasi tersentralisasi (`config/config.go` & `config/loader.go`) dengan dukungan `.env` fallback.
- [x] Setup `cmd/main.go` yang melakukan Dependency Injection untuk semua layer secara *clean*.
- [x] Environment Variables (`.env` dan `.env.example`).
- [x] Git ignores (`.gitignore`).

### 2. Domain Layer
- [x] Entity Database murni tanpa library eksternal (`Transaction`, `WebhookLog`, `Customer`, `PaymentMethod`, `Refund`, `OutboxEvent`).
- [x] Domain Error Handling (`AppError` beserta Sentinel Errors).
- [x] Definisi `EventEnvelope` dan Payload untuk pesan ke Kafka.

### 3. Usecase Layer (Business Logic)
- [x] **Inbound & Outbound Ports**: Definisi *contract interfaces*.
- [x] **`ChargeTransactionUsecase`**: Integrasi gateway Midtrans untuk mendapatkan `SnapToken` dan penyimpanan ke DB secara transaksional.
- [x] **`HandleMidtransWebhookUsecase`**: Validasi `SignatureKey` SHA-512 ketat, merubah state transaksi.

### 4. Advanced Architecture (Pola Lanjutan)
- [x] **Outbox Pattern**: Transaksi yang berhasil (settled) dimasukkan ke tabel `outbox_events` (Atomic dengan status update).
- [x] **Background Worker**: `OutboxWorker` melakukan *polling* setiap interval 2 detik, me-publish ke Kafka, dan menandai event sebagai `published`.
- [x] **Idempotency Store (Redis)**: Penyimpanan `Idempotency-Key` secara distributed via `redis/go-redis/v9`. Mencegah notifikasi/request duplikat.
- [x] **Observability**: 
  - **Tracing**: Setup **OpenTelemetry (OTEL)** via gRPC eksportir.
  - **Metrics**: Endpoin **Prometheus** di `/metrics` berjalan via Fiber middleware.

### 5. Adapter Layer
- [x] **HTTP Handler**: `payment_handler.go` & `webhook_handler.go` (Fiber).
- [x] **Database Repository**: GORM implementation (`transaction_repository`, `webhook_log_repository`, `outbox_repository`, `tx_manager`).
- [x] **Midtrans Client (Gateway)**: `midtrans-go/snap`.
- [x] **Kafka Publisher**: `segmentio/kafka-go`.

### 6. Testing & Deployment
- [x] **Unit Testing**: 
  - `charge_transaction_usecase_test.go`
  - `handle_midtrans_webhook_usecase_test.go` (Test lengkap verifikasi valid/invalid signature dan logika state).
- [x] **Database Migration**: File SQL `up.sql` dan `down.sql` lengkap.
- [x] **Containerization**: `Dockerfile` (Multi-stage alpine) siap pakai.
- [x] **Orchestration**: `docker-compose.yaml` untuk menjalankan App, Postgres, Redis, Zookeeper, dan Kafka.

---

## ⏳ NEXT PLAN (RENCANA BERIKUTNYA)

Mengingat `payment-service` sudah masuk dalam kategori **Production-Ready**, langkah selanjutnya berfokus pada integrasi antar-layanan (Microservices inter-communication) dan penyempurnaan UI/Operasional:

### 1. Integrasi dengan Order Service
- Payment Service hanya memproses transaksi, tapi belum ada service yang membuat pesanan (Order).
- **Rencana**: Membuat `order-service` atau memastikan Payment Service dapat dipanggil via internal gRPC / REST dari service lain saat checkout.

### 2. Kafka Consumer di Service Lain
- Payment Service sudah mem-publish event `PAYMENT_SETTLED` ke topik `payment.payment.settled`.
- **Rencana**: Harus ada *Consumer* di sisi `notification-service` untuk mengirim email resi, dan *Consumer* di sisi `order-service` untuk mengubah status barang menjadi 'Diproses'.

### 3. Simulasi Midtrans Simulator / Frontend
- Saat ini Snap Token sudah dikembalikan, namun tidak bisa di-test tanpa antarmuka yang menjalankan script Midtrans Snap (`window.snap.pay(token)`).
- **Rencana**: Membuat file HTML statis (Frontend sederhana) atau Postman Collection lengkap yang langsung membaca respon Token lalu menampilkan UI pembayaran.

### 4. Implementasi Endpoint GET Transaction Status
- Klien mungkin ingin mem-polling status pembayarannya secara manual jika Webhook terlambat.
- **Rencana**: Membuat endpoint `GET /api/v1/payments/{order_id}/status` untuk mengecek status transaksi langsung ke DB.
