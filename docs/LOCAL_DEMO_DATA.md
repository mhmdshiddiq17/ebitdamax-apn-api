# Import Data Demo KDKMP

Command ini mengimpor satu fixture demo Manager KDKMP dari database legacy
lokal. Ia tidak mengimpor akun legacy, NIK asli, foto, dokumen, Plan EBITDA
Matrix, atau Meeting Minutes.

Prasyarat: PostgreSQL refactor berjalan melalui Docker, akun Manager seed
sudah dibuat dengan `go run ./cmd/seed`, dan variabel `LEGACY_DB_*` tersedia
di `.env` atau environment shell.

Periksa data tanpa menulis apa pun:

```sh
go run ./cmd/import-legacy-kdkmp-demo
```

Jika ringkasan sudah benar, lakukan import:

```sh
go run ./cmd/import-legacy-kdkmp-demo --apply
```

Command berhenti bila master task sudah terisi, akun Manager sudah terhubung
ke KDKMP lain, atau marker `DEMO-KDKMP-LEGACY` sudah ada. Hal ini mencegah
data kerja lokal tertimpa oleh fixture demo.

## Clone Data Organisasi dari Legacy (DEV-1)

Untuk memperkaya data dev (bukan satu fixture), gunakan
`cmd/clone-legacy-org`. Command ini menyalin entry KDKMP + akun Manager KDKMP
dari database legacy lokal (`LEGACY_DB_*`, default `ebitda`) ke database
refactor secara incremental & idempotent (entry upsert by `nik`, user by
`lower(email)`, tautan manager–entry via peta NIK).

```sh
# pratinjau rencana (tidak menulis)
go run ./cmd/clone-legacy-org --regional-managers=7 --reset-manager-passwords=3

# tulis
go run ./cmd/clone-legacy-org --apply --regional-managers=7 --reset-manager-passwords=3
```

- `--limit N` membatasi jumlah entry & manager (uji cepat).
- `--regional-managers N` membuat akun `manager-wilayah-<provinsi>@agrinas.test`
  (`password123`) + assignment scope province untuk N provinsi terbanyak.
- `--reset-manager-passwords N` mereset N manager pertama (punya entry) ke
  `password123`; manager lain memakai hash legacy (tidak bisa login).
- Password/2FA/SK legacy tidak disalin; riwayat harian, laporan, dan objek
  berkas tidak ikut (keputusan S9). Backup disarankan sebelum `--apply`
  (`pg_dump` database refactor).
- Detail riwayat eksekusi dan angka hasil ada di `SPRINT.md`
  ("Persiapan Data Dev (pra-Sprint 15)").
