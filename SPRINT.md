# SPRINT — EBITDA Max APN (Refactor, scope manager)

Dokumen per-sprint: goal, task, hasil review, retro.
Durasi sprint: **2 minggu** · Metode: agile · Scope: `manager` + `manager-wilayah` + `superadmin`.

## Definition of Done (DoD)

1. Berjalan lokal via `docker compose up` (API + Web + infra)
2. Test tersedia & lulus (`go test ./...` / test web)
3. `go vet` + lint/typecheck kedua repo hijau
4. Endpoint baru terdaftar di Swagger (`swag init`)
5. Tidak ada regresi sprint sebelumnya
6. Dicatat di `BACKLOG.md` dengan status `done`
7. **Skema DB tidak berubah** (schema beku — 41 tabel existing)

---

## Sprint 0 — Foundation ✅ (SELESAI)

**Goal:** Infrastruktur + kedua aplikasi berjalan, schema DB existing dipakai apa adanya.

**Deliverable:** docker compose (Postgres 5433, Redis 6380, MinIO), migrasi goose 41 tabel (parity), Go API + healthz + Dockerfile + Swagger, Next.js 16 + shadcn + apiFetch + proxy guard, CI kedua repo.

**Retro:** pendekatan dump-schema menjamin parity; filter `\restrict`/`search_path` wajib saat dump ulang; `swag init` tiap tambah endpoint.

---

## Sprint 1 — Auth Core (BERJALAN)

**Goal:** Manager & superadmin dapat login/logout dengan session Redis; navigasi dasar per role; halaman auth & profil.

**Status per package:**
- [x] **Pkg 1:** Session Redis + cookie, login/logout/me, middleware auth + RequireLevels, CORS, seeder roles+users, update BACKLOG/SPRINT
- [x] **Pkg 2:** Halaman web login + dashboard awal + tema merah-putih (reset password **dibatalkan** — tidak dibutuhkan)
- [x] **Pkg 3:** Layout shell: sidebar collapsible + header (navigasi per role, theme toggle, user menu + logout)
- [x] **Pkg 4:** Settings profil (edit profil, ganti kata sandi)

**Catatan teknis Pkg 1:**
- Session: `internal/session` (Redis, sliding expiration, test miniredis)
- Cookie: `ebitda_session`, HttpOnly, SameSite=Lax, Secure saat production
- Seed: `go run ./cmd/seed` (idempotent) — superadmin@agrinas.test / manager@agrinas.test (password dev: `password123`)
- CORS: whitelist via `CORS_ALLOWED_ORIGINS`

**Catatan teknis Pkg 2:**
- Design system: **palet merah-putih** pada token shadcn (`globals.css` light/dark), tanpa warna mentah
- `src/lib/server-api.ts`: fetch server-side yang meneruskan cookie session
- `/login` (client form) → `POST /auth/login`; `/dashboard` (server) → `GET /auth/me`, redirect 401 ke `/login`
- Logout via `src/components/logout-button.tsx`

**Catatan teknis Pkg 3:**
- Route group `(app)` — layout shell server component: fetch `/auth/me` + redirect 401
- Navigasi per role di `src/lib/navigation.ts` (manager, manager-wilayah, superadmin); item belum dibangun berstatus `soon` (tampil non-aktif, label "segera")
- Komponen: shadcn `sidebar` (collapsible), `dropdown-menu`, `avatar`, `breadcrumb`; Base UI memakai prop `render` (bukan `asChild`)
- Tema: `next-themes` (class, system) + toggle; cookie session via konstanta bersama `src/lib/constants.ts`
- Hook `use-mobile` bawaan shadcn diganti `useSyncExternalStore` agar lolos lint React 19

**Catatan teknis Pkg 4:**
- Endpoint: `PATCH /api/v1/profile` (nama, email + cek email unik) dan `PUT /api/v1/profile/password` (current password + min 8 + konfirmasi)
- Validasi bahasa Indonesia; perubahan email **tidak** mereset `email_verified_at` (tidak ada alur verifikasi di scope ini)
- Halaman `/settings/profile` (route group `(app)`) + menu sidebar "Pengaturan" aktif

**Review Sprint 1 (✅ goal tercapai):**
- Manager & superadmin bisa login/logout dengan session Redis; navigasi berbeda per role; profil & kata sandi bisa diubah.
- Terverifikasi E2E: login/me/logout, session TTL 7 hari, redirect guard (proxy + server), settings API (6 kasus validasi), halaman settings render.

**Retro Sprint 1:**
- 🟢 Keep: pola package kecil + verifikasi E2E tiap langkah; seeder idempotent memudahkan rebuild
- 🟡 Improve: selalu matikan proses `go run`/`next dev` lama sebelum E2E (sempat 404 karena port 4000 masih dipegang proses lama)
- 🔵 Catatan Pkg 4: ganti kata sandi belum meng-invalidate session lain — kandidat hardening di S9

---

## Sprint 2 — Auth Lanjutan ✅ (SELESAI)

**Goal:** 2FA TOTP, passkeys (WebAuthn), notifikasi in-app, onboarding manager, settings security/appearance.

**Status per package:**
- [x] **Pkg 1:** 2FA TOTP backend (enable/confirm/disable, recovery codes sekali-pakai, challenge login + batas percobaan)
- [x] **Pkg 2:** Halaman web keamanan: aktivasi 2FA (QR + kode + recovery codes) & challenge saat login
- [x] **Pkg 3:** Passkeys backend (WebAuthn register/login/kelola)
- [ ] **Pkg 4:** Passkeys frontend — **HOLD** (fitur jarang dipakai; menunggu keputusan SSO Lark)
- [ ] **Pkg 5:** Notifikasi DB + bell + halaman notifikasi — **SKIP** (tidak dibutuhkan saat ini)
- [x] **Pkg 6:** Onboarding manager + settings security/appearance

**Catatan teknis Pkg 1:**
- Endpoint: `GET /two-factor`, `POST /two-factor/enable` (secret+URI), `POST /two-factor/confirm` (kode pertama → recovery codes), `POST /two-factor/recovery-codes`, `DELETE /two-factor`, `POST /auth/two-factor-challenge`
- Secret TOTP dienkripsi at-rest (AES-256-GCM, kunci turunan `APP_KEY`) via `internal/crypto`; recovery codes disimpan **hash bcrypt** (hanya tampil sekali — beda dari Fortify yang menyimpan terenkripsi)
- Challenge pending di Redis (`twofactor:challenge:*`, TTL 5 menit, cookie `ebitda_2fa`), maks 5 percobaan lalu challenge dihapus
- Login: bila 2FA aktif → `{two_factor_required:true}` + cookie challenge; verifikasi → session penuh
- Catatan migrasi S9: secret 2FA lama (format Laravel) tidak dapat didekripsi → user terkait harus enroll ulang
- Test: `internal/crypto` + `internal/twofactor` (recovery code, challenge lifecycle, TOTP) — E2E 18 skenario lolos

**Catatan teknis Pkg 2:**
- Halaman `/settings/security`: kartu 2FA dengan state machine (status → password → QR/kode → recovery codes → regenerate/disable)
- QR via `qrcode.react`; recovery codes ditampilkan sekali + tombol salin (`navigator.clipboard`)
- `/two-factor-challenge`: form kode/recovery code; login form otomatis redirect saat `two_factor_required`
- `SettingsNav` (Profil | Keamanan) di halaman settings; `proxy.ts` menangani `/two-factor-challenge`
- E2E UI (Playwright): login → aktivasi 2FA (QR+secret) → konfirmasi → recovery codes tampil → logout → login tertahan 2FA → verifikasi TOTP → dashboard → disable. Semua lolos.
- ⚠️ Satu jalur belum terverifikasi via UI otomatis: submit **recovery code** di halaman challenge (klik otomatis tidak memicu submit saat diuji; submit TOTP & jalur error terbukti jalan, API recovery code lolos uji curl). Disarankan cek manual 1 menit saat review.

**Catatan teknis Pkg 3:**
- Library `github.com/go-webauthn/webauthn v0.18.2`; Relying Party dikonfigurasi via env (`WEBAUTHN_RP_ID`, `WEBAUTHN_RP_ORIGINS`, `WEBAUTHN_RP_DISPLAY_NAME`); service nonaktif (503) bila config kosong
- Endpoint: `GET /passkeys`, `POST /passkeys/register/options|register` (auth), `DELETE /passkeys/{id}` (auth, cek kepemilikan via `user_id`), `POST /auth/passkey/options|login` (publik)
- Challenge ceremony disimpan di Redis (`passkey:register:*`, `passkey:login:*`, TTL 5 menit, cookie `ebitda_passkey`); `credential` disimpan sebagai JSON `webauthn.Credential`, `credential_id` = base64url; counter/`last_used_at` diperbarui setiap login sukses
- Login passkey langsung membuat session penuh (passkey dianggap faktor kuat, tidak melewati challenge 2FA)
- Migrasi S9: passkey lama (format Laravel) tidak kompatibel → user enroll ulang
- Unit test: enabled/disabled, challenge roundtrip + expiry, adapter user, encoding credential ID
- **E2E Playwright dengan virtual authenticator (WebAuthn sungguhan)**: registrasi `201` (attestation terverifikasi) → login passkey `200` + session + `last_used_at` terisi → opsi akun tanpa passkey `422` → finish tanpa challenge `422` → hapus passkey milik user lain oleh superadmin `404` → hapus milik sendiri `200` → list kosong → opsi setelah hapus `422`. Semua lolos.

**Catatan teknis Pkg 6:**
- Endpoint `POST /users/complete-onboarding` (auth; hanya manager domain kdkmp → selain itu 403; idempotent)
- Tour onboarding port custom (tanpa library): highlight + popover, `sessionStorage`, Escape = lewati, 3 langkah awal (sidebar, area kerja, menu akun); langkah KDKMP/task ditambahkan saat halaman terkait dibangun (Sprint 4–5)
- Atribut `data-tour` dipasang di `app-sidebar`, konten layout, `user-menu`; overlay memakai token baru `--overlay`
- Halaman `/settings/appearance` (Terang/Gelap/Sistem) via `next-themes` + item "Tampilan" di `SettingsNav`
- **E2E Playwright**: flag reset → login manager → tour muncul (Langkah 1/3) → Lanjutkan 2× → Selesai → flag `true` di DB → reload tanpa tour ulang; appearance: Gelap → `html.dark` + persist reload → Terang → Sistem; API: tanpa session `401`, superadmin `403`, idempotent `200`

**Review Sprint 2 (✅ goal tercapai dengan penyesuaian prioritas):**
- 2FA TOTP lengkap (backend + UI + challenge login), passkeys backend WebAuthn lengkap, onboarding manager + appearance settings selesai.
- Passkeys frontend di-hold (jarang dipakai, menunggu arah SSO Lark); notifikasi di-skip.
- Terverifikasi E2E browser: 2FA lifecycle 18 skenario, passkey ceremony penuh via virtual authenticator, tour onboarding, ganti tema.

**Retro Sprint 2:**
- 🟢 Keep: pengujian ceremony WebAuthn via CDP virtual authenticator sangat efektif (tanpa mock); pola state turunan (`isActive`) menghindari lint React 19
- 🟡 Improve: verifikasi submit recovery code di UI (1 jalur belum tercakup automation)
- 🔵 Catatan: migrasi S9 — secret 2FA & passkey lama (format Laravel) tidak kompatibel → user enroll ulang; kandidat hardening invalidasi session saat ganti kata sandi

---

## Sprint 3 — Master Data Manager (BERJALAN)

**Goal:** Master data yang dibutuhkan fitur manager: Roles, Users KDKMP (+regional assignment, SK), Task Categories, Tasks.

**Status per package:**
- [x] **Pkg 1:** Roles CRUD — backend (list/create/update/delete + guard) & halaman web (tabel, cari, urut, paginasi, dialog form, konfirmasi hapus)
- [x] **Pkg 2:** Users KDKMP CRUD + regional assignment + upload/preview SK (MinIO)
- [ ] **Pkg 3:** Task Categories CRUD
- [ ] **Pkg 4:** Tasks CRUD (multi-role, period, BMC, cost JSONB, additional fields)
- [ ] **Pkg 5:** Verifikasi rebuild DB + E2E

**Catatan teknis Pkg 1:**
- Endpoint superadmin-only (`RequireLevels(superadmin)`): `GET/POST /roles`, `PUT/DELETE /roles/{id}`; list filter domain (default `kdkmp`), search, sort whitelist (name/level/created_at), paginasi 15
- Slug otomatis unik via `internal/slug` (`role-uji`, `role-uji-2`, …); nama unik case-insensitive; guard hapus: dipakai user → 409, dipakai task → 409
- Halaman `/roles`: TanStack Query + initialData SSR, debounce pencarian 350ms, dialog form (Base UI, reset via `key`), AlertDialog konfirmasi; akses langsung non-superadmin → redirect `/dashboard`
- Komponen shadcn baru: `table`, `dialog`, `alert-dialog`, `select`
- E2E API (12 skenario) + E2E Playwright UI (create/edit/search/delete/guard 403) — semua lolos; bug `created_at` NULL pada role hasil seed lama diperbaiki (data + seeder backfill)

**Catatan teknis Pkg 2:**
- Endpoint superadmin: `GET/POST /users`, `PUT/DELETE /users/{id}`, `POST /users/{id}/manager-sk-document`, `GET /region-options`, `GET /kdkmp-options`; preview SK `GET /users/{id}/manager-sk-document` (superadmin atau manager pemilik)
- Validasi mirror app lama: manager wajib terhubung 1 data KDKMP (unik antar akun), manager-wilayah wajib ≥1 cakupan wilayah (maks 25, tanpa duplikat, wilayah harus punya KDKMP terkelola), role lain tidak boleh punya cakupan; email unik (case-insensitive, disimpan lowercase); password min 8 + konfirmasi
- SK Manager disimpan di MinIO (`manager-sk/{userId}/{uuid}.pdf`, PDF maks 10 MB) + metadata JSON di `users.manager_sk_document`; upload ulang menghapus file lama; `email_verified_at` diisi saat create (tidak ada alur verifikasi email), username dibuat otomatis unik
- Halaman `/users`: tabel + filter role + pencarian + paginasi, dialog form dengan editor cakupan cascading (provinsi → kabupaten → kecamatan dari `/region-options`), unggah/pratinjau SK dari baris tabel (FormData), hapus dengan konfirmasi; `apiFetch` diperluas untuk `FormData`
- Catatan: kolom `lark_open_id` ada di migrasi Laravel yang lebih baru (SSO Lark) — belum ada di schema kita; dibahas saat S8 (SSO) / S9 (migrasi data)

**Review Sprint 3:** (isi saat review)
**Retro Sprint 3:** (isi saat retro)

---

## Sprint 4 — Task Management (RENCANA)

**Goal:** Dashboard task harian, start/finish report, dokumen MinIO, riwayat.

**Backlog:** S4-1 … S4-5

---

## Sprint 5 — Dashboard KDKMP (RENCANA)

**Goal:** Metrik harian, financial matrix, task selection, kehadiran, scoring, input harian.

**Backlog:** S5-1 … S5-5

---

## Sprint 6 — Monitoring Superadmin (RENCANA)

**Goal:** Admin KDKMP dashboard, task report per entry, konsolidasi wilayah, regional access, announcements.

**Backlog:** S6-1 … S6-5

---

## Sprint 7 — SDM & Monitoring Nasional (RENCANA)

**Goal:** SDM data, import koperasi, cron sarpras, peta monitoring, portal eksternal.

**Backlog:** S7-1 … S7-5

---

## Sprint 8 — Meeting & Pelengkap (RENCANA)

**Goal:** Meeting minutes + action items, LMS/Lumbung, polish.

**Backlog:** S8-1 … S8-4

---

## Sprint 9 — Parity & Cutover (RENCANA)

**Goal:** Audit parity, migrasi data in-scope, hardening, switch production.

**Backlog:** S9-1 … S9-5
