# BACKLOG — EBITDA Max APN (Refactor Go + Next.js)

**Scope: role `manager` (Kepala Toko/KDKMP) + `manager-wilayah` (regional) + `superadmin`.**
Modul APN corporate & workflow KDKMP Gerai (`ebitda_kdkmp`) **di luar scope**.

Status: `todo` / `in_progress` / `review` / `done` / `blocked` · Prioritas: **P0** wajib, **P1** penting, **P2** opsional.

Repos: API `ebitda-refactor` (Go/Gin) · Web `ebitda-refactor-web` (Next.js 16)

## Peta Epic

| # | Epic | Sprint | Status |
|---|------|--------|--------|
| E0 | Foundation & Infrastruktur | S0 | done |
| E1 | Auth Core (session, login, reset) | S1 | done |
| E2 | Auth Lanjutan (2FA, passkeys, notifikasi, onboarding) | S2 | done* |
| E3 | Master Data Manager (users, roles, kategori, task) | S3 | in_progress |
| E4 | Task Management (dashboard, report, dokumen) | S4 | todo |
| E5 | Dashboard KDKMP (metrik, matriks, seleksi, skor) | S5 | todo |
| E6 | Monitoring Superadmin (admin KDKMP, konsolidasi, announcement) | S6 | todo |
| E7 | SDM & Monitoring Nasional (import, sarpras, peta) | S7 | todo |
| E8 | Meeting Minutes & Pelengkap (action items, LMS, Lumbung) | S8 | todo |
| E9 | Parity, Data Migration & Cutover | S9 | todo |

## Sprint 0 — Foundation ✅

| ID | Story | Status |
|----|-------|--------|
| S0-1 | Docker compose (Postgres `ebitdamax_apn`, Redis, MinIO) | done |
| S0-2 | Migrasi goose schema existing (41 tabel, parity) — **skema beku** | done |
| S0-3 | Scaffold Go API (GORM, Redis, MinIO, healthz, Dockerfile) | done |
| S0-4 | Swagger/OpenAPI (swaggo) | done |
| S0-5 | Scaffold Next.js 16 (shadcn, api-client, proxy guard) | done |
| S0-6 | CI kedua repo + BACKLOG/SPRINT | done |

## Sprint 1 — Auth Core (BERJALAN)

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S1-1 | Session manager Redis + cookie `ebitda_session` (HttpOnly, SameSite=Lax) | P0 | done |
| S1-2 | Endpoint `POST /auth/login`, `POST /auth/logout`, `GET /auth/me` + middleware auth | P0 | done |
| S1-3 | Seeder `cmd/seed`: roles (superadmin, manager, manager-wilayah) + akun superadmin & manager | P0 | done |
| S1-4 | Middleware `RequireLevels` (mirror role.level) | P0 | done |
| S1-5 | CORS (kredensial + origin whitelist) | P0 | done |
| S1-6 | ~~Lupa/reset password (email)~~ | — | **cancelled** (tidak dibutuhkan) |
| S1-7 | Halaman web: login + dashboard awal (server fetch, guard 401) | P0 | done |
| S1-8 | Layout shell: sidebar + header, navigasi per role | P0 | done |
| S1-9 | Settings profil: edit profil, ganti password | P0 | done |
| S1-10 | Tema merah-putih (light/dark) di token shadcn | P1 | done |

## Sprint 2 — Auth Lanjutan

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S2-1 | 2FA TOTP backend: enable/confirm/disable + recovery codes + challenge login | P0 | done |
| S2-2 | Passkeys/WebAuthn backend: register, login, kelola perangkat | P0 | done |
| S2-3 | Notifikasi DB: list, read, read-all, bell | P0 | skip (tidak dibutuhkan saat ini) |
| S2-4 | Onboarding manager (`has_completed_onboarding` + tour) | P1 | done |
| S2-5 | Settings security & appearance | P0 | done |
| S2-6 | Halaman web: aktivasi 2FA (QR + kode + recovery codes) + challenge saat login | P0 | done |
| S2-7 | Halaman web: login & kelola passkey | P0 | hold (menunggu keputusan; fitur jarang dipakai) |

## Sprint 3 — Master Data Manager

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S3-1 | Users KDKMP CRUD + regional assignments + SK document (MinIO) | P0 | done |
| S3-2 | Roles KDKMP CRUD (backend + halaman web) | P0 | done |
| S3-3 | Task Categories CRUD | P0 | todo |
| S3-4 | Tasks CRUD (multi-role, period, BMC, cost JSONB, additional fields) | P0 | todo |
| S3-5 | Verifikasi rebuild DB dari nol (goose + seed) | P1 | todo |

## Sprint 4 — Task Management

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S4-1 | Task dashboard harian (task per role, period key) | P0 | todo |
| S4-2 | Start/finish report + additional fields | P0 | todo |
| S4-3 | Upload foto + dokumen ke MinIO (fase start/finish) | P0 | todo |
| S4-4 | Preview/download dokumen & foto | P0 | todo |
| S4-5 | Riwayat task selesai (14 hari) | P0 | todo |

## Sprint 5 — Dashboard KDKMP

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S5-1 | Metrics harian (revenue, cost, durasi, completion, time compliance) | P0 | todo |
| S5-2 | Actual variable cost + financial matrix harian | P0 | todo |
| S5-3 | Task selection + bundle BMC | P0 | todo |
| S5-4 | Kehadiran operasional + alokasi personel | P0 | todo |
| S5-5 | Performance scoring + upsert/input harian | P0 | todo |

## Sprint 6 — Monitoring Superadmin

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S6-1 | Admin KDKMP dashboard (filter wilayah, status, summary) | P0 | todo |
| S6-2 | Task report per entry/tanggal | P0 | todo |
| S6-3 | Konsolidasi wilayah (province/regency/district/village) | P0 | todo |
| S6-4 | Regional access (row-level + locked filters) | P0 | todo |
| S6-5 | Announcements (notifikasi per role) | P0 | todo |

## Sprint 7 — SDM & Monitoring Nasional

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S7-1 | SDM Data KDKMP (index, update jumlah karyawan) | P0 | todo |
| S7-2 | Import koperasi karyawan (NIK + resolusi provinsi) | P0 | todo |
| S7-3 | Cron sync sarpras status (15 menit) | P0 | todo |
| S7-4 | Peta monitoring: meta + binary payload + Leaflet | P0 | todo |
| S7-5 | Integrasi portal eksternal (cache Redis) | P1 | todo |

## Sprint 8 — Meeting & Pelengkap

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S8-1 | Meeting minutes CRUD + items + attachment MinIO | P0 | todo |
| S8-2 | Status history + action items | P0 | todo |
| S8-3 | LMS KDKMP & Lumbung Chat | P2 | todo |
| S8-4 | Polish UI + pengaturan sisa | P1 | todo |
| S8-5 | Integrasi SSO Lark (login via Lark Platform) — menunggu akses/API Lark; desain: OAuth/OIDC callback + mapping user | P1 | todo |

## Sprint 9 — Parity & Cutover

| ID | Story | Prioritas | Status |
|----|-------|-----------|--------|
| S9-1 | Audit parity route/halaman scope manager | P0 | todo |
| S9-2 | Migrasi data production (hanya tabel in-scope) + verifikasi | P0 | todo |
| S9-3 | Hardening keamanan + performance pass | P0 | todo |
| S9-4 | Backup, rollback plan, switch DNS | P0 | todo |
| S9-5 | Dokumentasi & monitoring pasca-cutover | P1 | todo |
