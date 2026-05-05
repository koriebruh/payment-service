# Payment Service - Codebase Overview

Welcome to the `payment-service` repository! This document provides the latest overview of the service, its architecture, and how to navigate the codebase.

## 📖 Apa itu Payment Service?
`payment-service` adalah microservice yang bertanggung jawab untuk memproses seluruh transaksi pembayaran dari sistem. Service ini terintegrasi dengan **Midtrans** sebagai payment gateway utama, dan mempublikasikan status pembayaran (misalnya: ketika pembayaran berhasil/settlement) ke layanan lain melalui **Kafka**.

## 🏗️ Arsitektur Sistem
Service ini dibangun di atas bahasa pemrogaman **Go (Golang)** menggunakan **Hexagonal Architecture (Ports & Adapters)**. Aturan boundary pada arsitektur ini **sangat ketat**:

1. **Domain Layer (`internal/domain`)**: Jantung aplikasi. Tidak memiliki dependency ke framework atau library eksternal (zero external imports). Berisi Entity, Value Objects, Sentinel Errors, dan Business Rules.
2. **Usecase Layer (`internal/usecase`)**: Berisi Business Logic (Orkestrasi alur aplikasi). Layer ini mendefinisikan contract interfaces (Ports) yang dibutuhkan (seperti Repository, Gateway, Message Publisher), serta mengontrol Database Transaction Boundary.
3. **Adapter Layer (`internal/adapter`)**: Implementasi teknikal dari Ports (GORM untuk DB, Go Fiber untuk HTTP Handler, Kafka untuk messaging, dan HTTP Client SDK untuk Midtrans). Semua framework khusus berada di sini.

Untuk memahami standar arsitektur lebih lanjut, wajib membaca:
- `docs/ARCHITECTURE.md`
- `docs/CODING_GO_MICROSERVICE_STANDAR.md`

## 🛠️ Tech Stack Utama
- **Routing & HTTP**: Go Fiber
- **Database**: PostgreSQL
- **ORM**: GORM
- **Database Migrations**: golang-migrate (`go-migrate`)
- **Message Broker**: Apache Kafka (Producer)
- **Validation**: go-validator

## 📂 Navigasi Direktori
- `cmd/main.go` — Entrypoint aplikasi, tempat dependency injection dilakukan.
- `internal/domain/` — Definisi entity inti seperti `Transaction`, `WebhookLog`, `Refund`, dll.
- `internal/usecase/` — Proses bisnis seperti inisiasi *charge* ke Midtrans dan *handle webhook callback*.
- `internal/adapter/` — Integrasi eksternal (Fiber HTTP Handlers, GORM Repositories, Kafka Publishers, Midtrans Client).
- `config/` — Manajemen konfigurasi dan environment.
- `pkg/` — Komponen library reusable (mis. Response formatter, Logger, Validator).
- `migrations/` — Berisi script migrasi `.sql` database.

## 🚀 Fitur yang Tersedia (Dalam Pengembangan)
- **POST /api/v1/payments/charge**: Endpoint untuk membuat tagihan baru dan mendapatkan URL/Token pembayaran dari Midtrans.
- **POST /api/v1/payments/webhook/midtrans**: Endpoint *callback* yang akan dipanggil oleh Midtrans secara asinkron untuk memperbarui status transaksi. Jika transaksi berhasil, sistem akan men-trigger event Kafka ke consumer (misal: Notification/Order Service).

## 📄 File Tambahan Penting
- `docs/PROGRESS.MD` — Lacak status pengerjaan komponen service.
- `docs/DO/PAYMENT-SERVICE.MD` — Skema basis data dan flow pembayaran detail.
