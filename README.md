# EBITDA Max APN — Refactor (Go + Next.js)

Refactor aplikasi EBITDA Max APN dari Laravel/Inertia (`../ebitdamax-apn`)
menjadi dua repo terpisah:

| Repo | Isi |
|---|---|
| `ebitda-refactor` (repo ini) | REST API Go/Gin + worker/scheduler |
| `../ebitda-refactor-web` | Frontend Next.js 16 (App Router) |

Sumber perilaku adalah aplikasi Laravel lama. Scope aktif adalah paritas fitur
**Manager KDKMP** (`roles.domain = kdkmp` dan `roles.slug = manager`); peran
`manager-wilayah` dan `superadmin` hanya sebagai pendukung pengelolaan akun,
role, task, dan data. Detail scope, sprint, dan status ada di `BACKLOG.md` dan
`SPRINT.md`.

## Stack

- **Backend:** Go 1.26, Gin, GORM, PostgreSQL 17, Redis 7, MinIO, JWT
  (access + refresh rotating), TOTP 2FA, WebAuthn passkey, Lark SSO, Swagger,
  cron sarpras (advisory lock).
- **Frontend:** Next.js 16, React 19, TanStack Query, shadcn/Base UI,
  Tailwind CSS 4, Recharts.
- **Database:** migrasi goose di `migrations/` (baseline parity
  `00001_initial_schema.sql` + `00002_customer_analyses.sql` +
  `00003_lark_sso.sql`).

## Prasyarat

- Go 1.26+
- Node.js 20+ dan npm
- Docker & Docker Compose
- PostgreSQL client (`psql`) untuk menjalankan migrasi
- Git

## 1. Menjalankan Backend

```sh
# 1. Siapkan environment
cp .env.example .env

# 2. Jalankan dependency (Postgres 5433, Redis 6380, MinIO 9000/9001)
docker compose up -d
docker compose ps          # tunggu semua service "healthy"

# 3. Terapkan migrasi (urutan wajib)
for f in migrations/0000*.sql; do
  awk 'index($0, "-- +goose Down") { exit } { print }' "$f" | \
    PGPASSWORD=ebitdamax_dev psql -h 127.0.0.1 -p 5433 -U ebitdamax -d ebitdamax_apn \
      -q -v ON_ERROR_STOP=1
done

# 4. Seed role + akun dev (idempotent, aman diulang)
go run ./cmd/seed

# 5. Jalankan API (port 4000)
go run .
```

Verifikasi:

```sh
curl http://localhost:4000/healthz   # {"status":"ok","database":"up","redis":"up","minio":"up"}
```

- Swagger UI: <http://localhost:4000/swagger/index.html>
- MinIO Console: <http://localhost:9001> (`minioadmin` / `minioadmin_dev`)

Akun dev hasil seed:

| Email | Password | Role |
|---|---|---|
| `superadmin@agrinas.test` | `password123` | Superadmin |
| `manager@agrinas.test` | `password123` | Manager KDKMP |

### Data dev opsional

```sh
go run ./cmd/import-legacy-kdkmp-demo --apply     # 1 fixture demo KDKMP (butuh LEGACY_DB_*)
go run ./cmd/clone-legacy-org --apply             # clone entry + manager dari DB legacy
go run ./cmd/sync-sarpras                         # tarik titik koperasi dari portalkdkmp.id
go run ./cmd/sync-sdm --apply                     # derive titik sarpras → sdm_kdkmp_entries
```

Daftar lengkap command ada di `cmd/README.md`; runbook di `docs/`.

## 2. Menjalankan Frontend

```sh
cd ../ebitda-refactor-web

# 1. Install dependency
npm install

# 2. Pastikan .env.local menunjuk API dan JWT_SECRET sama dengan backend
#    NEXT_PUBLIC_API_URL=http://localhost:4000/api/v1
#    JWT_SECRET=dev-jwt-secret-change-me

# 3. Jalankan dev server (port 3000)
npm run dev
```

Buka <http://localhost:3000> lalu login memakai akun seed di atas.

> `JWT_SECRET` di `.env.local` frontend **harus sama** dengan `JWT_SECRET` di
> `.env` backend; nilai ini dipakai proxy Next.js untuk memverifikasi access
> token di Edge.

## Verifikasi & Kualitas

```sh
# Backend
go vet ./... && go test ./...

# Frontend
cd ../ebitda-refactor-web && npm run lint && npm run build
```

## Port & Cookie

| Layanan | Port |
|---|---|
| API Go | 4000 |
| Web Next.js | 3000 |
| PostgreSQL | 5433 |
| Redis | 6380 |
| MinIO API / Console | 9000 / 9001 |

Auth web memakai cookie `ebitda_access` (JWT, 1 jam) dan `ebitda_refresh`
(rotating, 7 hari). Klien API non-browser memakai header
`Authorization: Bearer <access_token>`.

## Catatan

- Jangan mengubah schema di luar migrasi yang disetujui (lihat `BACKLOG.md`).
- Detail operasional: `docs/S9_OPERATIONS.md`, `docs/S15_SARPRAS_SYNC.md`,
  `docs/LOCAL_DEMO_DATA.md`, `docs/LARK_SSO.md`.
