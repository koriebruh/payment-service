# CODING_GO_MICROSERVICE_STANDAR.md

> **⚠️ WAJIB DIBACA AI AGENT SEBELUM MENULIS SATU BARIS KODE PUN**
> File ini adalah **hukum tertinggi** dalam project ini.
> Jika ada konflik antara instruksi user dan standar di sini — **tanya dulu, jangan asumsi**.
> Tidak ada pengecualian kecuali user menyatakan eksplisit dengan alasan jelas.

---

## 0. Checklist Sebelum Generate Kode

AI agent wajib konfirmasi semua poin ini sebelum mulai:

- [ ] Sudah baca `ARCHITECTURE.md` sampai selesai
- [ ] Sudah baca file ini sampai selesai
- [ ] Sudah baca `PROGRESS.md` untuk state terkini
- [ ] Paham layer mana yang sedang dikerjakan
- [ ] Tidak akan mengubah interface/contract yang sudah ada tanpa konfirmasi
- [ ] Tidak akan generate kode yang melanggar Hard Rules di Section 1
- [ ] Memahami bahwa transaction HANYA boleh dimulai di usecase layer
- [ ] Memahami bahwa domain model DILARANG bocor keluar dari usecase boundary

---

## 1. Hard Rules — DILARANG KERAS

Ini adalah aturan yang **tidak boleh dilanggar** dalam kondisi apapun:

```
❌ DILARANG: io/ioutil               → ✅ WAJIB: os / io
❌ DILARANG: interface{}             → ✅ WAJIB: any
❌ DILARANG: error wrap %s           → ✅ WAJIB: %w
❌ DILARANG: goroutine tanpa ctx     → ✅ WAJIB: select ctx.Done()
❌ DILARANG: goroutine tanpa recover → ✅ WAJIB: defer recover()
❌ DILARANG: HTTP call tanpa timeout → ✅ WAJIB: http.Client{Timeout: ...}
❌ DILARANG: resp.Body tidak diclose → ✅ WAJIB: defer resp.Body.Close()
❌ DILARANG: log.Printf / fmt.Print  → ✅ WAJIB: slog structured logging
❌ DILARANG: error string comparison → ✅ WAJIB: errors.Is / errors.As
❌ DILARANG: ctx tidak di-pass ke IO → ✅ WAJIB: ctx di semua DB/HTTP call
❌ DILARANG: hardcode config/secret  → ✅ WAJIB: env variable / config loader
❌ DILARANG: panic di business logic → ✅ WAJIB: return error
❌ DILARANG: SELECT *               → ✅ WAJIB: SELECT kolom eksplisit
❌ DILARANG: global mutable state    → ✅ WAJIB: stateless, inject dependency
❌ DILARANG: magic number            → ✅ WAJIB: named constant
❌ DILARANG: overuse pointer         → ✅ WAJIB: pointer hanya jika perlu mutasi/nil
❌ DILARANG: raw sql tanpa param     → ✅ WAJIB: parameterized query
❌ DILARANG: field sensitif di JSON  → ✅ WAJIB: json:"-" untuk password/token
❌ DILARANG: secret di git           → ✅ WAJIB: .env + .gitignore
❌ DILARANG: switch statement lama   → ✅ WAJIB: switch expression (Go 1.21+)
```

---

## 2. Error Handling

### 2.1 Prinsip Utama

- **Setiap error wajib di-handle** — tidak ada `_` untuk error kecuali ada komentar alasan jelas
- **Wrap error dengan konteks** menggunakan `%w` agar chain tetap terjaga
- **Sentinel error** untuk kondisi yang perlu dibedakan oleh caller
- **Custom error type** untuk membawa data tambahan (HTTP status, error code)

### 2.2 Struktur Error

```go
// domain/errors.go

// AppError adalah error standar seluruh aplikasi
type AppError struct {
    Code       string // machine-readable, e.g. "ACCOUNT_NOT_FOUND"
    Message    string // human-readable
    HTTPStatus int    // HTTP status code
    Err        error  // underlying error untuk chain
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// Constructor helpers
func NewNotFoundError(resource, id string) *AppError {
    return &AppError{
        Code:       resource + "_NOT_FOUND",
        Message:    fmt.Sprintf("%s with id %s not found", resource, id),
        HTTPStatus: http.StatusNotFound,
    }
}

func NewValidationError(message string) *AppError {
    return &AppError{
        Code:       "VALIDATION_ERROR",
        Message:    message,
        HTTPStatus: http.StatusUnprocessableEntity,
    }
}

func NewInternalError(err error) *AppError {
    return &AppError{
        Code:       "INTERNAL_ERROR",
        Message:    "an internal error occurred",
        HTTPStatus: http.StatusInternalServerError,
        Err:        err,
    }
}

// Sentinel errors
var (
    ErrAccountNotFound      = &AppError{Code: "ACCOUNT_NOT_FOUND", HTTPStatus: 404}
    ErrInsufficientBalance  = &AppError{Code: "INSUFFICIENT_BALANCE", HTTPStatus: 422}
    ErrDuplicateRequest     = &AppError{Code: "DUPLICATE_REQUEST", HTTPStatus: 409}
    ErrUnauthorized         = &AppError{Code: "UNAUTHORIZED", HTTPStatus: 401}
)
```

### 2.3 Cara Wrap Error

```go
// ✅ Wrap dengan konteks layer + operasi
func (u *transferUsecase) Execute(ctx context.Context, req TransferRequest) error {
    acc, err := u.accountRepo.FindByID(ctx, req.FromAccountID)
    if err != nil {
        return fmt.Errorf("transferUsecase.Execute: find source account: %w", err)
    }
    return nil
}

// ✅ Check error spesifik
if errors.Is(err, ErrAccountNotFound) {
    // handle not found
}

var appErr *AppError
if errors.As(err, &appErr) {
    // akses appErr.HTTPStatus, appErr.Code
}

// ❌ DILARANG
if err.Error() == "account not found" { }
return fmt.Errorf("failed: %s", err) // hilangkan chain
_ = someOperation()                  // ignore error tanpa alasan
```

---

## 3. Naming Convention

### 3.1 Aturan Umum

| Konteks | Convention | Contoh |
|---|---|---|
| Package | lowercase, no underscore, singkat | `usecase`, `repo`, `handler` |
| Interface | noun atau verb-er | `AccountRepository`, `EventPublisher` |
| Struct | PascalCase | `TransferService`, `AccountHandler` |
| Fungsi exported | PascalCase | `FindByID`, `Execute` |
| Fungsi unexported | camelCase | `validate`, `buildQuery` |
| Constant | PascalCase atau SCREAMING_SNAKE | `StatusActive`, `MAX_RETRY` |
| Error var | `ErrXxx` | `ErrAccountNotFound` |
| Config key (env) | SCREAMING_SNAKE | `DB_HOST`, `KAFKA_BROKERS` |
| Test func | `Test_[Func]_[Scenario]` | `Test_TransferUsecase_InsufficientBalance` |

### 3.2 Interface Naming

```go
// ✅ Interface di-define di layer yang membutuhkan, bukan yang mengimplementasi
// internal/usecase/port.go
type AccountRepository interface {
    FindByID(ctx context.Context, id string) (*domain.Account, error)
    UpdateBalance(ctx context.Context, id string, amount decimal.Decimal) error
}

type EventPublisher interface {
    Publish(ctx context.Context, event domain.Event) error
}
```

### 3.3 File Naming

```
handler/transfer_handler.go
usecase/transfer_usecase.go
repository/account_repository.go
domain/account.go
domain/errors.go
domain/events.go
```

---

## 4. API Response — ApiResponseFactory

### 4.1 Standar Response

Semua response HTTP **wajib** menggunakan struktur ini:

```go
// pkg/response/response.go

type ApiResponse[T any] struct {
    Success   bool        `json:"success"`
    Code      string      `json:"code"`
    Message   string      `json:"message"`
    Data      T           `json:"data,omitempty"`
    Error     *ErrorDetail `json:"error,omitempty"`
    Meta      *Meta        `json:"meta,omitempty"`  // untuk paginated response
    RequestID string      `json:"request_id"`
    Timestamp time.Time   `json:"timestamp"`
}

type ErrorDetail struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Fields  map[string]string `json:"fields,omitempty"` // validation error per field
}

type Meta struct {
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    TotalItems int64 `json:"total_items"`
    TotalPages int   `json:"total_pages"`
}
```

### 4.2 ApiResponseFactory

```go
// pkg/response/factory.go

type ApiResponseFactory struct{}

func NewApiResponseFactory() *ApiResponseFactory {
    return &ApiResponseFactory{}
}

// Success response dengan data
func (f *ApiResponseFactory) Success[T any](
    requestID string,
    code string,
    message string,
    data T,
) ApiResponse[T] {
    return ApiResponse[T]{
        Success:   true,
        Code:      code,
        Message:   message,
        Data:      data,
        RequestID: requestID,
        Timestamp: time.Now().UTC(),
    }
}

// Success response tanpa data (create, delete, update)
func (f *ApiResponseFactory) SuccessNoData(
    requestID string,
    code string,
    message string,
) ApiResponse[any] {
    return ApiResponse[any]{
        Success:   true,
        Code:      code,
        Message:   message,
        RequestID: requestID,
        Timestamp: time.Now().UTC(),
    }
}

// Paginated response
func (f *ApiResponseFactory) SuccessPaginated[T any](
    requestID string,
    data T,
    meta Meta,
) ApiResponse[T] {
    return ApiResponse[T]{
        Success:   true,
        Code:      "SUCCESS",
        Message:   "data retrieved successfully",
        Data:      data,
        Meta:      &meta,
        RequestID: requestID,
        Timestamp: time.Now().UTC(),
    }
}

// Error response
func (f *ApiResponseFactory) Error(
    requestID string,
    appErr *domain.AppError,
) ApiResponse[any] {
    return ApiResponse[any]{
        Success: false,
        Code:    appErr.Code,
        Message: appErr.Message,
        Error: &ErrorDetail{
            Code:    appErr.Code,
            Message: appErr.Message,
        },
        RequestID: requestID,
        Timestamp: time.Now().UTC(),
    }
}

// Validation error response
func (f *ApiResponseFactory) ValidationError(
    requestID string,
    fields map[string]string,
) ApiResponse[any] {
    return ApiResponse[any]{
        Success: false,
        Code:    "VALIDATION_ERROR",
        Message: "request validation failed",
        Error: &ErrorDetail{
            Code:    "VALIDATION_ERROR",
            Message: "one or more fields are invalid",
            Fields:  fields,
        },
        RequestID: requestID,
        Timestamp: time.Now().UTC(),
    }
}
```

### 4.3 Contoh Penggunaan di Handler

```go
func (h *TransferHandler) Transfer(c *fiber.Ctx) error {
    requestID := c.Locals("request_id").(string)

    var req TransferRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(
            h.factory.Error(requestID, domain.NewValidationError("invalid request body")),
        )
    }

    result, err := h.usecase.Execute(c.Context(), req)
    if err != nil {
        var appErr *domain.AppError
        if errors.As(err, &appErr) {
            return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
        }
        return c.Status(500).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
    }

    return c.Status(201).JSON(h.factory.Success(requestID, "TRANSFER_CREATED", "transfer initiated", result))
}
```

---

## 5. Centralized Config

### 5.1 Hierarki Config (prioritas dari tertinggi)

```
1. etcd (production / k8s)         ← tertinggi, runtime dynamic
2. Environment Variable             ← override lokal / container
3. .env file                        ← local development
4. Default value                    ← fallback terakhir
```

### 5.2 Struktur Config

```go
// config/config.go

type Config struct {
    App      AppConfig
    DB       DBConfig
    Redis    RedisConfig
    Kafka    KafkaConfig
    OTEL     OTELConfig
    Etcd     EtcdConfig
    Auth     AuthConfig
    RateLimit RateLimitConfig
}

type AppConfig struct {
    Name        string        `env:"APP_NAME"        envDefault:"service"`
    Env         string        `env:"APP_ENV"         envDefault:"development"`
    Port        int           `env:"APP_PORT"        envDefault:"8080"`
    Timeout     time.Duration `env:"APP_TIMEOUT"     envDefault:"30s"`
    Version     string        `env:"APP_VERSION"     envDefault:"v1"`
}

type DBConfig struct {
    Host            string        `env:"DB_HOST"             envDefault:"localhost"`
    Port            int           `env:"DB_PORT"             envDefault:"5432"`
    Name            string        `env:"DB_NAME,required"`
    User            string        `env:"DB_USER,required"`
    Password        string        `env:"DB_PASSWORD,required"`
    MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS"   envDefault:"25"`
    MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS"   envDefault:"10"`
    ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m"`
    ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE"    envDefault:"1m"`
}

type OTELConfig struct {
    Enabled        bool   `env:"OTEL_ENABLED"         envDefault:"false"`
    Endpoint       string `env:"OTEL_ENDPOINT"        envDefault:"localhost:4317"`
    ServiceName    string `env:"OTEL_SERVICE_NAME"`
    MetricsEnabled bool   `env:"OTEL_METRICS_ENABLED" envDefault:"true"`
    TracingEnabled bool   `env:"OTEL_TRACING_ENABLED" envDefault:"true"`
    SampleRate     float64 `env:"OTEL_SAMPLE_RATE"    envDefault:"1.0"`
}

type EtcdConfig struct {
    Enabled   bool          `env:"ETCD_ENABLED"  envDefault:"false"`
    Endpoints []string      `env:"ETCD_ENDPOINTS" envSeparator:","`
    Prefix    string        `env:"ETCD_PREFIX"   envDefault:"/config"`
    Timeout   time.Duration `env:"ETCD_TIMEOUT"  envDefault:"5s"`
}

type RateLimitConfig struct {
    Enabled     bool          `env:"RATE_LIMIT_ENABLED"      envDefault:"true"`
    MaxRequests int           `env:"RATE_LIMIT_MAX_REQUESTS" envDefault:"100"`
    Window      time.Duration `env:"RATE_LIMIT_WINDOW"       envDefault:"1m"`
    ByIP        bool          `env:"RATE_LIMIT_BY_IP"        envDefault:"true"`
}
```

### 5.3 Config Loader dengan etcd Fallback

```go
// config/loader.go

func Load() (*Config, error) {
    // 1. Load dari env / .env file dulu sebagai base
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("config.Load: parse env: %w", err)
    }

    // 2. Jika etcd enabled, override dengan nilai dari etcd
    if cfg.Etcd.Enabled {
        if err := loadFromEtcd(cfg); err != nil {
            // ⚠️ Gagal ke etcd tidak fatal — fallback ke env yang sudah di-load
            slog.Warn("config.Load: etcd unavailable, using env fallback",
                "error", err)
        }
    }

    return cfg, nil
}

func loadFromEtcd(cfg *Config) error {
    client, err := clientv3.New(clientv3.Config{
        Endpoints:   cfg.Etcd.Endpoints,
        DialTimeout: cfg.Etcd.Timeout,
    })
    if err != nil {
        return fmt.Errorf("loadFromEtcd: connect: %w", err)
    }
    defer client.Close()

    ctx, cancel := context.WithTimeout(context.Background(), cfg.Etcd.Timeout)
    defer cancel()

    resp, err := client.Get(ctx, cfg.Etcd.Prefix, clientv3.WithPrefix())
    if err != nil {
        return fmt.Errorf("loadFromEtcd: get keys: %w", err)
    }

    // Map etcd KV ke config fields via reflection / manual mapping
    for _, kv := range resp.Kvs {
        applyEtcdKV(cfg, string(kv.Key), string(kv.Value))
    }

    return nil
}
```

---

## 6. Structured Logging

### 6.1 Format & Library

- **Library:** Go standard `slog` (Go 1.21+) dengan JSON handler
- **Format:** JSON — mudah di-ingest ke ELK / Loki / Datadog
- **Level:** DEBUG (dev), INFO (prod), WARN, ERROR

### 6.2 Field Wajib di Setiap Log Entry

| Field | Tipe | Keterangan |
|---|---|---|
| `timestamp` | RFC3339 UTC | Waktu log dibuat |
| `level` | string | DEBUG/INFO/WARN/ERROR |
| `service` | string | Nama service |
| `trace_id` | string | Distributed trace ID dari OTEL |
| `request_id` | string | ID unik per request HTTP |
| `message` | string | Pesan log |

Field tambahan wajib jika konteksnya ada:

| Field | Kapan |
|---|---|
| `user_id` | Request yang sudah terautentikasi |
| `account_id` | Operasi yang berkaitan dengan akun |
| `event_type` | Saat publish/consume event |
| `duration_ms` | Setelah operasi yang perlu diukur |
| `error` | Saat log level WARN / ERROR |
| `stack_trace` | Saat panic recovery |

### 6.3 Setup Logger

```go
// pkg/logger/logger.go

func New(cfg *config.AppConfig) *slog.Logger {
    opts := &slog.HandlerOptions{
        Level:     parseLevel(cfg.LogLevel),
        AddSource: cfg.Env != "production", // source hanya di non-prod
    }

    handler := slog.NewJSONHandler(os.Stdout, opts)

    return slog.New(handler).With(
        "service", cfg.Name,
        "version", cfg.Version,
        "env",     cfg.Env,
    )
}

// Inject ke context agar bisa diambil di mana saja
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}
```

### 6.4 Cara Logging yang Benar

```go
// ✅ Structured — semua key-value, tidak ada string formatting
slog.Info("transfer processed",
    "trace_id",    traceID,
    "request_id",  requestID,
    "transfer_id", transferID,
    "amount",      amount,
    "duration_ms", elapsed.Milliseconds(),
)

slog.Error("transfer failed",
    "trace_id",   traceID,
    "request_id", requestID,
    "error",      err.Error(),
)

// ❌ DILARANG
log.Printf("transfer %s processed in %dms", id, ms)
fmt.Println("error:", err)
slog.Info(fmt.Sprintf("transfer %s done", id)) // jangan format di pesan
```

---

## 7. Observability — Metrics & Tracing

### 7.1 Toggle On/Off via Config

```go
// Semua observability harus bisa dimatikan via config
// OTEL_ENABLED=false → skip setup, no-op provider
// OTEL_METRICS_ENABLED=false → skip metrics
// OTEL_TRACING_ENABLED=false → skip tracing

func SetupOTEL(ctx context.Context, cfg *config.OTELConfig) (shutdown func(), err error) {
    if !cfg.Enabled {
        // Return no-op provider agar kode di atas tidak perlu if-check
        otel.SetTracerProvider(trace.NewNoopTracerProvider())
        return func() {}, nil
    }

    // Setup OTLP exporter ke collector
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("SetupOTEL: create exporter: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(cfg.ServiceName),
        )),
    )
    otel.SetTracerProvider(tp)

    shutdown = func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }
    return shutdown, nil
}
```

### 7.2 Metrics Wajib

```go
// pkg/metrics/metrics.go
// Wajib ada metric ini di setiap service:

var (
    // HTTP request duration histogram
    HTTPRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status_code"},
    )

    // HTTP request total counter
    HTTPRequestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status_code"},
    )

    // DB query duration
    DBQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "table"},
    )

    // Message broker published/consumed
    EventPublishedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "events_published_total",
            Help: "Total number of events published",
        },
        []string{"topic", "event_type", "status"},
    )

    EventConsumedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "events_consumed_total",
            Help: "Total number of events consumed",
        },
        []string{"topic", "event_type", "status"},
    )

    // Active goroutines
    ActiveGoroutines = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_goroutines",
            Help: "Number of active goroutines",
        },
    )
)
```

### 7.3 Tracing Convention

```go
// ✅ Setiap usecase dan repository wajib create span
func (u *transferUsecase) Execute(ctx context.Context, req TransferRequest) (*TransferResult, error) {
    ctx, span := otel.Tracer("transfer-service").Start(ctx, "TransferUsecase.Execute")
    defer span.End()

    span.SetAttributes(
        attribute.String("transfer.from_account_id", req.FromAccountID),
        attribute.String("transfer.to_account_id", req.ToAccountID),
        attribute.Float64("transfer.amount", req.Amount),
    )

    result, err := u.process(ctx, req)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }

    return result, nil
}
```

---

## 8. Event / Message Schema

### 8.1 Standar Payload Event

**Setiap event wajib mengikuti envelope ini:**

```go
// domain/events.go

type EventEnvelope struct {
    EventID       string          `json:"event_id"`        // UUID unik per event
    EventType     string          `json:"event_type"`      // TRANSFER_INITIATED, ACCOUNT_DEBITED, dll
    SchemaVersion string          `json:"schema_version"`  // "1.0.0" — WAJIB ADA
    AggregateID   string          `json:"aggregate_id"`    // ID entitas utama (transfer_id, account_id)
    AggregateType string          `json:"aggregate_type"`  // "transfer", "account"
    ServiceSource string          `json:"service_source"`  // nama service yang publish
    TraceID       string          `json:"trace_id"`        // untuk distributed tracing
    CorrelationID string          `json:"correlation_id"`  // untuk menghubungkan event chain
    OccurredAt    time.Time       `json:"occurred_at"`     // waktu event terjadi (bukan publish)
    PublishedAt   time.Time       `json:"published_at"`    // waktu event di-publish
    Payload       json.RawMessage `json:"payload"`         // data spesifik event
    Metadata      map[string]string `json:"metadata,omitempty"` // context tambahan
}

// Contoh payload spesifik
type TransferInitiatedPayload struct {
    TransferID    string          `json:"transfer_id"`
    FromAccountID string          `json:"from_account_id"`
    ToAccountID   string          `json:"to_account_id"`
    Amount        decimal.Decimal `json:"amount"`
    Currency      string          `json:"currency"`
    IdempotencyKey string         `json:"idempotency_key"`
}
```

### 8.2 Schema Version Rules

```
- Format: MAJOR.MINOR.PATCH (semver)
- MAJOR naik jika ada breaking change (field dihapus / tipe berubah)
- MINOR naik jika ada field baru (backward compatible)
- PATCH naik jika ada perbaikan tanpa perubahan struktur
- Consumer WAJIB check schema_version sebelum deserialize
- Consumer WAJIB tolak event dengan schema version yang tidak dikenal
```

### 8.3 Event Publisher Interface

```go
type EventPublisher interface {
    Publish(ctx context.Context, topic string, event EventEnvelope) error
}

// ✅ Selalu set trace_id saat publish
func buildEvent(ctx context.Context, eventType string, payload any) EventEnvelope {
    payloadBytes, _ := json.Marshal(payload)
    traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

    return EventEnvelope{
        EventID:       uuid.NewString(),
        EventType:     eventType,
        SchemaVersion: "1.0.0",
        ServiceSource: "transfer-service",
        TraceID:       traceID,
        OccurredAt:    time.Now().UTC(),
        PublishedAt:   time.Now().UTC(),
        Payload:       payloadBytes,
    }
}
```

---

## 9. Idempotency

### 9.1 Kapan Wajib Digunakan

- Semua operasi **write** yang bisa di-retry oleh client (transfer, payment, order create)
- Semua consumer event yang memproses pesan dari message broker
- **TIDAK perlu** untuk operasi read (GET)

### 9.2 Implementasi

```go
// Client wajib kirim header atau field ini:
// Header: X-Idempotency-Key: <uuid>
// atau field: idempotency_key di request body

// pkg/idempotency/idempotency.go
type IdempotencyStore interface {
    // Get existing result jika key sudah pernah diproses
    Get(ctx context.Context, key string) (*IdempotencyRecord, error)
    // Set result setelah operasi sukses
    Set(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error
}

type IdempotencyRecord struct {
    Key        string          `json:"key"`
    StatusCode int             `json:"status_code"`
    Response   json.RawMessage `json:"response"`
    CreatedAt  time.Time       `json:"created_at"`
}

// Middleware idempotency untuk handler
func IdempotencyMiddleware(store IdempotencyStore) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("X-Idempotency-Key")
        if key == "" {
            // Jika endpoint wajib idempotency, tolak request tanpa key
            // Jika opsional, lanjutkan
            return c.Next()
        }

        // Cek apakah key sudah pernah diproses
        record, err := store.Get(c.Context(), key)
        if err == nil && record != nil {
            // Return cached response
            return c.Status(record.StatusCode).JSON(record.Response)
        }

        // Simpan state "in-progress" untuk mencegah concurrent duplicate
        // Proses request
        if err := c.Next(); err != nil {
            return err
        }

        // Simpan hasil ke store
        return nil
    }
}
```

---

## 10. Transactional Outbox Pattern

### 10.1 Konsep

Saat service perlu **update DB + publish event secara atomik**, gunakan Outbox Pattern:
1. Simpan event ke tabel `outbox` dalam transaksi DB yang sama dengan data utama
2. Background worker membaca outbox dan publish ke message broker
3. Hapus / mark sebagai published setelah berhasil

**Tujuan:** Jamin konsistensi antara state DB dan event yang dipublish (no dual-write problem).

### 10.2 Tabel Outbox

```sql
-- migrations/xxx_create_outbox_table.sql
CREATE TABLE outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id    VARCHAR(255) NOT NULL,
    aggregate_type  VARCHAR(100) NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    schema_version  VARCHAR(20)  NOT NULL DEFAULT '1.0.0',
    payload         JSONB        NOT NULL,
    trace_id        VARCHAR(255),
    correlation_id  VARCHAR(255),
    status          VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    retry_count     INT          NOT NULL DEFAULT 0,
    max_retries     INT          NOT NULL DEFAULT 3,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ,
    failed_at       TIMESTAMPTZ,
    error_message   TEXT
);

CREATE INDEX idx_outbox_status ON outbox_events(status, created_at)
    WHERE status = 'PENDING';
```

### 10.3 Implementasi

```go
// domain/outbox.go
type OutboxEvent struct {
    ID            string
    AggregateID   string
    AggregateType string
    EventType     string
    SchemaVersion string
    Payload       json.RawMessage
    TraceID       string
    CorrelationID string
    Status        string // PENDING, PUBLISHED, FAILED
    RetryCount    int
    MaxRetries    int
}

// repository/outbox_repository.go
type OutboxRepository interface {
    Save(ctx context.Context, tx *sql.Tx, event OutboxEvent) error
    FetchPending(ctx context.Context, limit int) ([]OutboxEvent, error)
    MarkPublished(ctx context.Context, id string) error
    MarkFailed(ctx context.Context, id string, errMsg string) error
    IncrementRetry(ctx context.Context, id string) error
}

// usecase: simpan data + outbox dalam satu transaksi
func (u *transferUsecase) Execute(ctx context.Context, req TransferRequest) error {
    return u.db.WithTx(ctx, func(tx *sql.Tx) error {
        // 1. Update data utama
        if err := u.transferRepo.Create(ctx, tx, transfer); err != nil {
            return fmt.Errorf("create transfer: %w", err)
        }

        // 2. Simpan event ke outbox dalam transaksi yang sama
        outboxEvent := OutboxEvent{
            AggregateID:   transfer.ID,
            AggregateType: "transfer",
            EventType:     "TRANSFER_INITIATED",
            SchemaVersion: "1.0.0",
            Payload:       mustMarshal(transferPayload),
            TraceID:       traceID,
        }
        if err := u.outboxRepo.Save(ctx, tx, outboxEvent); err != nil {
            return fmt.Errorf("save outbox event: %w", err)
        }

        return nil
    })
}

// background worker: publish outbox events
func (w *OutboxWorker) Run(ctx context.Context) {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processOutbox(ctx)
        }
    }
}

func (w *OutboxWorker) processOutbox(ctx context.Context) {
    events, err := w.outboxRepo.FetchPending(ctx, 10)
    if err != nil {
        slog.Error("outbox: fetch pending failed", "error", err)
        return
    }

    for i := 0; i < len(events); i++ {
        if err := w.publisher.Publish(ctx, events[i]); err != nil {
            slog.Error("outbox: publish failed",
                "event_id", events[i].ID,
                "error", err)
            if events[i].RetryCount >= events[i].MaxRetries {
                w.outboxRepo.MarkFailed(ctx, events[i].ID, err.Error())
            } else {
                w.outboxRepo.IncrementRetry(ctx, events[i].ID)
            }
            continue
        }
        w.outboxRepo.MarkPublished(ctx, events[i].ID)
    }
}
```

---

## 11. Dead Letter Queue (DLQ)

### 11.1 Kapan Event Masuk DLQ

- Retry sudah mencapai batas maksimal
- Schema version tidak dikenal consumer
- Payload corrupt / tidak bisa di-deserialize
- Business validation gagal dan tidak bisa di-retry

### 11.2 Implementasi

```go
// Setiap consumer wajib implementasi retry + DLQ fallback

type ConsumerConfig struct {
    Topic      string
    DLQTopic   string        // format: {topic}.dlq
    MaxRetries int           // default: 3
    RetryDelay time.Duration // exponential backoff base
}

func (c *Consumer) processWithRetry(ctx context.Context, msg Message) error {
    var lastErr error

    for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
        if attempt > 0 {
            backoff := time.Duration(attempt) * c.cfg.RetryDelay
            time.Sleep(backoff)
        }

        if err := c.handler(ctx, msg); err != nil {
            lastErr = err
            slog.Warn("consumer: retry",
                "attempt", attempt+1,
                "max", c.cfg.MaxRetries,
                "event_id", msg.EventID,
                "error", err)
            continue
        }
        return nil
    }

    // Kirim ke DLQ setelah semua retry gagal
    return c.sendToDLQ(ctx, msg, lastErr)
}

func (c *Consumer) sendToDLQ(ctx context.Context, msg Message, originalErr error) error {
    dlqMsg := DLQMessage{
        OriginalMessage: msg,
        FailedAt:        time.Now().UTC(),
        Reason:          originalErr.Error(),
        ServiceName:     c.serviceName,
        RetryCount:      c.cfg.MaxRetries,
    }

    if err := c.publisher.Publish(ctx, c.cfg.DLQTopic, dlqMsg); err != nil {
        slog.Error("consumer: failed to send to DLQ",
            "event_id", msg.EventID,
            "error", err)
        return fmt.Errorf("sendToDLQ: %w", err)
    }

    slog.Warn("consumer: message sent to DLQ",
        "event_id",  msg.EventID,
        "dlq_topic", c.cfg.DLQTopic,
        "reason",    originalErr.Error())

    return nil
}
```

---

## 12. Inter-Service Communication

### 12.1 Batasan Komunikasi

```
✅ Service boleh:
   - Call service lain via gRPC (sync, internal)
   - Call service lain via REST (sync, external/public)
   - Publish event ke message broker (async)
   - Subscribe event dari message broker (async)

❌ Service DILARANG:
   - Share database langsung dengan service lain
   - Call lebih dari 2 hop sync secara berantai (A→B→C→D)
   - Hardcode URL service lain (wajib via service discovery / env)
```

### 12.2 HTTP Client Standard

```go
// pkg/httpclient/client.go — wajib dipakai, bukan http.DefaultClient

func NewHTTPClient(cfg HTTPClientConfig) *http.Client {
    return &http.Client{
        Timeout: cfg.Timeout, // wajib set, default 10s
        Transport: &http.Transport{
            MaxIdleConns:        cfg.MaxIdleConns,        // default 100
            MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,  // default 10
            IdleConnTimeout:     cfg.IdleConnTimeout,      // default 90s
            TLSHandshakeTimeout: 10 * time.Second,
        },
    }
}
```

### 12.3 Retry + Exponential Backoff

```go
// pkg/retry/retry.go

type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    var lastErr error
    delay := cfg.BaseDelay

    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        if err := fn(); err != nil {
            lastErr = err

            // Jangan retry jika context sudah cancel
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }

            if attempt < cfg.MaxAttempts-1 {
                slog.Warn("retry.WithRetry: attempt failed",
                    "attempt", attempt+1,
                    "next_delay", delay,
                    "error", err)
                time.Sleep(delay)
                delay = time.Duration(float64(delay) * cfg.Multiplier)
                if delay > cfg.MaxDelay {
                    delay = cfg.MaxDelay
                }
            }
            continue
        }
        return nil
    }

    return fmt.Errorf("retry.WithRetry: all %d attempts failed: %w",
        cfg.MaxAttempts, lastErr)
}
```

### 12.4 Circuit Breaker

```go
// Gunakan library: github.com/sony/gobreaker

func NewCircuitBreaker(name string, cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker {
    return gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        name,
        MaxRequests: cfg.MaxRequests,      // request saat half-open
        Interval:    cfg.Interval,         // window untuk counting failure
        Timeout:     cfg.Timeout,          // waktu tunggu sebelum half-open
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 5 && failureRatio >= 0.6
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            slog.Warn("circuit breaker state changed",
                "name", name,
                "from", from.String(),
                "to",   to.String())
        },
    })
}
```

---

## 13. Locking

### 13.1 Optimistic Locking (untuk konflik jarang terjadi)

```go
// Tambah kolom version di tabel
// ALTER TABLE accounts ADD COLUMN version INT NOT NULL DEFAULT 0;

// Repository update dengan version check
func (r *accountRepo) UpdateBalance(
    ctx context.Context,
    id string,
    amount decimal.Decimal,
    currentVersion int,
) error {
    result, err := r.db.ExecContext(ctx,
        `UPDATE accounts SET balance = $1, version = version + 1
         WHERE id = $2 AND version = $3`,
        amount, id, currentVersion,
    )
    if err != nil {
        return fmt.Errorf("UpdateBalance: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return ErrOptimisticLockConflict // caller bisa retry
    }
    return nil
}
```

### 13.2 Pessimistic Locking (untuk operasi finansial kritis)

```go
// Gunakan SELECT FOR UPDATE untuk lock row selama transaksi
func (r *accountRepo) FindByIDForUpdate(
    ctx context.Context,
    tx *sql.Tx,
    id string,
) (*Account, error) {
    var acc Account
    err := tx.QueryRowContext(ctx,
        `SELECT id, balance, version FROM accounts
         WHERE id = $1 FOR UPDATE`, // LOCK baris ini
        id,
    ).Scan(&acc.ID, &acc.Balance, &acc.Version)

    if err == sql.ErrNoRows {
        return nil, ErrAccountNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("FindByIDForUpdate: %w", err)
    }
    return &acc, nil
}
```

---

## 14. Health Check

### 14.1 Endpoint Wajib

```go
// GET /health  → liveness: apakah proses berjalan
// GET /ready   → readiness: apakah siap terima traffic
// GET /metrics → prometheus metrics endpoint

type HealthStatus struct {
    Status    string                 `json:"status"`    // "ok" | "degraded" | "down"
    Version   string                 `json:"version"`
    Timestamp time.Time              `json:"timestamp"`
    Checks    map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
    Status  string        `json:"status"`
    Latency time.Duration `json:"latency_ms"`
    Error   string        `json:"error,omitempty"`
}

func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
    checks := map[string]CheckResult{}
    allOK := true

    // Check DB
    dbStart := time.Now()
    if err := h.db.PingContext(c.Context()); err != nil {
        checks["database"] = CheckResult{Status: "down", Error: err.Error()}
        allOK = false
    } else {
        checks["database"] = CheckResult{Status: "ok", Latency: time.Since(dbStart)}
    }

    // Check Redis (jika dipakai)
    if h.redis != nil {
        redisStart := time.Now()
        if err := h.redis.Ping(c.Context()).Err(); err != nil {
            checks["redis"] = CheckResult{Status: "down", Error: err.Error()}
            allOK = false
        } else {
            checks["redis"] = CheckResult{Status: "ok", Latency: time.Since(redisStart)}
        }
    }

    status := HealthStatus{
        Status:    "ok",
        Version:   h.version,
        Timestamp: time.Now().UTC(),
        Checks:    checks,
    }

    if !allOK {
        status.Status = "degraded"
        return c.Status(503).JSON(status)
    }
    return c.Status(200).JSON(status)
}
```

---

## 15. Stateless Application

```
✅ State yang BOLEH ada di service:
   - In-memory cache dengan TTL (dan bisa di-invalidate)
   - Connection pool (DB, Redis, HTTP)
   - Config yang loaded saat startup

❌ State yang DILARANG ada di service:
   - Session user di memory (pakai Redis)
   - File upload di local disk (pakai object storage)
   - Sequence counter di memory (pakai DB sequence / Redis INCR)
   - Job state di memory tanpa persistence

Rule of thumb: Jika service di-restart atau di-scale ke 2 instance,
behavior harus identik. Jika tidak, berarti ada state yang salah tempat.
```

---

## 16. Database — Connection Pool & Pointer

### 16.1 Connection Pool

```go
// ✅ WAJIB konfigurasi connection pool
sqlDB, err := db.DB()
if err != nil {
    return fmt.Errorf("get sql db: %w", err)
}
sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)        // default: 25
sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)        // default: 10
sqlDB.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)  // default: 5m
sqlDB.SetConnMaxIdleTime(cfg.DB.ConnMaxIdleTime)  // default: 1m
```

### 16.2 Pointer Usage Rules

```go
// ✅ Gunakan pointer jika:
// 1. Struct besar (> ~64 bytes) dan di-pass ke banyak tempat
// 2. Perlu represent "tidak ada nilai" (nullable)
// 3. Perlu mutasi via method

// ✅ Gunakan value jika:
// 1. Struct kecil (primitive, small struct)
// 2. Tidak perlu nil
// 3. Immutable setelah dibuat

// ❌ DILARANG overuse pointer:
func processAmount(amount *decimal.Decimal) {} // ❌ decimal tidak perlu pointer
func getStatus() *string {}                    // ❌ string tidak perlu pointer

// ✅ Pointer yang tepat:
func (r *repo) FindByID(ctx context.Context, id string) (*Account, error) {}
// Account adalah domain struct yang besar, wajar pakai pointer
```

---

## 17. Rate Limiting

```go
// pkg/middleware/ratelimit.go
// Rate limit per IP — wajib pakai Redis untuk multi-instance

type RateLimiter struct {
    redis      *redis.Client
    maxRequest int
    window     time.Duration
}

func (rl *RateLimiter) Middleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        if !rl.cfg.Enabled {
            return c.Next()
        }

        ip := c.IP()
        key := fmt.Sprintf("ratelimit:%s", ip)

        count, err := rl.redis.Incr(c.Context(), key).Result()
        if err != nil {
            // Jika Redis gagal, jangan block request (fail-open)
            slog.Warn("ratelimit: redis unavailable, skipping", "error", err)
            return c.Next()
        }

        if count == 1 {
            rl.redis.Expire(c.Context(), key, rl.window)
        }

        c.Set("X-RateLimit-Limit",     fmt.Sprintf("%d", rl.maxRequest))
        c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(rl.maxRequest)-count)))

        if count > int64(rl.maxRequest) {
            return c.Status(429).JSON(fiber.Map{
                "success": false,
                "code":    "RATE_LIMIT_EXCEEDED",
                "message": "too many requests, please slow down",
            })
        }

        return c.Next()
    }
}
```

---

## 18. Input Validation

```go
// ✅ Validasi di handler layer sebelum masuk ke usecase
// Gunakan go-playground/validator

type TransferRequest struct {
    FromAccountID  string          `json:"from_account_id"  validate:"required,uuid4"`
    ToAccountID    string          `json:"to_account_id"    validate:"required,uuid4,nefield=FromAccountID"`
    Amount         decimal.Decimal `json:"amount"           validate:"required,gt=0"`
    Currency       string          `json:"currency"         validate:"required,len=3,uppercase"`
    IdempotencyKey string          `json:"idempotency_key"  validate:"required,uuid4"`
    Note           string          `json:"note"             validate:"omitempty,max=255"`
}

// pkg/validator/validator.go
func ValidateStruct(s any) map[string]string {
    validate := validator.New()
    validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
        if name == "-" { return "" }
        return name
    })

    err := validate.Struct(s)
    if err == nil {
        return nil
    }

    fieldErrors := make(map[string]string)
    var validationErrors validator.ValidationErrors
    if errors.As(err, &validationErrors) {
        for _, e := range validationErrors {
            fieldErrors[e.Field()] = buildValidationMessage(e)
        }
    }
    return fieldErrors
}
```

---

## 19. Pagination

```go
// Semua endpoint yang return list WAJIB pakai pagination

type PaginationRequest struct {
    Page    int `query:"page"     validate:"min=1"`
    PerPage int `query:"per_page" validate:"min=1,max=100"`
}

func (p *PaginationRequest) Normalize() {
    if p.Page == 0    { p.Page = 1 }
    if p.PerPage == 0 { p.PerPage = 20 }
    if p.PerPage > 100 { p.PerPage = 100 } // hard cap
}

func (p *PaginationRequest) Offset() int {
    return (p.Page - 1) * p.PerPage
}

type PaginatedResult[T any] struct {
    Items      []T   `json:"items"`
    TotalItems int64 `json:"total_items"`
    TotalPages int   `json:"total_pages"`
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    HasNext    bool  `json:"has_next"`
    HasPrev    bool  `json:"has_prev"`
}

func NewPaginatedResult[T any](items []T, total int64, req PaginationRequest) PaginatedResult[T] {
    totalPages := int(math.Ceil(float64(total) / float64(req.PerPage)))
    return PaginatedResult[T]{
        Items:      items,
        TotalItems: total,
        TotalPages: totalPages,
        Page:       req.Page,
        PerPage:    req.PerPage,
        HasNext:    req.Page < totalPages,
        HasPrev:    req.Page > 1,
    }
}
```

---

## 20. Caching

```go
// Cache hanya untuk data yang:
// 1. Sering dibaca (high read frequency)
// 2. Jarang berubah (low write frequency)
// 3. Bisa stale sebentar tanpa konsekuensi serius
// Contoh: config, reference data, product catalog, exchange rate

// pkg/cache/cache.go

type CacheConfig struct {
    DefaultTTL time.Duration
    KeyPrefix  string
}

type Cache struct {
    redis  *redis.Client
    cfg    CacheConfig
}

// Cache-Aside pattern — wajib dipakai
func GetOrSet[T any](
    ctx context.Context,
    c *Cache,
    key string,
    ttl time.Duration,
    fetch func() (T, error),
) (T, error) {
    fullKey := c.cfg.KeyPrefix + ":" + key

    // 1. Coba ambil dari cache
    val, err := c.redis.Get(ctx, fullKey).Result()
    if err == nil {
        var result T
        if jsonErr := json.Unmarshal([]byte(val), &result); jsonErr == nil {
            return result, nil
        }
    }

    // 2. Cache miss — ambil dari source
    result, err := fetch()
    if err != nil {
        var zero T
        return zero, err
    }

    // 3. Simpan ke cache
    if data, jsonErr := json.Marshal(result); jsonErr == nil {
        c.redis.Set(ctx, fullKey, data, ttl)
    }

    return result, nil
}

// Invalidation — wajib dipanggil saat data berubah
func (c *Cache) Invalidate(ctx context.Context, key string) error {
    fullKey := c.cfg.KeyPrefix + ":" + key
    return c.redis.Del(ctx, fullKey).Err()
}
```

---

## 21. API Versioning

```go
// Versi API wajib ada di URL path, bukan header
// Format: /api/v{major}

// ✅ Benar
// POST /api/v1/transfers
// GET  /api/v1/accounts/:id
// POST /api/v2/transfers  ← breaking change dari v1

// ❌ DILARANG
// POST /api/transfers     ← tanpa versi
// POST /transfers/v1      ← versi di akhir

// Routing
v1 := app.Group("/api/v1")
v1.Post("/transfers", transferHandler.Create)
v1.Get("/accounts/:id", accountHandler.FindByID)

// Saat ada breaking change:
// 1. Buat /api/v2 dengan handler baru
// 2. /api/v1 tetap jalan (deprecated, tapi tidak langsung dihapus)
// 3. Announce sunset date di response header: Sunset: Sat, 01 Jan 2026 00:00:00 GMT
// 4. Minimal 3 bulan deprecation window sebelum penghapusan

// Header untuk komunikasi deprecation
c.Set("Deprecation", "true")
c.Set("Sunset", "Sat, 01 Jan 2026 00:00:00 GMT")
c.Set("Link", "</api/v2/transfers>; rel=\"successor-version\"")
```

---

## 22. Graceful Shutdown

```go
// cmd/main.go — wajib ada di setiap service

func main() {
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // Setup OTEL
    otelShutdown, err := observability.SetupOTEL(context.Background(), &cfg.OTEL)
    if err != nil {
        slog.Error("failed to setup OTEL", "error", err)
        os.Exit(1)
    }

    app := fiber.New()
    // ... setup routes

    // Channel untuk signal OS
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

    // Jalankan server di goroutine terpisah
    go func() {
        if err := app.Listen(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
            slog.Error("server error", "error", err)
        }
    }()

    slog.Info("service started", "port", cfg.App.Port)

    // Tunggu signal
    <-quit
    slog.Info("shutdown signal received")

    // Beri waktu untuk request yang sedang diproses selesai
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Shutdown berurutan
    if err := app.ShutdownWithContext(ctx); err != nil {
        slog.Error("server shutdown error", "error", err)
    }

    otelShutdown() // flush semua telemetry

    // Tutup outbox worker, kafka consumer, dll
    slog.Info("service stopped gracefully")
}
```

---

## 23. Loop Convention

```go
// ✅ WAJIB pakai for i — eksplisit, predictable, sesuai standar project
for i := 0; i < len(items); i++ {
    process(items[i])
}

// for range boleh dipakai HANYA jika index tidak dibutuhkan
for _, item := range items {
    process(item)
}

// ❌ DILARANG for range jika index penting tapi di-ignore
for _, item := range items { // padahal butuh i
    items[i].Status = "done" // bug: i tidak terdefinisi di sini
}
```

---

## 24. Goroutine — Optimasi Logic untuk Case Tertentu

### 24.1 Kapan BOLEH dan TIDAK BOLEH pakai Goroutine

```
✅ BOLEH pakai goroutine:
   - Operasi I/O paralel yang independen (fetch data dari beberapa sumber sekaligus)
   - Background task yang tidak perlu ditunggu hasilnya (fire-and-forget dengan batas)
   - Fan-out: kirim ke banyak consumer secara paralel
   - Pipeline: tahap-tahap processing yang bisa overlap
   - Worker pool untuk memproses antrian job

❌ JANGAN pakai goroutine:
   - Operasi yang harus sequential (transaksi DB, step yang bergantung satu sama lain)
   - Hanya karena "mungkin lebih cepat" tanpa profiling
   - Jika jumlah goroutine tidak dibatasi (bisa OOM)
   - Untuk operasi yang sudah di-handle async oleh library (HTTP client, DB driver)
```

---

### 24.2 Pattern 1 — Fan-Out dengan errgroup (Paling Umum)

**Use case:** Perlu fetch data dari beberapa sumber sekaligus, semua hasilnya dibutuhkan.

```go
// Contoh: transfer usecase perlu validasi from_account DAN to_account secara paralel
// Daripada sequential (lambat), fetch keduanya bersamaan

import "golang.org/x/sync/errgroup"

func (u *transferUsecase) validateAccounts(
    ctx context.Context,
    fromID, toID string,
) (*domain.Account, *domain.Account, error) {

    var (
        fromAccount *domain.Account
        toAccount   *domain.Account
    )

    // errgroup otomatis cancel context jika salah satu goroutine error
    g, gCtx := errgroup.WithContext(ctx)

    g.Go(func() error {
        acc, err := u.accountRepo.FindByID(gCtx, fromID)
        if err != nil {
            return fmt.Errorf("fetch source account: %w", err)
        }
        fromAccount = acc
        return nil
    })

    g.Go(func() error {
        acc, err := u.accountRepo.FindByID(gCtx, toID)
        if err != nil {
            return fmt.Errorf("fetch destination account: %w", err)
        }
        toAccount = acc
        return nil
    })

    // Tunggu semua selesai — jika ada error, return error pertama
    if err := g.Wait(); err != nil {
        return nil, nil, err
    }

    return fromAccount, toAccount, nil
}
```

---

### 24.3 Pattern 2 — Fan-Out dengan Hasil Koleksi

**Use case:** Proses banyak item secara paralel, kumpulkan semua hasilnya.

```go
// Contoh: enrich list of transfers dengan data tambahan secara paralel

func enrichTransfers(
    ctx context.Context,
    transfers []domain.Transfer,
    enrichFn func(context.Context, domain.Transfer) (EnrichedTransfer, error),
) ([]EnrichedTransfer, error) {

    results := make([]EnrichedTransfer, len(transfers))
    g, gCtx := errgroup.WithContext(ctx)

    for i := 0; i < len(transfers); i++ {
        i := i // capture loop variable — wajib di Go < 1.22
        transfer := transfers[i]

        g.Go(func() error {
            enriched, err := enrichFn(gCtx, transfer)
            if err != nil {
                return fmt.Errorf("enrich transfer %s: %w", transfer.ID, err)
            }
            results[i] = enriched // aman: tiap goroutine tulis index berbeda
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }

    return results, nil
}
```

---

### 24.4 Pattern 3 — Worker Pool (Bounded Concurrency)

**Use case:** Proses banyak item tapi concurrency harus dibatasi agar tidak overload DB / downstream service.

```go
// ❌ JANGAN spawn goroutine tanpa batas
for i := 0; i < len(items); i++ {
    go process(items[i]) // jika len(items) = 10000 → 10000 goroutine sekaligus
}

// ✅ Worker pool — batasi jumlah goroutine yang berjalan bersamaan
func processWithWorkerPool(
    ctx context.Context,
    items []Item,
    workerCount int, // biasanya 5-20, sesuaikan dengan kapasitas downstream
    processFn func(context.Context, Item) error,
) error {
    g, gCtx := errgroup.WithContext(ctx)

    // Semaphore via buffered channel untuk limit concurrency
    sem := make(chan struct{}, workerCount)

    for i := 0; i < len(items); i++ {
        item := items[i]

        g.Go(func() error {
            // Acquire slot
            select {
            case sem <- struct{}{}:
            case <-gCtx.Done():
                return gCtx.Err()
            }
            defer func() { <-sem }() // Release slot

            return processFn(gCtx, item)
        })
    }

    return g.Wait()
}

// Penggunaan:
err := processWithWorkerPool(ctx, pendingTransfers, 10, func(ctx context.Context, t Item) error {
    return usecase.ProcessTransfer(ctx, t)
})
```

---

### 24.5 Pattern 4 — Pipeline

**Use case:** Setiap tahap processing bisa berjalan overlap. Tahap A produce ke channel, tahap B consume sambil tahap A masih jalan.

```go
// Contoh: read records dari DB → transform → publish ke kafka
// Tanpa pipeline: read semua dulu, transform semua, publish semua (memory besar)
// Dengan pipeline: read → transform → publish berjalan bersamaan

func runPipeline(ctx context.Context, ...) error {
    // Stage 1: read dari DB, kirim ke channel
    rawCh := make(chan domain.Transfer, 100) // buffer 100 agar producer tidak block terus
    g, gCtx := errgroup.WithContext(ctx)

    g.Go(func() error {
        defer close(rawCh) // wajib close agar stage berikutnya tahu selesai
        return fetchFromDB(gCtx, rawCh)
    })

    // Stage 2: transform, kirim ke channel berikutnya
    enrichedCh := make(chan EnrichedTransfer, 100)
    g.Go(func() error {
        defer close(enrichedCh)
        for raw := range rawCh {
            select {
            case <-gCtx.Done():
                return gCtx.Err()
            default:
            }
            enriched, err := transform(gCtx, raw)
            if err != nil {
                return fmt.Errorf("transform: %w", err)
            }
            enrichedCh <- enriched
        }
        return nil
    })

    // Stage 3: publish ke Kafka
    g.Go(func() error {
        for enriched := range enrichedCh {
            select {
            case <-gCtx.Done():
                return gCtx.Err()
            default:
            }
            if err := publish(gCtx, enriched); err != nil {
                return fmt.Errorf("publish: %w", err)
            }
        }
        return nil
    })

    return g.Wait()
}
```

---

### 24.6 Pattern 5 — Fire-and-Forget yang Aman

**Use case:** Kirim notifikasi / audit log secara async, tidak perlu tunggu hasilnya, tidak boleh ganggu flow utama.

```go
// ❌ DILARANG — goroutine liar tanpa batas dan tanpa recover
go sendNotification(transfer)

// ✅ Fire-and-forget yang aman: pakai bounded goroutine pool atau worker queue
// Opsi 1: kirim ke channel yang diproses worker pool yang sudah berjalan
func (u *transferUsecase) Execute(ctx context.Context, req Request) (*Result, error) {
    result, err := u.processTransfer(ctx, req)
    if err != nil {
        return nil, err
    }

    // Kirim ke notification queue — non-blocking, dengan timeout
    notifEvent := buildNotificationEvent(result)
    select {
    case u.notifQueue <- notifEvent: // berhasil masuk queue
    default:
        // Queue penuh — log warning, jangan gagalkan request utama
        slog.Warn("notification queue full, skipping",
            "transfer_id", result.ID)
    }

    return result, nil
}

// Worker yang consume notification queue (dijalankan di background saat startup)
func (w *NotificationWorker) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case event := <-w.queue:
            func() {
                defer func() {
                    if r := recover(); r != nil {
                        slog.Error("notification worker panic",
                            "error", r,
                            "event", event)
                    }
                }()
                if err := w.sender.Send(ctx, event); err != nil {
                    slog.Error("send notification failed",
                        "error", err,
                        "event_id", event.ID)
                }
            }()
        }
    }
}
```

---

### 24.7 Pattern 6 — Timeout per Goroutine

**Use case:** Setiap goroutine dalam fan-out punya timeout sendiri agar satu yang lambat tidak block semua.

```go
func fetchWithTimeout(
    ctx context.Context,
    id string,
    timeout time.Duration,
    fetchFn func(context.Context, string) (*Result, error),
) (*Result, error) {
    // Buat context dengan timeout khusus untuk goroutine ini
    tCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    type outcome struct {
        result *Result
        err    error
    }

    ch := make(chan outcome, 1)
    go func() {
        r, err := fetchFn(tCtx, id)
        ch <- outcome{r, err}
    }()

    select {
    case out := <-ch:
        return out.result, out.err
    case <-tCtx.Done():
        return nil, fmt.Errorf("fetchWithTimeout: timeout after %s: %w",
            timeout, tCtx.Err())
    }
}
```

---

### 24.8 Aturan Goroutine yang Wajib Diikuti

```
1. SETIAP goroutine wajib ada mekanisme stop (ctx.Done())
2. SETIAP goroutine wajib ada recover() untuk panic
3. JUMLAH goroutine yang spawn harus dibatasi (worker pool / semaphore)
4. JANGAN share variabel antar goroutine tanpa sinkronisasi (mutex / channel)
5. GUNAKAN errgroup jika goroutine harus dikumpulkan hasilnya
6. CHANNEL yang di-produce WAJIB di-close oleh producer (bukan consumer)
7. JANGAN close channel dua kali — panic
8. HINDARI goroutine leak: pastikan semua goroutine bisa exit
9. UKUR dulu dengan profiling sebelum optimasi dengan goroutine
10. PREFER errgroup over raw WaitGroup — lebih aman untuk error propagation
```

---

### 24.9 Anti-Pattern yang Dilarang

```go
// ❌ Spawn goroutine tanpa batas
for i := 0; i < len(items); i++ {
    go process(items[i]) // bisa spawn ribuan goroutine
}

// ❌ Goroutine tanpa ctx
go func() {
    for { doWork() } // tidak bisa di-stop
}()

// ❌ Goroutine tanpa recover
go func() {
    riskyOp() // panic → crash seluruh service
}()

// ❌ Share slice/map tanpa lock
results := make([]Result, 0)
var wg sync.WaitGroup
for i := 0; i < len(items); i++ {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        results = append(results, process(item)) // DATA RACE!
    }(items[i])
}

// ❌ Close channel dari consumer
go func() {
    close(ch) // consumer tidak boleh close, hanya producer
}()

// ❌ WaitGroup tanpa error handling
var wg sync.WaitGroup
for i := 0; i < len(items); i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := process(); err != nil {
            // error hilang ke mana?
        }
    }()
}
wg.Wait() // tidak tahu ada error atau tidak
// ✅ Ganti dengan errgroup
```

---


## 25. Testing — Implementasi

### 25.1 Struktur Test per Layer

```go
// usecase/{action}_{entity}_usecase_test.go

// SEMUA outbound port di-mock — tidak boleh ada DB/network call
type mockAccountRepo struct {
    findByIDForUpdateFn func(ctx context.Context, tx usecase.Tx, id string) (*domain.Account, error)
    updateBalanceFn     func(ctx context.Context, tx usecase.Tx, id string, delta decimal.Decimal, version int) error
}

func (m *mockAccountRepo) FindByIDForUpdate(ctx context.Context, tx usecase.Tx, id string) (*domain.Account, error) {
    return m.findByIDForUpdateFn(ctx, tx, id)
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, tx usecase.Tx, id string, delta decimal.Decimal, version int) error {
    return m.updateBalanceFn(ctx, tx, id, delta, version)
}

// mockTxManager — simulasi transaksi tanpa DB
type mockTxManager struct{}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(usecase.Tx) error) error {
    return fn(nil) // tx nil, repo mock tidak perlu tx nyata
}
```

### 25.2 Table-Driven Test (Wajib untuk Usecase)

```go
func Test_ExecuteTransferUsecase_Execute(t *testing.T) {
    tests := []struct {
        name          string
        req           dto.ExecuteTransferRequest
        setupMock     func(*mockAccountRepo, *mockOutboxRepo)
        wantErr       error
        wantErrCode   string
    }{
        {
            name: "success",
            req: dto.ExecuteTransferRequest{
                FromAccountID:  "acc-1",
                ToAccountID:    "acc-2",
                Amount:         decimal.NewFromInt(100000),
                IdempotencyKey: "idem-key-1",
            },
            setupMock: func(ar *mockAccountRepo, or *mockOutboxRepo) {
                ar.findByIDForUpdateFn = func(_ context.Context, _ usecase.Tx, id string) (*domain.Account, error) {
                    if id == "acc-1" {
                        return &domain.Account{ID: "acc-1", Balance: decimal.NewFromInt(500000), Status: domain.AccountActive}, nil
                    }
                    return &domain.Account{ID: "acc-2", Balance: decimal.NewFromInt(100000), Status: domain.AccountActive}, nil
                }
                ar.updateBalanceFn = func(_ context.Context, _ usecase.Tx, _ string, _ decimal.Decimal, _ int) error {
                    return nil
                }
                or.saveFn = func(_ context.Context, _ usecase.Tx, _ domain.OutboxEvent) error {
                    return nil
                }
            },
            wantErr: nil,
        },
        {
            name: "insufficient balance",
            req: dto.ExecuteTransferRequest{
                FromAccountID:  "acc-1",
                Amount:         decimal.NewFromInt(999999999),
                IdempotencyKey: "idem-key-2",
            },
            setupMock: func(ar *mockAccountRepo, or *mockOutboxRepo) {
                ar.findByIDForUpdateFn = func(_ context.Context, _ usecase.Tx, _ string) (*domain.Account, error) {
                    return &domain.Account{Balance: decimal.NewFromInt(100)}, nil
                }
            },
            wantErr:     domain.ErrInsufficientBalance,
            wantErrCode: "INSUFFICIENT_BALANCE",
        },
        {
            name: "source account not found",
            req: dto.ExecuteTransferRequest{
                FromAccountID:  "acc-not-exist",
                IdempotencyKey: "idem-key-3",
            },
            setupMock: func(ar *mockAccountRepo, or *mockOutboxRepo) {
                ar.findByIDForUpdateFn = func(_ context.Context, _ usecase.Tx, _ string) (*domain.Account, error) {
                    return nil, domain.ErrAccountNotFound
                }
            },
            wantErr: domain.ErrAccountNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mockAccountRepo{}
            mockOutbox := &mockOutboxRepo{}
            tt.setupMock(mockRepo, mockOutbox)

            uc := NewExecuteTransferUsecase(
                &mockTxManager{},
                mockRepo,
                mockOutbox,
            )

            _, err := uc.Execute(context.Background(), tt.req)

            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("want error %v, got %v", tt.wantErr, err)
                }
            } else if err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### 25.3 Integration Test Repository (dengan testcontainers)

```go
// adapter/repository/account_repository_test.go

func TestAccountRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Spin up PostgreSQL container
    ctx := context.Background()
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections"),
        ),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer container.Terminate(ctx)

    connStr, _ := container.ConnectionString(ctx, "sslmode=disable")
    db, _ := sql.Open("postgres", connStr)

    // Run migration
    runMigrations(db)

    repo := NewAccountRepository(db)

    t.Run("FindByID existing account", func(t *testing.T) {
        // seed data
        insertTestAccount(db, "acc-test-1")

        acc, err := repo.FindByID(ctx, "acc-test-1")
        if err != nil {
            t.Fatal(err)
        }
        if acc.ID != "acc-test-1" {
            t.Errorf("want acc-test-1, got %s", acc.ID)
        }
    })

    t.Run("FindByID not found", func(t *testing.T) {
        _, err := repo.FindByID(ctx, "not-exist")
        if !errors.Is(err, domain.ErrAccountNotFound) {
            t.Errorf("want ErrAccountNotFound, got %v", err)
        }
    })
}
```

### 25.4 Handler Test (HTTP)

```go
// adapter/handler/transfer_handler_test.go

func Test_TransferHandler_Transfer(t *testing.T) {
    tests := []struct {
        name           string
        body           string
        mockUsecase    func() port.ExecuteTransferUsecase
        wantStatus     int
        wantCode       string
    }{
        {
            name: "success",
            body: `{"from_account_id":"acc-1","to_account_id":"acc-2","amount":100000,"idempotency_key":"key-1"}`,
            mockUsecase: func() port.ExecuteTransferUsecase {
                return &mockExecuteTransferUsecase{
                    executeFn: func(_ context.Context, _ dto.ExecuteTransferRequest) (*dto.ExecuteTransferResult, error) {
                        return &dto.ExecuteTransferResult{TransferID: "tx-1"}, nil
                    },
                }
            },
            wantStatus: 201,
            wantCode:   "TRANSFER_CREATED",
        },
        {
            name: "validation error — missing idempotency key",
            body: `{"from_account_id":"acc-1","to_account_id":"acc-2","amount":100000}`,
            mockUsecase: func() port.ExecuteTransferUsecase {
                return &mockExecuteTransferUsecase{}
            },
            wantStatus: 400,
            wantCode:   "VALIDATION_ERROR",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            app := fiber.New()
            handler := NewTransferHandler(tt.mockUsecase(), response.NewApiResponseFactory())
            app.Post("/api/v1/transfers", handler.Transfer)

            req := httptest.NewRequest("POST", "/api/v1/transfers",
                strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")

            resp, err := app.Test(req)
            if err != nil {
                t.Fatal(err)
            }

            if resp.StatusCode != tt.wantStatus {
                t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
            }

            var body response.ApiResponse[any]
            json.NewDecoder(resp.Body).Decode(&body)
            if body.Code != tt.wantCode {
                t.Errorf("want code %s, got %s", tt.wantCode, body.Code)
            }
        })
    }
}
```

### 25.5 Test Rules

```
1. Test file selalu di package yang sama dengan kode yang ditest (_test suffix)
2. Test name pattern: Test_{FunctionName}_{Scenario}
3. Jangan hardcode port, credential, atau URL di test file
4. Test harus bisa dijalankan paralel (t.Parallel() jika tidak ada shared state)
5. Coverage target: domain + usecase minimum 80%
6. Jika usecase sulit di-unit test → itu sinyal ada dependency langsung ke infra
7. Integration test wajib pakai build tag: //go:build integration
```

---

## 26. Mapping — Implementasi

Referensi boundary-nya di `ARCHITECTURE.md` Section 6.

### 26.1 Handler Mapper

```go
// adapter/handler/dto/request.go — HTTP binding struct
type TransferRequestDTO struct {
    FromAccountID  string  `json:"from_account_id"  validate:"required,uuid4"`
    ToAccountID    string  `json:"to_account_id"    validate:"required,uuid4,nefield=FromAccountID"`
    Amount         float64 `json:"amount"           validate:"required,gt=0"`
    Currency       string  `json:"currency"         validate:"required,len=3"`
    IdempotencyKey string  `json:"idempotency_key"  validate:"required,uuid4"`
    Note           string  `json:"note"             validate:"omitempty,max=255"`
}

// adapter/handler/dto/response.go — HTTP response struct
type TransferResponseDTO struct {
    TransferID string    `json:"transfer_id"`
    Status     string    `json:"status"`
    CreatedAt  time.Time `json:"created_at"`
}

// adapter/handler/mapper.go — mapping di adapter, bukan di usecase/domain
func toExecuteTransferRequest(dto TransferRequestDTO, requestID, traceID string) usecasedto.ExecuteTransferRequest {
    return usecasedto.ExecuteTransferRequest{
        FromAccountID:  dto.FromAccountID,
        ToAccountID:    dto.ToAccountID,
        Amount:         decimal.NewFromFloat(dto.Amount),
        Currency:       dto.Currency,
        IdempotencyKey: dto.IdempotencyKey,
        RequestID:      requestID,
        TraceID:        traceID,
    }
}

func toTransferResponseDTO(result *usecasedto.ExecuteTransferResult) TransferResponseDTO {
    return TransferResponseDTO{
        TransferID: result.TransferID,
        Status:     result.Status,
        CreatedAt:  result.CreatedAt,
    }
}
```

### 26.2 gRPC Mapper

```go
// adapter/grpc/server/mapper.go
func protoToExecuteTransferRequest(req *pb.ExecuteTransferRequest, traceID string) dto.ExecuteTransferRequest {
    return dto.ExecuteTransferRequest{
        FromAccountID:  req.FromAccountId,
        ToAccountID:    req.ToAccountId,
        Amount:         decimal.NewFromString(req.Amount), // proto pakai string untuk decimal
        Currency:       req.Currency,
        IdempotencyKey: req.IdempotencyKey,
        TraceID:        traceID,
    }
}

func resultToProtoResponse(result *dto.ExecuteTransferResult) *pb.ExecuteTransferResponse {
    return &pb.ExecuteTransferResponse{
        TransferId: result.TransferID,
        Status:     result.Status,
        CreatedAt:  timestamppb.New(result.CreatedAt),
    }
}
```

### 26.3 Consumer Mapper

```go
// adapter/consumer/mapper.go
func eventPayloadToRequest(envelope domain.EventEnvelope) (dto.ExecuteTransferRequest, error) {
    var payload domain.TransferRequestedPayload
    if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
        return dto.ExecuteTransferRequest{}, fmt.Errorf("unmarshal payload: %w", err)
    }
    return dto.ExecuteTransferRequest{
        FromAccountID:  payload.FromAccountID,
        ToAccountID:    payload.ToAccountID,
        Amount:         payload.Amount,
        IdempotencyKey: payload.IdempotencyKey, // event_id bisa dipakai sebagai idempotency key
        TraceID:        envelope.TraceID,
    }, nil
}
```

### 26.4 Repository Mapper

```go
// adapter/repository/mapper.go
// DB row → domain entity (hanya di repository layer)

type transferRow struct {
    ID             string          `db:"id"`
    FromAccountID  string          `db:"from_account_id"`
    ToAccountID    string          `db:"to_account_id"`
    Amount         decimal.Decimal `db:"amount"`
    Status         string          `db:"status"`
    IdempotencyKey string          `db:"idempotency_key"`
    CreatedAt      time.Time       `db:"created_at"`
    Version        int             `db:"version"`
}

func rowToDomain(row transferRow) *domain.Transfer {
    return &domain.Transfer{
        ID:             row.ID,
        FromAccountID:  row.FromAccountID,
        ToAccountID:    row.ToAccountID,
        Amount:         row.Amount,
        Status:         domain.TransferStatus(row.Status),
        IdempotencyKey: row.IdempotencyKey,
        CreatedAt:      row.CreatedAt,
        Version:        row.Version,
    }
}

func domainToRow(t *domain.Transfer) transferRow {
    return transferRow{
        ID:             t.ID,
        FromAccountID:  t.FromAccountID,
        ToAccountID:    t.ToAccountID,
        Amount:         t.Amount,
        Status:         string(t.Status),
        IdempotencyKey: t.IdempotencyKey,
        CreatedAt:      t.CreatedAt,
        Version:        t.Version,
    }
}
```

---

> **Penutup untuk AI Agent:**
> Dokumen ini adalah **kontrak teknis** antara engineer dan AI agent.
> Generate kode yang melanggar aturan di sini berarti generate kode yang salah,
> terlepas dari apakah kode tersebut compile atau berjalan.
> Jika ada edge case yang tidak tercakup di sini — **tanya dulu**.
> Selalu rujuk `ARCHITECTURE.md` untuk boundary dan flow, file ini untuk implementasi.