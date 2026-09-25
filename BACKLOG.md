# BACKLOG — EBITDA Max APN (Refactor Paritas Manager KDKMP)

`../ebitdamax-apn` adalah sumber perilaku. API Go dan web Next.js hanya
merefaktor fitur yang dipakai langsung oleh **Manager KDKMP**:
`roles.domain = kdkmp` dan `roles.slug = manager`.

`manager-wilayah` dan `superadmin` hanya berada dalam scope sebagai peran
pendukung pengelolaan akun, role, task, dan data Manager KDKMP. Mereka bukan
target fitur baru tersendiri.

Repos: API `ebitda-refactor` (Go/Gin) · Web `ebitda-refactor-web` (Next.js 16)

## Aturan Backlog

- Status: `todo`, `in_progress`, `review`, `done`, `blocked`, `hold`, `skip`,
  atau `deferred`.
- **P0** wajib untuk paritas/cutover, **P1** penting, **P2** opsional.
- Baseline database adalah `migrations/00001_initial_schema.sql`. Tidak ada
  perubahan schema pada Sprint 7–9 tanpa persetujuan eksplisit.
- Setiap item `hold`, `skip`, `blocked`, dan `deferred` menyebutkan alasannya;
  item tersebut tidak dihitung sebagai pekerjaan sprint aktif.

## Batas Scope

| Kategori | Keputusan |
|---|---|
| Manager KDKMP | Target paritas utama: auth, profil, task, dashboard, perencanaan, dan kolaborasi. |
| Master data pendukung | Selesai pada Sprint 3 dan dipertahankan hanya untuk mendukung Manager KDKMP. |
| APN corporate | Di luar scope: organisasi, EBITDA tree/value, kalkulasi, value chain, dan dashboard APN. |
| Monitoring non-manager | Di luar scope aktif: dashboard regional/superadmin, SDM nasional, import Excel, peta, sarpras, dan portal eksternal. |
| Integrasi eksternal | Tidak dikerjakan tanpa kontrak akses/SSO yang siap; dicatat sebagai `blocked`. |

## Peta Epic

| # | Epic | Sprint | Status |
|---|---|---|---|
| E0 | Foundation & Infrastruktur | S0 | done |
| E1 | Auth Core | S1 | done |
| E2 | Auth Lanjutan & Onboarding | S2 | done* |
| E3 | Master Data Pendukung Manager | S3 | done* |
| E4 | Task Management Manager | S4 | done |
| E5 | Dashboard Harian Manager KDKMP | S5 | done |
| E7 | Kolaborasi Manager | S7 | done |
| E8 | Kesiapan Manager | S8 | done |
| E9 | Paritas & Cutover Manager | S9 | in_progress |

`*` Epic selesai dengan item hold/skip yang tercatat di bagian status khusus.

## Sprint 0 — Foundation

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S0-1 | Compose Postgres, Redis, dan MinIO untuk lingkungan lokal | P0 | done |
| S0-2 | Goose baseline schema parity | P0 | done |
| S0-3 | Scaffold Go API, health check, Dockerfile, dan Swagger | P0 | done |
| S0-4 | Scaffold Next.js, API client, dan proxy guard | P0 | done |

## Sprint 1 — Auth Core

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S1-1 | Session Redis, login, logout, dan current user | P0 | done |
| S1-2 | CORS, auth guard, serta navigasi per role | P0 | done |
| S1-3 | Profil, ganti kata sandi, tema, dan shell aplikasi | P0 | done |
| S1-4 | Seeder role dan akun pengembangan | P1 | done |
| S1-5 | Reset password melalui email | — | skip (tidak dibutuhkan) |

## Sprint 2 — Auth Lanjutan & Onboarding

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S2-1 | 2FA TOTP dan recovery code | P0 | done |
| S2-2 | Backend passkeys/WebAuthn | P1 | done |
| S2-3 | Onboarding Manager KDKMP dan pengaturan keamanan/tampilan | P1 | done |
| S2-4 | Halaman frontend passkeys | P2 | hold (menunggu keputusan SSO Lark) |
| S2-5 | Notifikasi in-app | P2 | skip (tidak dibutuhkan) |

## Sprint 3 — Master Data Pendukung Manager

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S3-1 | Roles KDKMP CRUD | P0 | done |
| S3-2 | Users KDKMP, regional assignment, dan dokumen SK | P0 | done |
| S3-3 | Task categories CRUD | P0 | done |
| S3-4 | Tasks CRUD: multi-role, BMC, biaya, dan field laporan | P0 | done |
| S3-5 | Rebuild database kosong + E2E | P1 | skip (ditangani saat cutover) |

## Sprint 4 — Task Management Manager

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S4-1 | Dashboard task sesuai role dan periode | P0 | done |
| S4-2 | Start/finish task, field dinamis, foto, dokumen, dan alokasi | P0 | done |
| S4-3 | Riwayat selesai serta preview/download dokumen | P0 | done |

## Sprint 5 — Dashboard Harian Manager KDKMP

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S5-1 | Metrics revenue, cost, durasi, completion, dan time compliance | P0 | done |
| S5-2 | Actual variable cost dan financial matrix harian | P0 | done |
| S5-3 | Pilihan task dan bundle BMC | P0 | done |
| S5-4 | Kehadiran operasional dan guard alokasi | P0 | done |
| S5-5 | Input/upsert harian, scoring, dan sinkronisasi task selesai | P0 | done |
| S5-6 | Revisi Financial Matrix dengan Recharts; tabel sebagai alternatif aksesibel | P1 | done |

## Sprint 7 — Kolaborasi Manager

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S7-1 | List/create/update/delete Meeting Minutes milik manager | P0 | done |
| S7-2 | Item meeting, urutan, PIC, tenggat, dan status | P0 | done |
| S7-3 | Attachment meeting pada MinIO beserta preview/download terotorisasi | P1 | done |
| S7-4 | Riwayat status dan halaman Action Items Manager | P1 | done |
| S7-5 | Halaman web, navigasi, dan verifikasi owner/access boundary | P0 | done |

## Sprint 8 — Kesiapan Manager

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S8-1 | Tampilan dokumen SK Manager pada profil menggunakan endpoint yang sudah ada | P1 | done |
| S8-2 | Lengkapi navigasi dan onboarding untuk dashboard, task, matrix, dan meeting | P1 | done |
| S8-3 | Polish mobile, aksesibilitas, empty state, dan error state seluruh alur manager | P1 | done |
| S8-4 | Verifikasi read-only mapping task/field revenue dan biaya terhadap data produksi | P0 | done |

## Sprint 9 — Paritas & Cutover Manager

| ID | Story | Prioritas | Status |
|---|---|---|---|
| S9-1 | Audit route, halaman, dan gate terhadap fitur Manager KDKMP legacy | P0 | done |
| S9-2 | Migrasi dan verifikasi data hanya untuk tabel in-scope yang sudah ada | P0 | done |
| S9-3 | Hardening session, akses, validasi, dan performance pass | P0 | done |
| S9-4 | Rebuild database, backup, rollback plan, dan cutover | P0 | todo |
| S9-5 | Dokumentasi operasional dan monitoring pasca-cutover | P1 | todo |

## Blocked — Kontrak Eksternal Belum Tersedia

| ID | Story | Prioritas | Status |
|---|---|---|---|
| B-1 | LMS KDKMP iframe untuk manager | P1 | blocked (URL, akses, dan kontrak sesi belum tersedia) |
| B-2 | Lumbung Chat iframe | P2 | blocked (URL dan kontrak akses eksternal belum tersedia) |
| B-3 | SSO Lark | P1 | blocked (akses/API Lark dan parity kolom user belum tersedia) |

## Deferred — Setelah Seluruh Sprint Inti

| ID | Story | Prioritas | Status |
|---|---|---|---|
| D-1 | POS Revenue read-only pada Dashboard KDKMP | P1 | deferred (arahan user: setelah seluruh sprint) |
| D-2 | Customer Analysis Manager KDKMP | P1 | deferred (tabel `customer_analyses` belum ada; perlu parity migration setelah Sprint 9) |
