# Community Waste Collection API

RESTful API backend untuk **Sistem Pengumpulan Sampah Komunitas**, dibangun menggunakan **Go**, **Fiber v2**, **PostgreSQL**, dan **sqlc**.

Proyek ini dibuat untuk memenuhi **Take Home Test Backend Developer (Golang)** di **PT. Green Energi Utama**.

---

## 📋 Daftar Isi
1. [Tech Stack](#-tech-stack)
2. [Arsitektur & Struktur Proyek](#-arsitektur--struktur-proyek)
3. [Aturan Bisnis (Business Rules)](#-aturan-bisnis-business-rules)
4. [Quick Start (Docker Compose)](#-quick-start-docker-compose)
5. [Setup Lokal (Manual)](#-setup-lokal-manual)
6. [Database Migration & Seeding](#-database-migration--seeding)
7. [Unit Testing](#-unit-testing)
8. [Environment Variables](#-environment-variables)
9. [Dokumentasi API & Postman](#-dokumentasi-api--postman)

---

## 🛠 Tech Stack

- **Bahasa Pemrograman:** Go (Golang) 1.24+
- **HTTP Framework:** [GoFiber v2](https://github.com/gofiber/fiber) (Fast, Express-like HTTP engine)
- **Database:** PostgreSQL 16 (Native ENUMs, UUID, Foreign Keys)
- **Database Driver & Generator:** [sqlc](https://sqlc.dev) + [pgx/v5](https://github.com/jackc/pgx/v5) (Compile-time type-safe SQL, zero reflection)
- **Database Migrations:** [golang-migrate](https://github.com/golang-migrate/migrate)
- **Containerization:** Docker & Docker Compose
- **Testing:** [testify/mock](https://github.com/stretchr/testify)

---

## 🏛 Arsitektur & Struktur Proyek

Proyek ini mengadopsi **Clean / Layered Architecture** dengan pemisahan tanggung jawab yang tegas:

```
community-waste-api/
├── cmd/
│   ├── api/
│   │   └── main.go                 # Entrypoint server Fiber & Graceful Shutdown
│   └── seeder/
│       └── main.go                 # Seeder data realistis untuk pengujian reviewer
├── db/
│   ├── migrations/                 # Migrasi SQL DDL (Up & Down)
│   │   ├── 000001_create_households.up.sql
│   │   ├── 000001_create_households.down.sql
│   │   ├── 000002_create_waste_pickups.up.sql
│   │   ├── 000002_create_waste_pickups.down.sql
│   │   ├── 000003_create_payments.up.sql
│   │   └── 000003_create_payments.down.sql
│   ├── queries/                    # SQL Queries untuk sqlc
│   │   ├── households.sql
│   │   ├── pickups.sql
│   │   ├── payments.sql
│   │   └── reports.sql
│   └── sqlc/                       # Type-safe Go code hasil generate sqlc
│       ├── db.go
│       ├── models.go
│       ├── querier.go
│       ├── households.sql.go
│       ├── pickups.sql.go
│       ├── payments.sql.go
│       └── reports.sql.go
├── internal/
│   ├── config/                     # Loader environment variables
│   ├── domain/                     # Model domain, DTOs, dan custom errors
│   ├── repository/                 # Store pattern & transaction runner (ExecTx)
│   ├── service/                    # Layer logika bisnis & penegakan 5 aturan bisnis
│   └── handler/                    # Presentation layer (HTTP handlers, router, middlewares)
├── pkg/
│   ├── database/                   # Pool pgx & runner migrasi otomatis
│   ├── response/                   # Standardized JSON response envelope
│   └── uploader/                   # File validation (MIME, size) & penyimpanan disk lokal
├── tests/
│   └── unit/                       # Unit tests aturan bisnis dengan mock Store
├── uploads/
│   └── payments/                   # Volume penyimpanan bukti pembayaran
├── postman/
│   ├── Community_Waste_API.postman_collection.json
│   └── Community_Waste_API.postman_environment.json
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── sqlc.yaml
├── .env.example
└── README.md
```

### Penjelasan Layer:
1. **Handler Layer (`internal/handler/http`)**: Menerima request HTTP, memvalidasi input dasar, meneruskan payload ke service, dan mengembalikan HTTP status code & JSON response yang terstandarisasi.
2. **Service Layer (`internal/service`)**: Pusat aturan bisnis. Bertanggung jawab memvalidasi alur operasional (BR-01 s/d BR-05) dan membungkus operasi multi-tabel ke dalam transaksi atomik database (`ExecTx`).
3. **Repository Layer (`internal/repository` & `db/sqlc`)**: Menggunakan query type-safe yang digenerate oleh `sqlc` dengan `emit_interface: true` untuk abstraksi `Querier` dan `Store`. Mempermudah mocking tanpa dependensi ke DB riil saat unit testing.

---

## ⚖ Aturan Bisnis (Business Rules)

| ID | Aturan Bisnis | Implementasi & Penegakan | Status HTTP & Penanganan |
|---|---|---|---|
| **BR-01** | Household **tidak dapat membuat pickup baru** jika memiliki payment berstatus `pending`. | Diperiksa di `PickupService.Create` via `CountPendingPaymentsByHouseholdID`. | `422 Unprocessable Entity` jika ada tagihan pending. |
| **BR-02** | Pickup **hanya dapat dijadwalkan** jika status saat ini bernilai `pending`. | Diperiksa di `PickupService.Schedule`. Pickup yang sudah scheduled/completed/canceled ditolak. | `422 Unprocessable Entity` / `400 Bad Request`. |
| **BR-03** | Pickup tipe `electronic` **tidak dapat dijadwalkan** kecuali `safety_check == true`. | Diperiksa di `PickupService.Schedule`. Validasi khusus tipe electronic. | `422 Unprocessable Entity` jika safety check bernilai false. |
| **BR-04** | Saat pickup ditandai `completed`, otomatis generate tagihan `Payment`: <br>• `organic`, `plastic`, `paper` = Rp50.000 <br>• `electronic` = Rp100.000 | Dieksekusi secara **atomik dalam 1 Transaksi Database (`ExecTx`)** di `PickupService.Complete`. | `200 OK` (Record payment status `pending` langsung terbuat). |
| **BR-05** | Konfirmasi payment (`/confirm`) **wajib upload file bukti bayar**. File disimpan di disk lokal & URL/path disimpan di record. | Diproses di `PaymentService.Confirm` via `LocalUploader`. Validasi format file (`jpg, jpeg, png, pdf`) dan ukuran max. | `200 OK` (Status berubah menjadi `paid`, `payment_date` terisi, path disimpan). |

---

## 🚀 Quick Start (Docker Compose)

Cukup **satu perintah** untuk menjalankan seluruh sistem (Aplikasi Go + PostgreSQL):

```bash
docker compose up --build
```

Setelah kontainer berjalan:
- API siap diakses pada: `http://localhost:8080`
- Cek status server: `http://localhost:8080/health`
- Migrasi database otomatis dijalankan saat aplikasi startup.

Untuk menghentikan kontainer:
```bash
docker compose down
```

---

## 💻 Setup Lokal (Manual)

### Prasyarat
- Go 1.22+
- PostgreSQL 14+ berjalan lokal
- Port `8080` dan `5432` tersedia

### Langkah Menjalankan:
1. Clone repositori:
   ```bash
   git clone <repo-url>
   cd community-waste-api
   ```
2. Salin environment variable:
   ```bash
   cp .env.example .env
   ```
   *Sesuaikan kredensial database di file `.env` jika diperlukan.*
3. Jalankan aplikasi (migrasi berjalan otomatis saat startup):
   ```bash
   go run cmd/api/main.go
   # atau menggunakan Makefile:
   make run
   ```

---

## 🗄 Database Migration & Seeding

### Seeding Data Pengujian
Untuk mempermudah review dan pengujian seluruh skenario (termasuk skenario lolos dan skenario ditolak aturan bisnis), jalankan script seeder:

```bash
go run cmd/seeder/main.go
# atau menggunakan Makefile:
make seed
```

**Data yang digenerate oleh seeder:**
- 3 Rumah Tangga (Budi Santoso, Siti Rahmawati, Ahmad Fauzi)
- 5 Waste Pickups:
  - Budi Santoso: Organic (Scheduled), Plastic (Completed)
  - Siti Rahmawati: Electronic (Pending, `safety_check: true`), Paper (Pending)
  - Ahmad Fauzi: Electronic (Pending, `safety_check: false` ➡️ *siap menguji kegagalan BR-03*)
- 2 Payments:
  - Budi Santoso: Plastic pickup (Status `paid`, Rp50.000)
  - Ahmad Fauzi: Electronic pickup (Status `pending`, Rp100.000 ➡️ *siap menguji kegagalan BR-01 saat Ahmad membuat pickup baru*)

---

## 🧪 Unit Testing

Unit test dirancang untuk menguji seluruh logika aturan bisnis (**BR-01** s/d **BR-05**) secara cepat dan terisolasi menggunakan **Mock Store**:

```bash
go test -v ./tests/unit/...
# atau menggunakan Makefile:
make test
```

Contoh output:
```
=== RUN   TestPickupService_BR01_PendingPaymentCheck
    --- PASS: TestPickupService_BR01_PendingPaymentCheck/BR-01_Fail (0.00s)
    --- PASS: TestPickupService_BR01_PendingPaymentCheck/BR-01_Success (0.00s)
=== RUN   TestPickupService_BR02_OnlyPendingCanBeScheduled
    --- PASS: TestPickupService_BR02_OnlyPendingCanBeScheduled (0.00s)
=== RUN   TestPickupService_BR03_ElectronicSafetyCheck
    --- PASS: TestPickupService_BR03_ElectronicSafetyCheck/BR-03_Fail (0.00s)
    --- PASS: TestPickupService_BR03_ElectronicSafetyCheck/BR-03_Success (0.00s)
=== RUN   TestPickupService_BR04_AutoPaymentGenerationOnComplete
    --- PASS: TestPickupService_BR04_AutoPaymentGenerationOnComplete/Organic_50k (0.00s)
    --- PASS: TestPickupService_BR04_AutoPaymentGenerationOnComplete/Electronic_100k (0.00s)
=== RUN   TestPaymentService_BR05_PaymentConfirmation
    --- PASS: TestPaymentService_BR05_PaymentConfirmation/BR-05_Fail (0.00s)
    --- PASS: TestPaymentService_BR05_PaymentConfirmation/BR-05_Success (0.00s)
PASS
```

---

## ⚙ Environment Variables

| Variable | Default Value | Deskripsi |
|---|---|---|
| `APP_ENV` | `development` | Environment mode (`development` / `production`) |
| `APP_PORT` | `8080` | Port listen server HTTP |
| `BASE_URL` | `http://localhost:8080` | Base URL server untuk file upload |
| `DB_HOST` | `localhost` (docker: `postgres`) | Host server PostgreSQL |
| `DB_PORT` | `5432` | Port server PostgreSQL |
| `DB_USER` | `postgres` | Username PostgreSQL |
| `DB_PASSWORD` | `postgres` | Password PostgreSQL |
| `DB_NAME` | `waste_collection_db` | Nama database |
| `DB_SSLMODE` | `disable` | SSL mode koneksi database |
| `UPLOAD_DIR` | `./uploads/payments` | Direktori penyimpanan bukti bayar |
| `MAX_UPLOAD_SIZE_MB` | `5` | Batas maksimum ukuran upload (MB) |

---

## 📬 Dokumentasi API & Postman

File koleksi Postman telah disertakan pada direktori `postman/`:
1. `postman/Community_Waste_API.postman_collection.json`
2. `postman/Community_Waste_API.postman_environment.json`

### Cara Menggunakan Postman:
1. Buka aplikasi **Postman**.
2. Klik tombol **Import** di kiri atas, lalu pilih kedua file di atas.
3. Pilih Environment **"Community Waste API - Local"**.
4. Request `POST /api/households`, `POST /api/pickups`, dan `POST /api/payments` sudah dilengkapi skrip pengujian otomatis yang akan menyimpan `household_id`, `pickup_id`, dan `payment_id` langsung ke variabel environment.

### Ringkasan Endpoint

#### 1. Household Endpoints
- `POST /api/households` - Tambah data rumah tangga
- `GET /api/households?page=1&limit=10` - Daftar rumah tangga dengan pagination
- `GET /api/households/:id` - Detail data rumah tangga
- `DELETE /api/households/:id` - Hapus rumah tangga

#### 2. Waste Pickup Endpoints
- `POST /api/pickups` - Buat permintaan pengambilan sampah (type wajib)
- `GET /api/pickups?status=pending&household_id=:uuid&page=1&limit=10` - Filter & daftar pickup
- `GET /api/pickups/:id` - Detail pickup
- `PUT /api/pickups/:id/schedule` - Jadwalkan pickup (`pickup_date` wajib)
- `PUT /api/pickups/:id/complete` - Tandai selesai & trigger pembuatan payment otomatis
- `PUT /api/pickups/:id/cancel` - Batalkan pickup

#### 3. Payment Endpoints
- `POST /api/payments` - Buat payment manual terkait household & waste
- `GET /api/payments?status=paid&start_date=2026-10-01&end_date=2026-10-31` - Filter & rentang tanggal
- `GET /api/payments/:id` - Detail payment
- `PUT /api/payments/:id/confirm` - Konfirmasi payment dengan upload bukti bayar (`multipart/form-data`, field: `proof_file`)

#### 4. Reporting Endpoints
- `GET /api/reports/waste-summary` - Rekap agregat sampah berdasarkan type dan status
- `GET /api/reports/payment-summary` - Rekap pembayaran per status dan total pendapatan

#### 5. Static Files & Health Check
- `GET /health` - Server health check
- `GET /uploads/payments/:filename` - Akses file bukti transfer yang telah di-upload
