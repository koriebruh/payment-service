# ARCHITECTURE.md
> **Dokumen ini mendefinisikan prinsip, struktur, boundary, dan topology sistem.**
> Baca bersama `CODING_GO_MICROSERVICE_STANDAR.md` — keduanya wajib dibaca sebelum menulis satu baris kode.
>
> **Scope file ini:** Arsitektur, struktur, boundary, flow, topology.
> **Cara menulis kode:** → `CODING_GO_MICROSERVICE_STANDAR.md`
> **Status fitur saat ini:** → `PROGRESS.md`

---

## 0. Architecture Principles (Non-Negotiable)

Ini adalah kompas. Jika ada keputusan yang bertentangan dengan prinsip ini — keputusan itu yang salah, bukan prinsipnya.

```
1. Business logic lives ONLY in domain and usecase layer.
2. All dependencies MUST point inward — adapter → usecase → domain.
3. Every external system (DB, gRPC, broker, cache) is an adapter — never core.
4. Side effects (I/O, network, DB write) MUST be isolated in adapter layer.
5. Data consistency is more important than convenience.
6. Explicit is better than implicit — no hidden magic, no framework auto-wiring in domain.
7. Every write operation MUST be deterministic and idempotent.
8. Domain model MUST NOT leak outside usecase boundary.
9. Transaction ownership belongs ONLY to usecase layer.
10. Event publishing MUST be atomic with its state change — always use Outbox.
```

Jika kode yang di-generate melanggar satu saja dari prinsip di atas → **tolak dan generate ulang**.

---

## 1. Pola Arsitektur — Hexagonal (Ports & Adapters)

Project ini menggunakan **Hexagonal Architecture** dipadukan dengan prinsip **Clean Architecture**.

Domain ada di pusat. Semua ketergantungan mengarah ke dalam. Tidak ada pengecualian.

### Dependency Rule (Wajib, Tidak Boleh Dilanggar)

```
┌──────────────────────────────────────────────────────┐
│                     ADAPTER                          │
│  handler │ grpc/server │ grpc/client │ consumer      │
│  publisher │ repository │ worker                     │
│                                                      │
│  → boleh import: usecase port, domain, pkg, config   │
│  → DILARANG import: layer adapter lain secara langsung│
└───────────────────────┬──────────────────────────────┘
                        │ import
┌───────────────────────▼──────────────────────────────┐
│                     USECASE                          │
│  business logic + port interface definition          │
│                                                      │
│  → boleh import: domain saja                         │
│  → DILARANG import: adapter, config, framework       │
└───────────────────────┬──────────────────────────────┘
                        │ import
┌───────────────────────▼──────────────────────────────┐
│                      DOMAIN                          │
│  entity │ value object │ domain event                │
│  domain error │ domain rule                          │
│                                                      │
│  → DILARANG import: apapun selain std library        │
└──────────────────────────────────────────────────────┘

middleware → wrapping adapter/handler saja, tidak masuk usecase/domain
pkg/       → utility murni, zero business logic, boleh diimport siapapun
config/    → boleh diimport oleh cmd/ dan adapter/ saja
```

### Compile-Time Boundary Enforcement

Arsitektur yang baik bukan sekadar "jangan dilakukan" — tapi **"tidak mungkin dilakukan"**.

```
Enforcement yang wajib diterapkan:

1. domain/ package → zero external module import
   Jika go.sum berubah karena perubahan di domain/ → itu violation

2. usecase/ package → hanya import domain/
   Jika ada import path ke internal/adapter/ di usecase/ → itu violation

3. Jika kode compile tapi melanggar layer boundary → arsitektur GAGAL

Tools untuk enforce:
- go-cleanarch (linter untuk clean architecture)
- CI: script yang scan import path dan fail jika ada violation
```

---

## 2. Directory Structure

**Semua direktori di bawah ini dianggap MANDATORY secara default.**

Jika ada direktori yang tidak ada:
- Wajib ada justifikasi eksplisit di ADR
- Ketiadaannya tidak boleh merusak konsistensi arsitektur

Pengecualian yang valid (wajib catat di ADR):
- Tidak ada `grpc/` → hanya jika service zero gRPC interaction
- Tidak ada `consumer/` → hanya jika service tidak consume event apapun

```
service-name/
│
├── cmd/
│   └── main.go                        # entrypoint — wire dependency saja, zero logic
│
├── internal/
│   │
│   ├── domain/                        # inti — zero external import
│   │   ├── entity.go                  # aggregate root, entity
│   │   ├── value_object.go            # value object (immutable)
│   │   ├── event.go                   # domain event struct definition
│   │   ├── rule.go                    # domain invariant / business rule
│   │   └── error.go                   # sentinel error & AppError struct
│   │
│   ├── usecase/
│   │   ├── port/
│   │   │   ├── inbound.go             # interface dipanggil adapter (handler, consumer)
│   │   │   └── outbound.go            # interface diimplementasi adapter (repo, publisher)
│   │   ├── dto/
│   │   │   ├── request.go             # usecase-level request struct
│   │   │   └── result.go              # usecase-level result struct
│   │   ├── {action}_{entity}_usecase.go
│   │   └── {action}_{entity}_usecase_test.go
│   │
│   ├── adapter/
│   │   ├── handler/
│   │   │   ├── dto/
│   │   │   │   ├── request.go         # HTTP request struct (JSON binding)
│   │   │   │   └── response.go        # HTTP response struct
│   │   │   ├── {entity}_handler.go
│   │   │   └── health_handler.go
│   │   │
│   │   ├── grpc/
│   │   │   ├── server/                # jika service ini serve gRPC
│   │   │   │   ├── pb/                # generated proto code — jangan diedit manual
│   │   │   │   ├── {entity}_server.go
│   │   │   │   └── mapper.go          # proto ↔ usecase dto mapping
│   │   │   └── client/                # jika service ini call service lain via gRPC
│   │   │       ├── {service}_client.go
│   │   │       └── mapper.go
│   │   │
│   │   ├── consumer/
│   │   │   ├── {entity}_consumer.go
│   │   │   └── mapper.go              # event payload ↔ usecase dto mapping
│   │   │
│   │   ├── publisher/
│   │   │   └── kafka_publisher.go
│   │   │
│   │   ├── repository/
│   │   │   ├── {entity}_repository.go
│   │   │   └── outbox_repository.go
│   │   │
│   │   └── worker/
│   │       └── outbox_worker.go
│   │
│   └── middleware/                    # HTTP cross-cutting concern saja
│       ├── auth.go
│       ├── request_id.go
│       ├── tracing.go
│       ├── logger.go
│       ├── recovery.go
│       └── ratelimit.go
│
├── pkg/                               # shared utility — boleh dipakai service lain
│   ├── response/                      # ApiResponseFactory
│   ├── validator/
│   ├── pagination/
│   ├── idempotency/
│   ├── retry/
│   ├── cache/
│   ├── logger/
│   ├── circuitbreaker/
│   └── metrics/
│
├── config/
│   ├── config.go
│   └── loader.go
│
├── proto/
│   └── {domain}/v{major}/
│       └── {domain}.proto
│
├── migrations/
│   ├── 000001_create_{entity}.up.sql
│   └── 000001_create_{entity}.down.sql
│
└── docs/
    ├── adr/
    │   └── ADR-001-*.md
    └── api/
        └── openapi.yaml
```

---

## 3. Layer Definitions & Responsibilities

### 3.1 Domain Layer

**Tanggung jawab:** Mendefinisikan apa yang ada dalam sistem.

**Wajib ada:** Entity, Value Object, Domain Event (struct saja), Sentinel Error, AppError, Business rule sebagai method entity.

**Dilarang ada:**
- Import package eksternal apapun
- Struct tag (`json:`, `db:`, `validate:`) — ini milik adapter
- I/O dalam bentuk apapun
- Dependency ke usecase atau adapter

```go
// ✅ Domain yang benar — zero external import
package domain

type Transfer struct {
    ID             string
    FromAccountID  string
    ToAccountID    string
    Amount         decimal.Decimal
    Status         TransferStatus
    IdempotencyKey string
    CreatedAt      time.Time
    Version        int
}

// Business rule sebagai method
func (t *Transfer) CanProcess() error {
    if t.Status != StatusPending {
        return ErrTransferNotPending
    }
    return nil
}

// Domain event — hanya definisi struct, bukan publish logic
type TransferInitiatedEvent struct {
    TransferID    string
    FromAccountID string
    Amount        decimal.Decimal
    OccurredAt    time.Time
}

// Sentinel error
var (
    ErrTransferNotPending  = errors.New("transfer is not in pending status")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrAccountNotFound     = errors.New("account not found")
)
```

### 3.2 Usecase Layer

**Tanggung jawab:** Orchestrate alur bisnis. Mendefinisikan port. Mengontrol transaction boundary.

**Wajib ada:** Inbound & outbound port interface, usecase dto (request/result), business logic orchestration, transaction management.

**Dilarang ada:**
- Import adapter package secara langsung
- Import library infra (GORM, Fiber, proto)
- Akses DB/network secara langsung
- Mapping dari/ke proto atau HTTP struct
- Transaction management di handler atau repository

```go
// usecase/port/outbound.go
type TransferRepository interface {
    FindByID(ctx context.Context, id string) (*domain.Transfer, error)
    Save(ctx context.Context, tx Tx, t *domain.Transfer) error
}

type TxManager interface {
    WithTx(ctx context.Context, fn func(tx Tx) error) error
}

// usecase/port/inbound.go
type ExecuteTransferUsecase interface {
    Execute(ctx context.Context, req dto.ExecuteTransferRequest) (*dto.ExecuteTransferResult, error)
}

// usecase/dto/request.go — bukan domain, bukan proto, bukan HTTP struct
type ExecuteTransferRequest struct {
    FromAccountID  string
    ToAccountID    string
    Amount         decimal.Decimal
    Currency       string
    IdempotencyKey string
    RequestID      string
    TraceID        string
}
```

### 3.3 Adapter Layer

**Tanggung jawab:** Implementasi port. Mapping antar representasi data. Boleh import library eksternal.

**Mapping yang wajib ada di adapter:**

```
handler:
  HTTP body          → dto.ExecuteTransferRequest    (masuk usecase)
  dto.Result         → HTTP response struct           (keluar ke client)

grpc/server:
  proto.Request      → dto.ExecuteTransferRequest
  dto.Result         → proto.Response

grpc/client:
  proto.Response     → return value sesuai outbound port interface

consumer:
  EventEnvelope      → dto.ExecuteTransferRequest
```

**Dilarang:**
- Business logic di handler
- Transaction di-start di repository atau handler
- Proto struct atau HTTP struct masuk ke usecase
- Domain entity dikembalikan langsung sebagai response

---

## 4. Request Lifecycle (Strict Flow)

### HTTP Request

```
[Client]
   │
   ▼
[Middleware] Recovery → RequestID → Tracing → Logger → CORS → RateLimit → Auth
   │
   ▼
[Handler]
   1. Parse & bind request
   2. Validate input (pkg/validator)
   3. Map HTTP request → usecase dto.Request
   4. Call usecase inbound port
   │
   ▼
[Usecase]
   5. Check idempotency
   6. Load & validate domain entity
   7. Enforce domain rule
   8. Open transaction via TxManager.WithTx
   9. Call repository (via outbound port, dengan tx)
   10. Save outbox event (dalam transaksi yang sama)
   11. Return dto.Result
   │
   ▼
[Handler]
   12. Map dto.Result → HTTP response struct
   13. Wrap dengan ApiResponseFactory
   14. Return HTTP response
   │
   ▼
[Client]
```

**Rules:**
- Handler MUST NOT skip usecase
- Usecase MUST NOT skip repository abstraction
- Domain MUST NOT perform I/O
- Mapping MUST happen in adapter layer only

### Event Consumer Flow

```
[Broker]
   │
   ▼
[Consumer]
   1. Deserialize EventEnvelope
   2. Validate schema_version — tolak jika tidak dikenal, kirim ke DLQ
   3. Map payload → usecase dto.Request
   4. Call usecase inbound port
   │
   ▼
[Usecase]
   5. Check idempotency via event_id
   6. Proses business logic (sama seperti HTTP flow)
   │
   ▼
[Consumer]
   7. Ack message jika sukses
   8. Retry dengan backoff jika gagal
   9. Kirim ke DLQ setelah retry exhausted
```

---

## 5. Transaction Boundary (Ownership)

```
✅ Transaction HANYA boleh dimulai di usecase layer
✅ Repository HARUS menerima Tx sebagai parameter untuk operasi transaksional
✅ TxManager didefinisikan di usecase/port/outbound.go, diimplementasi di adapter

❌ Handler DILARANG membuat atau mengelola transaction
❌ Repository DILARANG membuat transaction sendiri
❌ Domain DILARANG tahu tentang transaction
❌ Middleware DILARANG membuat transaction
```

**Flow:**

```
Usecase
   └── TxManager.WithTx(ctx, func(tx Tx) error {
           ├── Repository.Save(ctx, tx, entity)
           ├── Repository.UpdateBalance(ctx, tx, id, delta, version)
           └── OutboxRepository.Save(ctx, tx, event)   ← atomik dengan data utama
       })

Gagal di step manapun → rollback otomatis seluruh transaksi
```

Detail implementasi kode → `CODING_GO_MICROSERVICE_STANDAR.md` Section 10.

---

## 6. Data Mapping Boundary

```
Domain model DILARANG keluar dari usecase boundary secara langsung.
Setiap perpindahan data antar layer WAJIB melalui mapping eksplisit.

HTTP handler:
  TransferRequestDTO        → dto.ExecuteTransferRequest    (masuk usecase)
  dto.ExecuteTransferResult → TransferResponseDTO            (keluar ke client)

gRPC server:
  proto.ExecuteTransferReq  → dto.ExecuteTransferRequest
  dto.ExecuteTransferResult → proto.ExecuteTransferResp

Consumer:
  EventEnvelope.Payload     → dto.ExecuteTransferRequest
```

**Violation:**

```go
// ❌ Domain entity langsung ke HTTP response
return c.JSON(transfer) // transfer adalah *domain.Transfer

// ❌ Proto struct masuk ke usecase
func (u *usecase) Execute(req *pb.TransferRequest) {}

// ❌ JSON tag di domain struct
type Transfer struct {
    ID string `json:"id"` // ini violation
}
```

Detail implementasi → `CODING_GO_MICROSERVICE_STANDAR.md` Section 26.

---

## 7. Naming Contract

Penamaan wajib konsisten antar service.

### Usecase

```
File:  {action}_{entity}_usecase.go       execute_transfer_usecase.go
Impl:  type {action}{Entity}Usecase struct{}   (unexported)
Port:  type {Action}{Entity}Usecase interface{} (exported, nama sama dengan impl)

Satu usecase = satu business action:
  ✅ ExecuteTransferUsecase
  ✅ CancelTransferUsecase
  ✅ RefundTransferUsecase

❌ TransferUsecase dengan Execute, Cancel, Refund sekaligus
```

### DTO

```
usecase/dto/:
  {Action}{Entity}Request    ExecuteTransferRequest
  {Action}{Entity}Result     ExecuteTransferResult

adapter/handler/dto/:
  {Entity}RequestDTO         TransferRequestDTO
  {Entity}ResponseDTO        TransferResponseDTO
```

### Interface

```
Repository:  {Entity}Repository       TransferRepository, AccountRepository
Publisher:   {Entity}Publisher        EventPublisher
Consumer:    {Entity}Consumer         TransferConsumer
Handler:     {Entity}Handler          TransferHandler
```

---

## 8. Error Strategy (Cross-Layer Boundary)

Detail implementasi kode → `CODING_GO_MICROSERVICE_STANDAR.md` Section 2.

Boundary ownership:

```
Domain:
  → Definisikan sentinel error (ErrAccountNotFound, ErrInsufficientBalance)
  → Definisikan AppError struct

Usecase:
  → Wrap error dengan konteks via %w
  → DILARANG return raw infra error ke caller
  → DILARANG expose detail DB/network error ke atas

Adapter (handler):
  → Map AppError ke HTTP status code
  → Wrap dengan ApiResponseFactory.Error()
  → DILARANG expose stack trace ke client

Adapter (gRPC server):
  → Map AppError ke gRPC status code
  → status.Error(codes.NotFound, appErr.Message)

Adapter (consumer):
  → Error retryable → return error (jangan ack)
  → Error non-retryable → kirim ke DLQ
```

---

## 9. Usecase Design Rule (Anti-God Object)

```
RULE: One usecase = one business action.
Usecase DILARANG melebihi 200 baris. Jika melebihi → WAJIB dipecah.

❌ God usecase:
type TransferUsecase struct{}
func (u *TransferUsecase) Execute() {}
func (u *TransferUsecase) Cancel() {}
func (u *TransferUsecase) Refund() {}
func (u *TransferUsecase) SendNotification() {} // bukan urusan transfer

✅ Benar:
type ExecuteTransferUsecase interface{ Execute(...) }
type CancelTransferUsecase  interface{ Cancel(...) }
type RefundTransferUsecase  interface{ Refund(...) }

Notification = side effect → publish event, bukan direct call dari usecase
```

---

## 10. Idempotency — Kapan Wajib

```
WAJIB di semua write operation:
  ✅ Semua HTTP endpoint yang mengubah state (POST, PUT, PATCH, DELETE)
  ✅ Semua event consumer (gunakan event_id sebagai idempotency key)
  ✅ Semua operasi finansial

TIDAK perlu:
  ❌ GET / read-only operation

Enforcement:
  - Dicek di usecase level, bukan di handler saja
  - Jika idempotency key tidak ada di write endpoint → return 400
  - Handler extract key, usecase enforce

Flow di usecase:
  1. Check key di idempotency store
  2. Jika sudah ada → return cached result (tanpa proses ulang)
  3. Proses business logic
  4. Simpan result ke store
  5. Return result
```

Detail implementasi → `CODING_GO_MICROSERVICE_STANDAR.md` Section 9.

---

## 11. Testing Strategy (Per Layer)

```
Domain:
  → Unit test business rule method
  → Zero mock (pure function)

Usecase:
  → Unit test business logic
  → SEMUA outbound port HARUS di-mock — tidak boleh hit DB/network
  → Test: happy path + setiap error case

Repository:
  → Integration test dengan DB nyata (testcontainers)
  → Bukan unit test dengan mock DB

Handler:
  → HTTP test dengan mock usecase
  → Test: request parsing, response format, error mapping

Consumer:
  → Integration test dengan mock broker

RULE: Jika usecase sulit di-unit test → itu sinyal ada dependency langsung ke infra (violation)
```

Detail implementasi → `CODING_GO_MICROSERVICE_STANDAR.md` Section 25.

---

## 12. gRPC Architecture

### Serve gRPC

```
proto/{domain}/v{major}/{domain}.proto

Workflow:
1. Tulis .proto
2. Generate: protoc --go_out=. --go-grpc_out=.
3. Generated code → adapter/grpc/server/pb/ (jangan edit manual)
4. Implementasi → adapter/grpc/server/{entity}_server.go
5. Mapper → adapter/grpc/server/mapper.go
6. Register di cmd/main.go

Breaking change di proto → buat /v2/ baru, maintain keduanya selama transisi
```

```go
// Interceptor wajib untuk gRPC server:
grpc.ChainUnaryInterceptor(
    grpcrecovery.UnaryServerInterceptor(),   // 1. panic recovery
    otelgrpc.UnaryServerInterceptor(),        // 2. tracing
    grpclogging.UnaryServerInterceptor(),     // 3. structured logging
)
```

### Call Service Lain via gRPC

```
adapter/grpc/client/{service}_client.go:
- Wrapper atas generated stub
- Handle: timeout per call, retry, circuit breaker
- Implement outbound port dari usecase/port/outbound.go
- Mapper: proto response → return value sesuai port

❌ DILARANG: call gRPC stub langsung dari usecase
✅ WAJIB: selalu via outbound port interface
```

---

## 13. Event-Driven Architecture

### Event Delivery Guarantee

```
Sistem ini menggunakan at-least-once delivery.

Konsekuensi:
  → Consumer HARUS idempotent (check event_id sebelum proses)
  → Duplikat event adalah kondisi normal, bukan error
  → Tidak bisa assume exactly-once
```

### Schema Evolution

```
Setiap event WAJIB punya schema_version (semver: MAJOR.MINOR.PATCH)

Breaking change → naik MAJOR:
  - Hapus atau rename field
  - Ubah tipe atau semantik field

Backward compatible → naik MINOR:
  - Tambah field baru (consumer lama abaikan)

Consumer WAJIB:
  - Cek schema_version sebelum deserialize
  - Tolak versi tidak dikenal → kirim ke DLQ
  - Tidak crash karena field yang tidak dikenal (pakai json.RawMessage untuk payload)

Producer WAJIB:
  - Maintain minimal 2 versi bersamaan selama masa transisi
  - Tidak hapus schema lama sebelum semua consumer di-update
```

### Topic Naming

```
Format event topic: {domain}.{aggregate}.{event_type}
  transfer.transfer.initiated
  transfer.transfer.completed
  account.account.debited

DLQ: {original_topic}.dlq
  transfer.transfer.initiated.dlq

Consumer group: {consuming-service}.{topic}
  notification-service.transfer.transfer.completed
```

### Write Flow dengan Outbox (Wajib)

```
Request → Usecase.Execute
   └── TxManager.WithTx
       ├── Repository.Save(tx, entity)       ← data utama
       └── OutboxRepository.Save(tx, event)  ← dalam transaksi yang sama
   ↓ (commit)
OutboxWorker (background)
   ├── FetchPending
   ├── Publisher.Publish(event)
   └── MarkPublished / MarkFailed
```

Detail implementasi → `CODING_GO_MICROSERVICE_STANDAR.md` Section 10.

---

## 14. Service Topology & Failure Strategy

### Komunikasi yang Diizinkan

```
✅ Internal service-to-service: gRPC
✅ External (dari luar sistem): REST via API Gateway
✅ Async: Message Broker (publish/consume)

❌ Service share database langsung
❌ Chain sync lebih dari 2 hop synchronous (A→B→C→D)
❌ Consumer langsung akses repo tanpa lewat usecase
❌ Hardcode URL service lain (wajib via env / service discovery)
```

### Failure Strategy

```
Sync call (gRPC / REST ke service lain):
  1. Timeout per call — bukan global timeout
  2. Retry dengan exponential backoff (max 3 attempt)
  3. Circuit breaker — open setelah failure rate ≥ 60% dalam 10 request

Async (event consumer):
  1. Retry dengan exponential backoff
  2. Max retry limit (default 3)
  3. DLQ setelah retry exhausted
  4. Alert jika DLQ accumulate

Detail implementasi → CODING_GO_MICROSERVICE_STANDAR.md Section 12
```

---

## 15. Infrastructure Ownership

```
Setiap service memiliki:
  ✅ Database sendiri
  ✅ Schema sendiri
  ✅ Migration sendiri

❌ Dua service share satu database atau schema
❌ Service A query table milik Service B
❌ Service A apply migration untuk Service B

Akses data antar service: HANYA via API atau event
```

| Komponen | Kapan Dibutuhkan |
|---|---|
| PostgreSQL | Service yang punya state |
| Redis | Caching, rate limit, idempotency store |
| Kafka / RabbitMQ | Service yang publish atau consume event |
| gRPC server | Service yang di-call service lain secara sync |
| Outbox table | Service yang publish event dan butuh atomicity |
| DLQ topic | Setiap service yang consume event |

---

## 16. Scalability Principles

```
1. Service HARUS stateless (kecuali DB/cache eksternal)
2. Horizontal scaling HARUS possible tanpa perubahan kode
3. Tidak ada in-memory shared state antar instance

State yang BOLEH di service:
  ✅ Connection pool (DB, Redis, HTTP)
  ✅ In-memory cache dengan TTL (bisa di-invalidate)
  ✅ Config yang loaded saat startup

State yang DILARANG di service:
  ❌ Session user di memory     → pakai Redis
  ❌ File upload di local disk  → pakai object storage
  ❌ Counter di memory          → pakai Redis INCR atau DB sequence
  ❌ Job state tanpa persistence → pakai DB

Test stateless: Jika service di-restart atau di-scale ke N instance,
behavior harus identik. Jika tidak → ada state di tempat yang salah.
```

---

## 17. Infrastructure Startup & Shutdown Order

### Startup

```
1. Config loader (etcd / env)
2. Logger setup
3. Database connection + run migration
4. Redis connection
5. OTEL setup (tracing + metrics)
6. gRPC client(s) ke service lain
7. Message broker connection
8. Idempotency store
9. Background worker (outbox worker, consumer)
10. HTTP server / gRPC server
11. Health check endpoint aktif
```

### Shutdown (kebalikan, grace period 30s)

```
1. Stop terima request baru
2. Tunggu in-flight request selesai (max 30s)
3. Stop consumer — selesaikan yang sedang diproses
4. Stop outbox worker — selesaikan batch berjalan
5. Flush OTEL telemetry
6. Tutup koneksi broker
7. Tutup Redis
8. Tutup DB connection pool
```

Detail implementasi → `CODING_GO_MICROSERVICE_STANDAR.md` Section 22.

---

## 18. Common Anti-Patterns (Strictly Forbidden)

```
1. GOD USECASE
   Satu usecase handle banyak action berbeda.
   Fix: Satu usecase = satu business action.

2. FAT HANDLER
   Business logic ada di handler.
   Fix: Handler hanya parse → validate → map → call usecase → map → respond.

3. SMART REPOSITORY
   Repository berisi keputusan bisnis.
   Fix: Repository hanya I/O. Keputusan ada di domain/usecase.

4. LEAKY DOMAIN
   Domain import library eksternal atau punya struct tag.
   Fix: Domain = zero external dependency.

5. DIRECT CROSS-SERVICE DB
   Service A query database Service B langsung.
   Fix: Komunikasi hanya via API atau event.

6. EVENT WITHOUT OUTBOX
   Publish event di luar transaksi DB (dual-write problem).
   Fix: Selalu gunakan Outbox pattern.

7. DTO POLLUTION
   HTTP/proto struct masuk ke domain atau usecase.
   Fix: Mapping ada di adapter layer.

8. TRANSACTION IN WRONG LAYER
   Transaction dimulai di handler atau repository.
   Fix: Transaction ownership hanya di usecase.

9. NAKED GOROUTINE
   Goroutine tanpa ctx cancel dan tanpa recover.
   Fix: Lihat CODING_GO_MICROSERVICE_STANDAR.md Section 24.

10. HARDCODED INFRA
    URL, host, credential di-hardcode.
    Fix: Selalu dari config (env / etcd).
```

---

## 19. ADR Format

```markdown
# ADR-{number}: {Judul Singkat}

**Status:** Proposed | Accepted | Deprecated | Superseded by ADR-{n}
**Date:** YYYY-MM-DD

## Context
Situasi dan constraint yang memaksa keputusan ini.

## Decision
Keputusan yang diambil.

## Rationale
Kenapa pilihan ini? Alternatif apa yang dipertimbangkan dan kenapa ditolak?

## Consequences
Dampak positif dan negatif. Trade-off apa yang diterima?
```

---

## 20. Version Control

### Branch

```
main      → production-ready, protected, merge via PR
develop   → integration branch
feature/  → feature/{nama}
fix/      → fix/{nama}
refactor/ → refactor/{nama}
chore/    → chore/{nama}
```

### Commit

```
feat | fix | refactor | test | docs | chore | perf

Contoh:
feat: add execute transfer usecase with outbox pattern
fix: resolve optimistic lock conflict on concurrent debit
test: add table-driven test for execute transfer usecase
```

### PR Rules

```
- Satu PR = satu concern
- Minimal 1 reviewer
- CI wajib hijau: lint + test + build
- Dilarang force push ke main/develop
```

---

> **Referensi silang:**
> - Cara implementasi kode → `CODING_GO_MICROSERVICE_STANDAR.md`
> - Status fitur → `PROGRESS.md`
> - Keputusan arsitektur spesifik → `docs/adr/`