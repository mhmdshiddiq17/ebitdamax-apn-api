# SPRINT — EBITDA Max APN (Refactor Paritas Manager KDKMP)

Dokumen per-sprint: goal, task, hasil review, dan retro. Aplikasi Laravel
`../ebitdamax-apn` adalah sumber perilaku; refactor Go/Next.js hanya mengejar
paritas fitur Manager KDKMP.

Durasi sprint: **2 minggu** · Metode: agile · Target utama:
`roles.domain=kdkmp` + `roles.slug=manager`.

## Batas Produk

- **In scope:** alur kerja, perencanaan, kolaborasi, dan keamanan yang dipakai
  langsung oleh Manager KDKMP.
- **Pendukung:** `manager-wilayah` dan `superadmin` hanya sejauh diperlukan
  untuk menyiapkan akun, role, task, dan data Manager KDKMP.
- **Di luar scope aktif:** APN corporate, organisasi, dashboard nasional,
  monitoring regional/superadmin, SDM nasional, import Excel, peta, dan portal
  eksternal.
- **Schema:** baseline adalah `migrations/00001_initial_schema.sql`; tidak ada
  perubahan schema selama Sprint 7–9. Fitur legacy yang tabelnya belum ada
  ditunda sampai seluruh sprint inti selesai.

## Definition of Done (DoD)

1. Infra lokal berjalan via `docker compose up -d`; API dan web berjalan
   terpisah melalui `go run .` dan `npm run dev`.
2. Test tersedia dan lulus (`go test -mod=readonly ./...`), sesuai kebutuhan
   perubahan.
3. `go vet ./...`, `npm run lint`, dan `npm run build` hijau.
4. Endpoint baru terdaftar di Swagger (`swag init`)
5. Tidak ada regresi sprint sebelumnya
6. Dicatat di `BACKLOG.md` dengan status `done`
7. Tidak ada perubahan schema di luar parity migration yang telah disetujui.

---

## Sprint 0 — Foundation ✅ (SELESAI)

**Goal:** Infrastruktur + kedua aplikasi berjalan, schema DB existing dipakai apa adanya.

**Deliverable:** docker compose (Postgres 5433, Redis 6380, MinIO), migrasi goose 41 tabel (parity), Go API + healthz + Dockerfile + Swagger, Next.js 16 + shadcn + apiFetch + proxy guard, CI kedua repo.

**Retro:** pendekatan dump-schema menjamin parity; filter `\restrict`/`search_path` wajib saat dump ulang; `swag init` tiap tambah endpoint.

---

## Sprint 1 — Auth Core ✅ (SELESAI)

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
- Logout melalui footer sidebar dan endpoint `POST /auth/logout`

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

## Sprint 3 — Master Data Manager ✅ (SELESAI)

**Goal:** Master data yang dibutuhkan fitur manager: Roles, Users KDKMP (+regional assignment, SK), Task Categories, Tasks.

**Status per package:**
- [x] **Pkg 1:** Roles CRUD — backend (list/create/update/delete + guard) & halaman web (tabel, cari, urut, paginasi, dialog form, konfirmasi hapus)
- [x] **Pkg 2:** Users KDKMP CRUD + regional assignment + upload/preview SK (MinIO)
- [x] **Pkg 3:** Task Categories CRUD (backend + halaman web)
- [x] **Pkg 4:** Tasks CRUD (multi-role, period, BMC, cost JSONB, additional fields)
- [ ] **Pkg 5:** Verifikasi rebuild DB + E2E — **SKIP** (arahan user; rebuild tetap dicakup saat S9 cutover)

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

**Catatan teknis Pkg 3:**
- Endpoint superadmin: `GET/POST /task-categories`, `PUT/DELETE /task-categories/{id}`; list search (nama/slug/deskripsi), sort (name/created_at), paginasi 15, `tasks_count` via subquery
- Nama unik case-insensitive, slug otomatis unik (`kategori-ui`, `kategori-ui-2`); guard hapus: dipakai task → 409
- Halaman `/task-categories`: tabel + pencarian + urutan + paginasi + dialog form (nama + deskripsi) + konfirmasi hapus; menu sidebar "Kategori Tugas" aktif, proxy `/task-categories`
- Komponen shadcn baru: `textarea`
- E2E API (10 skenario: guard, CRUD, validasi, slug, tasks_count, guard 409) + E2E Playwright UI (create/edit/search/delete) — semua lolos

**Catatan teknis Pkg 4:**
- Tipe custom GORM: `CostBreakdown` (JSONB, parsing lenient angka/bool, `_configured` ditulis **boolean** agar kompatibel dua arah dengan aplikasi lama), `StringList` (options JSONB), `ClockTime` (kolom `time`, dukung scan `pgtype.Time`/`time.Time`/string)
- Endpoint superadmin: `GET/POST /tasks`, `PUT/DELETE /tasks/{id}`; list filter search (nama/deskripsi/kategori/role), kategori, role, status (active/inactive/all), sort whitelist (sort_order/name/execution_time/time_require/created_at), paginasi 15
- Validasi mirror app lama: kategori & role valid (role unik, min 1), `sort_order` unik (min 1), estimasi ≥1 menit, ambang waktu harus berpasangan & bawah ≤ atas, periode/BMC/tipe input/show_when dari enum, fixed & variable cost wajib lengkap (4 komponen ≥ 0), label field wajib ≤255
- Sinkronisasi: pivot `task_roles` di-replace; field tambahan di-upsert per `id` dan yang tidak dikirim dihapus; `field_name` slug unik per task (`jumlah_pelanggan`, `catatan_2`); `options` hanya untuk select/radio/checkbox (kosong → null); guard hapus: sudah ada laporan → 409
- Halaman `/tasks`: tabel (urut, task+kategori+BMC, role, periode, waktu, biaya, status), filter kategori/role/status + pencarian + urutan + paginasi; dialog form besar (multi-role toggle, biaya 4+4 komponen dengan total, editor field dinamis dengan opsi per baris); util `formatRupiah` di `src/lib/formatters.ts`
- E2E API (16 validasi + CRUD + filter + guard) & E2E Playwright UI (create lengkap dengan field dinamis → edit prefilled → pencarian → hapus) — semua lolos

**Review Sprint 3 (✅ goal tercapai):**
- Master data lengkap: Roles CRUD, Users KDKMP (+regional assignment + SK via MinIO), Task Categories, Tasks (multi-role/BMC/biaya/field dinamis).
- Verifikasi: E2E API (~50 skenario) + E2E Playwright UI untuk keempat halaman.
- Pkg 5 (rebuild DB) di-skip atas arahan user → dicatat sebagai risiko kecil; verifikasi rebuild dicakup saat S9 cutover.

**Retro Sprint 3:**
- 🟢 Keep: pola paket (backend → E2E API → UI → E2E Playwright) konsisten dan menemukan bug dini (created_at NULL, GORM JSONB type, `_configured` bool)
- 🟡 Improve: container Docker sempat mati 2× → pastikan Docker Desktop berjalan sebelum sesi; pertimbangkan auto-start saat boot
- 🔵 Catatan: kompatibilitas data dua arah dijaga (`_configured` boolean, email lowercase, slug unik)

---

## Sprint 4 — Task Management ✅ (SELESAI)

**Goal:** Manager dapat melihat task harian sesuai role, memulai & menyelesaikan task (field dinamis, foto, dokumen), melihat riwayat selesai, serta pratinjau/unduh berkas.

**Status per package:**
- [x] **Pkg 1:** Task dashboard harian — backend `GET /task-dashboard` + halaman `/dashboard/tasks` (task per role, period key, status per periode, ringkasan)
- [x] **Pkg 2:** Start & finish task — endpoint + form (field tambahan dinamis, foto, dokumen MinIO, alokasi anggota), guard KDKMP (kehadiran & pemilihan task) — sinkronisasi metrik KDKMP menyusul di S5
- [x] **Pkg 3:** Riwayat task selesai (14 hari, ringkasan harian) + endpoint & aksi pratinjau/unduh dokumen/foto

**Catatan Sprint 4:**
- **Pkg 1** — endpoint `GET /task-dashboard` (staff/manager/superadmin): task aktif sesuai role user + period key (once/daily/weekly ISO/monthly), laporan terbaru per (task, periode), task **selesai disembunyikan untuk non-superadmin** namun tetap dihitung di summary; manager KDKMP dibatasi task wajib ATAU yang dipilih/dikerjakan hari ini; ringkasan kehadiran operasional (hadir/terpakai/tersedia) untuk manager KDKMP
- Service baru: `internal/kdkmp` (timezone bisnis `KDKMP_BUSINESS_TIMEZONE`, `SelectionService`, `AllocationService`); model `TaskReport`/`TaskReportValue`/`EbitdamaxKdkmp` + tipe JSON `JSONIntMap`, `IntList`, `StoredDocuments`
- Bug fix: `ClockTime.String()` nil-safe (panic saat jam pelaksanaan kosong)
- Halaman `/dashboard/tasks`: ringkasan kartu metrik, kartu kehadiran operasional, daftar task dengan badge status; menu "Tugas Harian" aktif
- E2E API (5 skenario: manager, superadmin, tanpa pilihan, dengan pilihan, 401) + E2E Playwright UI — semua lolos
- **Pkg 2** — endpoint multipart `POST /tasks/{id}/start|finish`: foto wajib (maks 3 MB, JPG/PNG/WEBP/GIF), maks 10 dokumen/tahap (10 MB, tipe dokumen umum), nilai field tambahan (`values` JSON + `value_files[field_name]` untuk tipe file), alokasi 7 role + self-assigned khusus manager KDKMP
- Validasi & guard: task aktif + role terpasang, pemilihan task (403 untuk task opsional yang belum dipilih), kehadiran KDKMP wajib tersimpan, alokasi ≤ sisa tersedia (lock FOR UPDATE), task selesai tidak bisa dimulai lagi, rollback file MinIO saat transaksi gagal
- Dokumen disimpan via `internal/taskreport` ke MinIO: `task-reports/{uuid}/{phase}/documents|additional-fields/{fieldUuid}/...`; daftar dokumen di-merge ke kolom `started_documents`/`finished_documents`
- Dialog UI Mulai/Selesaikan: unggah foto & dokumen, render field dinamis per tipe input (text/textarea/angka/tanggal/boolean/select/radio/checkbox/file), input alokasi dengan info "Tersedia"
- 🐞 Bug ditemukan user saat uji: validasi alokasi ikut berjalan pada mode **finish** → diperbaiki (alokasi hanya untuk start)
- **Pkg 3** — `GET /task-dashboard/completed`: riwayat 14 hari dikelompokkan per tanggal + ringkasan (total/tepat waktu/terlambat/tidak dikerjakan) + daftar laporan; paginasi 15 hari
- Endpoint pratinjau/unduh: dokumen (per fase & indeks), foto start/finish, file field tambahan — stream dari MinIO dengan disposisi inline/attachment; otorisasi via `CanViewTaskReport` (owner, superadmin, manager wilayah dengan cakupan regional)
- Halaman `/dashboard/tasks/completed` + tautan "Riwayat 14 hari" dari Tugas Harian
- SyncKdkmpActualRevenueAction (metrik KDKMP setelah finish) sengaja ditunda ke S5
- E2E API: riwayat, preview/download (termasuk 404 indeks/fase, 403 manager lain, 200 manager wilayah, 200 superadmin, 401) — semua lolos

**Review Sprint 4 (✅ goal tercapai):**
- Manager dapat melihat task harian sesuai role & pemilihan, memulai/menyelesaikan task dengan foto + dokumen + field dinamis + alokasi anggota, melihat riwayat 14 hari dan pratinjau/unduh berkas.
- Terverifikasi E2E API (~30 skenario) + E2E Playwright UI (mulai & selesaikan task lengkap, riwayat).

**Retro Sprint 4:**
- 🟢 Keep: pola paket + E2E browser menemukan bug nyata (validasi alokasi di mode finish) dan bug panic nil `ClockTime`
- 🟡 Improve: hati-hati dengan file chooser Playwright (chooser menggantung bisa memblok interaksi) — tunggu/handle chooser sebelum aksi berikutnya
- 🔵 Catatan: sinkronisasi metrik KDKMP (actual revenue/cost/scoring) dilakukan saat start/finish di app lama — ditunda ke S5 bersama `SyncKdkmpActualRevenueAction`

---

## Sprint 5 — Dashboard KDKMP ✅ (SELESAI)

**Goal:** Metrik harian, financial matrix, task selection, kehadiran, scoring, input harian.

**Status per package:**
- [x] **S5-1:** Metrik harian dari laporan task selesai: actual revenue/cost, durasi, completion, dan time compliance.
- [x] **S5-2:** Actual variable cost rolling 30 hari + financial matrix (grafik dan tabel) dengan fallback fixed cost legacy Rp9.235.467.
- [x] **S5-3:** Pilihan task opsional dan ekspansi bundle poin BMC; task in-progress tetap terkunci.
- [x] **S5-4:** Input kehadiran tujuh role operasional, dengan guard alokasi yang sudah berjalan.
- [x] **S5-5:** Upsert input harian, review plan revenue di bawah Rp20.000.000, performance scoring, dan sinkronisasi setelah task selesai.
- [x] **S5-6:** Financial Matrix memakai Recharts: biaya, biaya kumulatif, dan revenue rencana/realisasi; tabel tetap tersedia sebagai alternatif aksesibel.

**Catatan Sprint 5:**
- API manager-only: `GET /kdkmp-dashboard`, `GET /kdkmp-dashboard/input`, serta `PUT /kdkmp-dashboard/today`, `/today/task-selection`, dan `/today/operational-attendance`. Semua memakai gate ketat `role.domain=kdkmp` + `role.slug=manager`.
- Halaman web: `/dashboard/kdkmp` (ringkasan, matrix biaya, riwayat) dan `/dashboard/kdkmp/input` (target/biaya, kehadiran, pilihan BMC); dashboard generik mengarahkan Manager KDKMP ke halaman baru.
- Semua visualisasi chart pada frontend memakai Recharts; Financial Matrix menggunakan `ComposedChart` dengan token warna merah-putih aplikasi.
- Perhitungan mengikuti aplikasi Laravel: target Rp20.000.000, ambang variable cost token listrik Rp3.000.000 dan bahan bakar Rp2.000.000, margin memakai fixed cost Rp9.235.467, dan bobot skor 55/30/15.
- POS Revenue read-only tidak masuk Sprint 5; tercatat sebagai `D-1` setelah semua sprint selesai.

**Review Sprint 5 (✅ goal tercapai):**
- Alur Manager KDKMP kini lengkap dari menyimpan kehadiran dan pilihan BMC, memulai/menyelesaikan task, hingga data aktual tersinkron ke dashboard harian.
- Endpoint tercatat di Swagger; uji unit perhitungan, Go test/vet, serta lint dan production build frontend lulus.

**Retro Sprint 5:**
- 🟢 Keep: gunakan tabel legacy sebagai kontrak perilaku sehingga tidak perlu mengubah skema beku.
- 🟡 Improve: sebelum cutover, cocokkan nama task/field revenue dan biaya terhadap data produksi karena formula legacy mengandalkannya.

---

## Sprint 7 — Kolaborasi Manager ✅ (SELESAI)

**Goal:** Manager dapat mengelola Meeting Minutes, attachment, dan Action Item
miliknya melalui tabel meeting yang sudah ada.

**Backlog:** S7-1 … S7-5.

**Paket S7-1 sampai S7-3 (✅ selesai):**

- API Manager KDKMP untuk list, detail, buat, ubah, dan hapus Meeting Minutes
  miliknya sendiri dengan item sebagai snapshot berurutan.
- Item menyimpan PIC, rentang tenggat, status legacy, dan urutan tanpa owner
  selector atau akses Superadmin.
- Lampiran memakai MinIO dengan upload terpisah, metadata tabel existing, dan
  preview/download yang selalu memeriksa kepemilikan parent.

**Paket S7-4 dan S7-5 (✅ selesai):**

- Perubahan status Action Item disimpan atomik dengan `FOR UPDATE` dan riwayat
  aktor; perubahan status melalui edit snapshot Meeting Minutes juga tercatat.
- Halaman `/meeting-minutes` dan `/meeting-minutes/action-items` memakai SSR
  untuk akses awal serta TanStack Query untuk mutasi/filter tanpa dependensi baru.
- Navigasi dan endpoint hanya tersedia untuk Manager KDKMP; Superadmin dan
  Manager Wilayah tidak menerima menu maupun akses endpoint tersebut.
- Verifikasi runtime mencakup CRUD, preview lampiran, dua sumber perubahan
  status, overdue, riwayat, dan boundary API/SSR; seluruh data QA dihapus.

---

## Sprint 8 — Kesiapan Manager ✅ (SELESAI)

**Goal:** Tutup kesenjangan UX, navigasi, onboarding, dan validasi data
operasional Manager KDKMP sebelum cutover.

**Penyelesaian:**

- **S8-1:** `GET /auth/me` menyediakan metadata SK hanya untuk Manager KDKMP;
  profil memakai preview endpoint terotorisasi yang sudah ada.
- **S8-2:** Dashboard memiliki tautan cepat Input/Tugas/Meeting. Tur versi 2
  melintasi dashboard, matrix, input, task, dan meeting; manager lama melihat
  pembaruan ini satu kali per browser tanpa perubahan schema.
- **S8-3:** Tabel utama dapat digeser di layar kecil, pencarian meeting/action
  item responsif, empty state memiliki tindakan pemulihan, dan error boundary
  aplikasi menyediakan retry.
- **S8-4:** Audit read-only mapping revenue/biaya didokumentasikan pada
  `docs/S8_MAPPING_AUDIT.md`; alias legacy tidak dimasukkan ke formula.

**Verifikasi:** `go test -mod=readonly ./...`, `go vet ./...`, `npm run lint`,
dan `npm run build` lulus. API dan SSR diuji dengan akun Manager KDKMP serta
Superadmin; browser automation tidak tersedia pada lingkungan kerja.

Integrasi LMS, Lumbung, dan Lark tetap `blocked` hingga kontrak eksternal
tersedia dan tidak menjadi syarat selesai sprint.

---

## Sprint 9 — Paritas & Cutover Manager ✅ (REHEARSAL LOKAL SELESAI)

**Goal:** Audit paritas alur Manager KDKMP, migrasi data tabel in-scope,
hardening, backup/rollback, dan cutover.

**Penyelesaian paket 1–3:**

- **S9-1:** Audit route, halaman, dan gate ada pada
  `docs/S9_PARITY_AUDIT.md`. Endpoint task yang sebelumnya masih memakai gate
  level umum kini hanya untuk Manager KDKMP; menu Manager Wilayah diselaraskan.
- **S9-2:** `cmd/migrate-legacy-kdkmp` menyediakan preflight read-only dan
  apply transaksional yang menolak target scope tidak kosong atau metadata
  berkas tanpa objek. Hasil dan prosedur cutover ada pada
  `docs/S9_DATA_MIGRATION.md`.
- **S9-3:** Sesi diganti saat login, challenge 2FA, dan login passkey; ganti
  kata sandi mencabut seluruh sesi aktif akun lalu membuat sesi baru. Header
  keamanan dasar dipusatkan di middleware; pencarian Meeting dibatasi 255
  karakter; audit indeks dan pagination didokumentasikan.

**Penyelesaian paket 4–5 (rehearsal lokal):**

- **S9-4:** `scripts/s9-rehearsal.sh` hanya menerima database Docker
  `ebitdamax_apn_rehearsal`; mode `--reset` membuat backup logical sebelum
  rebuild. Migrator menolak target lain, menjalankan transaksi relasional
  tanpa metadata berkas, lalu mengganti satu Manager lengkap menjadi akun QA
  lokal. Rehearsal berhasil dibangun ulang dari backup pada 25 September 2026.
- **S9-5:** Runbook, prosedur rollback, pemeriksaan layanan, dan bukti
  verifikasi tersedia di `docs/S9_REHEARSAL_RUNBOOK.md`,
  `docs/S9_OPERATIONS.md`, dan `docs/S9_REHEARSAL_REPORT.md`.

**Batas penutupan:** Sprint ini selesai sebagai **rehearsal lokal** sesuai
keputusan scope. Ini bukan clone penuh atau cutover produksi: legacy hanya
dibaca, database refactor utama tidak disentuh, dan objek MinIO legacy tidak
disalin. Cutover produksi memerlukan persetujuan baru serta salin dan validasi
artefak MinIO terlebih dahulu.

---

## Sprint 10 — Customer Insight Manager ✅

**Goal:** Manager KDKMP mencatat dan mengelola hasil wawancara pelanggan
secara mandiri.

- **S10-1:** `migrations/00002_customer_analyses.sql` menambahkan tabel dan
  indeks kepemilikan user; baseline rehearsal menerapkannya saat rebuild.
- **S10-2:** API `GET/POST/PUT /customer-analyses` memakai gate Manager KDKMP
  dan query owner. Validasi mengikuti enum, batas usia, serta batas teks
  legacy.
- **S10-3:** Halaman Customer Analysis menyediakan kartu responsif, detail,
  tambah, dan edit. Riwayat legacy tidak diimpor; fitur dimulai dengan data
  kosong sesuai keputusan produk.

## Sprint 11 — Komunikasi Manager ✅

**Goal:** Superadmin dapat mengirim pengumuman kepada Manager KDKMP, dan
Manager dapat membaca serta menandai notifikasinya.

- **S11-1:** `POST /announcements` hanya untuk Superadmin dan membuat
  notifikasi seluruh Manager KDKMP dalam satu transaksi.
- **S11-2:** `GET /notifications`, `PATCH /notifications/{id}/read`, dan
  `PATCH /notifications/read-all` dibatasi identitas penerima; tabel baseline
  `notifications` dipakai tanpa schema baru.
- **S11-3:** Pengumuman Superadmin, halaman Notifikasi Manager, unread badge,
  dan bell header aktif. Riwayat notifikasi legacy sengaja tidak dimigrasikan.

## Sprint 12 — POS Revenue Read-only ⛔

Sprint 12 belum dimulai. Ia memerlukan validasi URL, kredensial, format
respons, serta pemetaan NIK/companyId dari API POS sebelum adapter Go dapat
dibangun. Input pendapatan manual tidak dipakai sebagai pengganti.

---

## Sprint 13 — Auth Token (JWT) ✅ (SELESAI)

**Goal:** Ganti session Redis berstate dengan JWT: access token stateless (cookie HttpOnly untuk web,
`Authorization: Bearer` untuk API client/Swagger) + refresh token opaque yang dapat dicabut.

**Keputusan (disetujui user):** hybrid cookie+Bearer · access **1 jam** · refresh **7 hari rotating** ·
revoke refresh saja (tanpa denylist access) · proxy Next verifikasi signature+exp · flow Bearer di Swagger ·
**switch langsung tanpa dual-read legacy** · tanpa perubahan skema DB (Redis + config saja).

**Status per package:**
- [x] **Pkg 1:** `internal/token` (HS256, claims sub/sid/jti/iss/iat/exp, leeway 60s) + env JWT + login/2FA/passkey/ganti-password menerbitkan access token + middleware verifikasi JWT (cookie & Bearer)
- [x] **Pkg 2:** Refresh store (rotasi single-use, tombstones replay, index `user_refresh:{id}` revoke-all O(1)) + `/auth/refresh` + `/auth/logout-all` + auto-refresh transparan di middleware
- [x] **Pkg 3:** `POST /auth/token` (+`/auth/token/verify` 2FA) untuk klien Bearer + Swagger
- [x] **Pkg 4:** Web (proxy verifikasi Edge via Web Crypto, retry 401 sekali, E2E Playwright) + dokumen

**Catatan teknis Pkg 1:**
- Cookie baru `ebitda_access` (JWT, HttpOnly, SameSite=Lax, Secure saat production); `ebitda_refresh` menyusul di Pkg 2
- Middleware memverifikasi signature+issuer+exp (stateless, tanpa Redis) lalu tetap memuat user dari DB agar role/permission selalu segar
- Session manager lama dirampingkan (hanya menyimpan state sementara 2FA/passkey); `DestroyUserSessions` lama dihapus — pencabutan sesi lain saat ganti password **sementara nonaktif** dan akan kembali di Pkg 2 via index refresh O(1)
- Test: `internal/token` (6 kasus: valid, expired, tamper, secret salah, alg salah, issuer salah) + test hardening server diadaptasi; E2E: login→cookie JWT (3 bagian), /auth/me (cookie & Bearer), tamper 401, expired 401 "Sesi sudah berakhir", logout clear cookie, web SSR tetap jalan (`/dashboard` → final 200)
- Swagger: `CookieAuth` pindah ke `ebitda_access` + securityDefinition `BearerAuth` disiapkan untuk Pkg 3

**Catatan teknis Pkg 2:**
- `internal/session/refresh.go`: `RefreshStore` — key `refresh:{token}` (payload user/family/created_at, TTL 7 hari sliding), index `user_refresh:{userID}` (SET) untuk **revoke-all O(jumlah sesi user)**, tombstone `refresh_used:{token}` (TTL 10 menit) untuk deteksi replay
- `Rotate` (single-use, hanya di endpoint eksplisit `/auth/refresh`): token lama dihapus + ditombstone → token baru family sama; replay token tertombstone → **seluruh sesi user dicabut** + pesan khusus
- **Deviasi desain yang disengaja:** auto-refresh di middleware hanya `Get + Touch` (sliding TTL) **tanpa rotasi** — rotasi di middleware memicu false-positive replay saat halaman menembak banyak request paralel dengan access token yang sama-sama kedaluwarsa (dan Set-Cookie dari refresh sisi RSC tidak sampai ke browser). Rotasi/replay-detection tetap berjalan untuk klien yang memanggil `/auth/refresh` eksplisit
- Cookie `ebitda_refresh` (HttpOnly, SameSite=Lax, Secure saat production) diterbitkan bersama access di login/2FA/passkey; `logout` mencabut refresh + clear kedua cookie; `logout-all` mencabut seluruh sesi user
- Ganti password: `RevokeUser` mencabut semua refresh token lalu menerbitkan sesi baru untuk perangkat ini (fitur pencabutan sesi lain pulih dari Pkg 1)
- Test unit refresh store (lifecycle, rotasi, replay+revoke-all, revoke user terisolasi, touch TTL) + test hardening server diperluas (2 cookie, family id terikat claims)
- E2E: login 2 cookie & key Redis; auto-refresh middleware (access expired + refresh valid → 200 + access baru); `/auth/refresh` merotasi; replay → 401 + semua sesi dicabut; logout & logout-all mencabut refresh; ganti password mencabut sesi lain (kredensial dev direvert)

**Catatan teknis Pkg 3:**
- `POST /auth/token` (JSON): login klien non-browser → `{token_type: Bearer, access_token, expires_in, refresh_token}`; bila 2FA aktif → `202 {two_factor_required, challenge_token}`
- `POST /auth/token/verify`: verifikasi TOTP/recovery code dengan challenge dari body (bukan cookie) → token pair; batas percobaan tetap 5 (429)
- Logika kredensial diekstrak ke `findUserByCredentials` (dipakai login cookie & API); helper `issueTokenPair` (tanpa cookie) vs `issueAuthSession` (set cookie)
- Swagger: securityDefinition `BearerAuth` + tombol Authorize; E2E: token, Bearer /auth/me, kredensial salah 401, refresh via body, logout via body, 2FA penuh (enable → 202 → kode salah 422 → TOTP 200 → recovery code 200 → disable via Bearer, state direvert)

**Catatan teknis Pkg 4:**
- `proxy.ts` memverifikasi HS256 (Web Crypto `crypto.subtle`), issuer, dan exp di Edge; token invalid → redirect + hapus cookie; **expired + refresh cookie ada → diloloskan** (API akan memperbarui, termasuk untuk RSC yang tidak bisa menulis cookie browser)
- `apiFetch` retry sekali via `POST /auth/refresh` saat 401 (single-flight, dikecualikan untuk path `/auth/*`)
- Env web baru: `JWT_SECRET` (server-only, harus sama dengan API; **di-inline saat build** oleh Edge runtime — set juga di environment build produksi)
- E2E Playwright: login → dashboard; cookie di-tamper → redirect login; signature forged → redirect; access expired + refresh valid → halaman tetap tampil & `GET /auth/me` sisi klien memperbarui cookie access; logout UI → redirect + refresh dicabut (401) + cookie bersih

**Review Sprint 13 (✅ goal tercapai):**
- Auth berpindah dari session Redis stateful ke JWT access (1 jam) + refresh (7 hari, revocable, rotasi single-use dengan deteksi replay) dengan dukungan cookie (web) dan Bearer (klien API/Swagger).
- Pencabutan: logout, logout-all, dan ganti password (revoke-all O(1) via index `user_refresh:{id}`).
- Verifikasi: 17+ skenario E2E API + 6 skenario Playwright; unit test token & refresh store hijau.

**Retro Sprint 13:**
- 🟢 Keep: deviasi desain didokumentasikan (rotasi hanya di endpoint eksplisit) mencegah false-positive replay pada request paralel; proteksi cookie HttpOnly + proxy verify berlapis
- 🟡 Improve: `JWT_SECRET` web di-inline saat build Edge — tambahkan ke checklist deployment saat cutover S9
- 🔵 Catatan: access token tidak dapat dicabut sebelum kedaluwarsa (≤1 jam) sesuai keputusan "revoke refresh saja"; denylist `jti` dapat ditambahkan bila nanti dibutuhkan

**Catatan teknis Pkg 5 (Addendum — unifikasi login & token):**
- Arahan user: "login adalah momen mendapatkan token" — endpoint ganda dihapus.
- `POST /auth/login` kini mengembalikan data user **+ `{token_type, access_token, expires_in, refresh_token}`** di body sekaligus menyetel cookie HttpOnly `ebitda_access`/`ebitda_refresh`; cabang 2FA mengembalikan `challenge_token` di body (cookie `ebitda_2fa` tetap).
- `POST /auth/two-factor-challenge` menerima `challenge_token` dari body **atau fallback cookie**; sukses → token pair (body + cookie). `POST /auth/passkey/login` juga mengembalikan token pair di body.
- **Breaking (aman, tanpa konsumen):** `POST /auth/token` & `POST /auth/token/verify` dihapus; `api_token_handlers.go` dihapus; helper baru `applyAuthCookies` + `authResponseWithTokens`.
- Verifikasi: login (token body + 2 cookie, Bearer /me 200, kredensial salah 401, `/auth/token` 404), 2FA jalur body & cookie → token, recovery code via body, passkey login (virtual authenticator) → `access_token` di body + Bearer /me 200, web SSR login → dashboard 200; `go build/vet/test` + swagger (61 path) hijau.

---

## Sprint 14 — Monitoring Dashboard KDKMP (SELESAI)

**Goal:** Superadmin & manager wilayah dapat memonitor seluruh manager KDKMP lewat hierarki wilayah
("pohon EBITDA"): drill-down nasional → provinsi → kabupaten → kecamatan → desa, ringkasan revenue/gap,
chart bulanan, tabel rincian harian, dan detail task per KDKMP.

**Keputusan (disetujui user):** parity pola lama · akses superadmin (nasional) + manager-wilayah (scoped,
locked filters per-field) · tanpa perubahan skema DB.

**Status per package:**
- [x] **Pkg 1:** Regional access + konsolidasi — `internal/kdkmp/regional.go` (ManagedKdkmpQuery, AccessibleManagedKdkmpQuery, RegionOptions/AllRegionOptions, FilterContext + locked filters + scope label) & `internal/kdkmp/consolidation.go` (ConsolidateEntries per level, natural sort, gap)
- [x] **Pkg 2:** Bulk metrics (`MetricsForUsers`) + monthly financial matrix (titik harian + kumulatif)
- [x] **Pkg 3:** Endpoint `GET /admin/kdkmp-dashboard` + halaman monitoring
- [x] **Pkg 4:** Detail task per KDKMP/tanggal + tombol "Lihat Task"

**Catatan teknis Pkg 1:**
- `access.go` di-refactor memakai `accessibleScopeConditions` bersama (dipakai juga oleh CanViewTaskReport) — menghapus duplikasi logika scope
- Kontrak disamakan dengan aplikasi lama: key/label per level (national→village), `plan/actual/gap` nullable (null bila tidak ada nilai numerik), `complete_kdkmp` = jumlah record harian pada tanggal, metadata wilayah dari entry pertama grup, urutan natural case-insensitive (fungsi `naturalLess` sendiri, tanpa dependensi baru)
- Filter wilayah exact-match (`regionFilterConditions`), locked filter hanya bila 1 nilai unik (mirror `filterContext`)
- Test: 8 unit test baru (locked filters, scope label, kondisi scope akses, kondisi filter wilayah, konsolidasi nasional/provinsi, gap null, natural sort) — `go test ./...` 11 paket hijau
- Catatan: verifikasi perakitan query GORM dilakukan di E2E Pkg 3 (DryRun GORM postgres tetap butuh koneksi), fragmen SQL sudah teruji murni

**Catatan teknis Pkg 2:**
- `internal/kdkmp/metrics.go` (baru): `MetricsForUsers(ctx, db, userIDs, date)` — satu query user+role, satu query task per role (`tasksByRoleForRoles`, dipakai juga monthly matrix), satu query pilihan task per entry (`DailySelectedTaskIDsByKdkmpEntryAndDate`), satu query laporan selesai, satu query once-completed, satu query nilai expense/revenue; `computeMetrics` murni untuk matematika (durasi, completion, compliance, format) — single `MetricsForUser` kini wrapper bulk (perilaku lama dipertahankan, `dailyRevenueAndCost` dihapus)
- `internal/kdkmp/monthly_matrix.go` (baru): `MonthlyFinancialMatrixForEntry` (manager = user pertama per entry, `has_data` false bila tidak ada) — parity `KdkmpMonthlyFinancialMatrixService`: task eksekusi = wajib/terpilih per tanggal, fixed cost default bila tidak semua task terkonfigurasi, actual cost hanya bila durasi aktual > 0 (+ overage variable cost record), revenue dari record harian, kumulatif running sum di-round per titik; helper murni `datesBetween`, `monthlyExecutionTasks`, `monthlyFixedCost`, `monthlyCosts`
- `selection.go`: key pilihan per entry+tanggal diekstrak ke `entryDateKey` (dipakai bersama metrik)
- Test: 8 subtest baru (`metrics_test.go`, `monthly_matrix_test.go`) — `go test ./...` 14 paket hijau; verifikasi orkestrasi query menyusul di E2E Pkg 3

**Catatan teknis Pkg 3:**
- Endpoint `GET /api/v1/admin/kdkmp-dashboard` (`internal/server/kdkmp_monitoring_handlers.go`): middleware baru `RequireMonitoringAccess` (superadmin + manager wilayah saja — manager KDKMP biasa 403, berbeda dari policy lama yang juga mengizinkan manager/ebitda_kdkmp, sesuai keputusan sprint); validasi 422 (bulan ≤ bulan berjalan, status/level dikenal, detail_date ≤ hari ini, panjang filter ≤255); kontrak mirror lama: `entries` (paginated 25 + search/status), `summary` (total/complete/not_filled/requires_review), `filters` (+ locked region merge), `region_options`, `regional_access`, `consolidation`, `selected_kdkmp` (otomatis saat desa terfilter/terkunci), `monthly_financial_matrix`; transformEntry parity (8 kolom harian, `variable_cost` dari `plan_cost`, margin & scoring dihitung ulang dari record + metrik)
- Swagger di-regenerate (`swag init -g main.go -o docs --parseDependency --parseInternal`)
- Web: halaman `/admin/kdkmp-dashboard` (server fetch + react-query, pola users-table) — `src/components/kdkmp-monitoring/monitoring-dashboard.tsx` (ringkasan, pohon EBITDA breadcrumb + kartu wilayah, filter wilayah terkunci, tabel 8 kolom + status + paginasi) & `monthly-financial-matrix-chart.tsx` (recharts, token chart, klik tanggal + select tanggal aksesibel); tipe di `src/types/kdkmp-monitoring.ts`; menu sidebar "Monitoring KDKMP" → ready
- Kolom "Aksi / Lihat Task" sengaja belum ada (Pkg 4)
- E2E: superadmin (nasional → provinsi → kabupaten → kecamatan → desa, chart 26 titik, klik tanggal 23 Sep → tabel + badge "Lengkap"/"Review Plan Revenue", filter status/search, paginasi) & manager wilayah (scope "Wilayah penugasan", 4 filter terkunci, level dipaksa provinsi, chart otomatis) & manager KDKMP (redirect keluar); validasi API via curl (422 semua kasus); light/dark mode + 360px dicek; console browser bersih
- Test: 7 subtest baru `kdkmp_monitoring_handlers_test.go` (parse params + helper, tanpa DB) — `go test ./...` 14 paket hijau; build web hijau
- Catatan: kunjungan dashboard manager saat E2E memicu `SyncDailyMetrics` (perilaku normal aplikasi) sehingga record 26 Sep ter-update (scoring 2.5%, durasi 1 menit)

**Catatan teknis Pkg 4:**
- Endpoint `GET /api/v1/admin/kdkmp-dashboard/:entryID/tasks/:date` (`internal/server/kdkmp_monitoring_task_handlers.go`): validasi tanggal (422), entry harus dalam cakupan akses (404), laporan = milik manager entry, status completed, `period_key = date OR finished_at/started_at dalam rentang hari bisnis`, urut `finished_at DESC` (parity lama); payload laporan: foto (start/finish), dokumen (index per fase), nilai field tambahan (sortir `show_when, sort_order, id` — "finish" mendahului "start" sesuai sortBy lama), task + kategori + roles
- **Perubahan rute berkas**: 6 endpoint `task-reports/{id}/photos|documents|additional-fields/.../preview|download` dipindah dari grup `RequireKdkmpManager` ke grup authenticated biasa — otorisasi per laporan tetap via `CanViewTaskReport` (manager pemilik, manager wilayah dalam cakupan, superadmin); ini juga membuka akses berkas untuk superadmin/manager wilayah
- Web: halaman `/admin/kdkmp-dashboard/[entryID]/tasks/[date]` (server fetch, 404→notFound, 403→redirect) + `src/components/kdkmp-monitoring/task-reports-view.tsx` (tabel + dialog detail: waktu, foto dengan preview/unduh, data laporan termasuk berkas, dokumen); tombol "Lihat Task" di tabel monitoring (disabled tanpa manager; toast "Task belum selesai semua atau belum ada." bila completion < 100; dibuka di tab baru agar state drill-down monitoring tidak hilang — deviasi kecil dari lama yang same-tab)
- E2E: matriks otorisasi berkas diuji nyata dengan fixture sementara (entry Papua + manager + manager wilayah Papua + laporan berfoto via API): superadmin/JT MW/pemilik → **200 image/jpeg** (160B) + header attachment; Papua MW & manager entry lain → **403**; entry di luar cakupan → 404; tasks endpoint superadmin/JT MW → 200, Papua MW → 404, manager KDKMP → 403 (middleware); UI: halaman detail task render 2 laporan, dialog memuat 2 foto (naturalWidth > 0), 2 nilai field, empty state dokumen; toast guard "Lihat Task" terverifikasi; console bersih. Seluruh fixture E2E dihapus (users/entry/report/MinIO object/attendance record) — DB kembali ke 2 user/1 entry/62 report
- Test: 5 unit test baru `kdkmp_monitoring_task_handlers_test.go` (payload foto/dokumen/nilai/file, label fase) — `go test ./...` 14 paket hijau; build web hijau; swagger di-regenerate
