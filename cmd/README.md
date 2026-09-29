# Command CLI (`cmd/`)

Setiap folder di `cmd/` adalah program mandiri (`package main`) dan dijalankan
dengan `go run ./cmd/<nama-folder>`. API server berada di root repo
(`go run .`), bukan di sini.

## Cara menemukan command

```sh
go list ./cmd/...                 # daftar semua command
go run ./cmd/<nama-folder> --help # daftar flag command tersebut
```

Komentar pertama setiap `cmd/<folder>/main.go` juga menjelaskan tujuan dan
contoh pemakaiannya.

## Daftar command

| Command | Fungsi | Kapan dipakai | Flag penting | Efek default |
|---|---|---|---|---|
| `go run .` | API server (port 4000) + scheduler sarpras (env-gated) | development harian | `APP_PORT`, `SARPRAS_SCHEDULER_ENABLED`, `SARPRAS_SYNC_INTERVAL` | jalan terus |
| `go run ./cmd/seed` | Seed role + akun dev (superadmin, manager contoh) | DB baru atau akun dev hilang | — | idempotent (aman diulang) |
| `go run ./cmd/clone-legacy-org` | Clone entry KDKMP + Manager KDKMP + akun Manager Wilayah demo dari DB legacy lokal | memperkaya data dev | `--apply`, `--limit N`, `--regional-managers N`, `--reset-manager-passwords N` | **dry-run** (tulis hanya dengan `--apply`) |
| `go run ./cmd/sync-sarpras` | Tarik titik koperasi dari portal pembangunan | menyegarkan data peta/derive | `--dry-run`, `--max-pages N`, `--page-size N` | **langsung tulis** (pakai `--dry-run` lebih dulu) |
| `go run ./cmd/sync-sdm` | Derive titik sarpras → `sdm_kdkmp_entries` (by NIK) | setelah `sync-sarpras` | `--apply` | **dry-run** (tulis hanya dengan `--apply`) |
| `go run ./cmd/import-legacy-kdkmp-demo` | Impor 1 fixture demo KDKMP (tanggal digeser ke hari ini) | demo lokal | `--apply`, `--source-sdm-entry-id N` | **dry-run** (tulis hanya dengan `--apply`) |
| `go run ./cmd/migrate-legacy-kdkmp` | Migrasi penuh Manager KDKMP (cutover/rehearsal) | hanya untuk target **kosong** | `--apply`, `--rehearsal`, `--without-files` | **dry-run** (tulis hanya dengan `--apply`) |

## Contoh pemakaian

```sh
# data dev: pratinjau lalu tulis
go run ./cmd/clone-legacy-org --regional-managers=7 --reset-manager-passwords=3
go run ./cmd/clone-legacy-org --apply --regional-managers=7 --reset-manager-passwords=3

# sinkronisasi sarpras: uji parsial → penuh
go run ./cmd/sync-sarpras --dry-run --max-pages 2
go run ./cmd/sync-sarpras

# derive ke data SDM
go run ./cmd/sync-sdm
go run ./cmd/sync-sdm --apply
```

## Dokumen terkait

- `docs/S15_SARPRAS_SYNC.md` — runbook `sync-sarpras` & `sync-sdm` (env, jadwal,
  troubleshooting, E2E).
- `docs/LOCAL_DEMO_DATA.md` — `import-legacy-kdkmp-demo` & `clone-legacy-org`.
- `docs/S9_DATA_MIGRATION.md` / `docs/S9_REHEARSAL_RUNBOOK.md` —
  `migrate-legacy-kdkmp`.
