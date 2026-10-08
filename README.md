# Community Waste Collection API

RESTful API untuk **Sistem Pengumpulan Sampah Komunitas**.

---

## Arsitektur & Struktur Proyek

```
community-waste-api/
├── cmd/
│   ├── api/
│   │   └── main.go                 # Entrypoint server Fiber & Graceful Shutdown
│   └── seeder/
│       └── main.go                 # Seeder data
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
│   ├── database/                   # Pool pgx & runner migrasi otomatis
│   ├── response/                   # Standardized JSON response envelope
│   ├── uploader/                   # File validation (MIME, size) & penyimpanan direktori lokal
│   ├── repository/                 # Store pattern & transaction runner (ExecTx)
│   ├── service/                    # Layer logika bisnis
│   └── handler/                    # Presentation layer (HTTP handlers, router, middlewares)
├── tests/
│   └── unit/                       # Unit tests
├── uploads/
│   └── payments/                   # Direktori penyimpanan bukti pembayaran
├── postman/
│   ├── Community_Waste_API.insomnia_collection.json
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

## Quick Start (Docker Compose)

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

## Setup Lokal (Manual)

### Requirements
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

## Database Migration & Seeding

### Seeding Data

```bash
go run cmd/seeder/main.go
# atau menggunakan Makefile:
make seed
```
---

## Environment Variables

| Variable | Default Value
|---|---|---|
| `APP_ENV` | `development`
| `APP_PORT` | `8080`
| `BASE_URL` | `http://localhost:8080`
| `DB_HOST` | `localhost` (docker: `postgres`)
| `DB_PORT` | `5432`
| `DB_USER` | `postgres`
| `DB_PASSWORD` | `postgres`
| `DB_NAME` | `waste_collection_db`
| `DB_SSLMODE` | `disable`
| `UPLOAD_DIR` | `./uploads/payments`
| `MAX_UPLOAD_SIZE_MB` | `5`

---

## Dokumentasi API & Postman

File koleksi API telah disertakan untuk **Postman** dan **Insomnia**:
- **Postman:**
  1. `postman/Community_Waste_API.postman_collection.json`
  2. `postman/Community_Waste_API.postman_environment.json`
- **Insomnia:**
  1. `postman/Community_Waste_API.insomnia_collection.json` (Import via *Create > Import from File*)

### Cara Menggunakan Postman:
1. Buka aplikasi **Postman**.
2. Klik tombol **Import** di kiri atas, lalu pilih kedua file di atas.
3. Pilih Environment **"Community Waste API - Local"**.
4. Request `POST /api/households`, `POST /api/pickups`, dan `POST /api/payments` sudah dilengkapi skrip pengujian otomatis yang akan menyimpan `household_id`, `pickup_id`, dan `payment_id` langsung ke variabel environment.

### Ringkasan Endpoint

#### 1. Household Endpoints
- `POST /api/households` - Tambah data household
- `GET /api/households?limit=10` - List households dengan pagination (cursor-based)
- `GET /api/households/:id` - Detail data household
- `DELETE /api/households/:id` - Hapus household

#### 2. Waste Pickup Endpoints
- `POST /api/pickups` - Buat permintaan pengambilan sampah
- `GET /api/pickups?status=pending&household_id=:uuid&limit=10` - List Pickup dengan filter status
- `PUT /api/pickups/:id/schedule` - Jadwalkan pickup
- `PUT /api/pickups/:id/complete` - Tandai selesai & trigger pembuatan payment otomatis
- `PUT /api/pickups/:id/cancel` - Batalkan pickup

#### 3. Payment Endpoints
- `POST /api/payments` - Buat payment manual terkait household & waste
- `GET /api/payments?status=paid&start_date=2026-10-01&end_date=2026-10-31` - List Payment Filter & rentang tanggal
- `PUT /api/payments/:id/confirm` - Konfirmasi payment dengan upload bukti bayar (`multipart/form-data`, field: `proof_file`)

#### 4. Reporting Endpoints
- `GET /api/reports/waste-summary` - Rekap agregat sampah berdasarkan type dan status
- `GET /api/reports/payment-summary` - Rekap pembayaran per status dan total pendapatan

#### 5. Health Check
- `GET /health` - Server health check
