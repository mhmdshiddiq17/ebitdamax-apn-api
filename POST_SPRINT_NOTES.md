# Catatan Pasca-Sprint — Paritas Manager KDKMP

Dokumen ini mencatat temuan dari perbandingan refactor Go/Next.js dengan
`../ebitdamax-apn` (Laravel/Inertia). Catatan ini **bukan** perubahan scope atau
status backlog. Setelah sprint aktif selesai, setiap item yang disepakati harus
dipindahkan ke `BACKLOG.md` sebelum dikerjakan.

## Batas Acuan

- Aplikasi Laravel adalah sumber perilaku untuk fitur Manager KDKMP.
- Refactor menggunakan skema PostgreSQL yang sama, tetapi paritas skema tidak
  berarti semua perilaku aplikasi lama sudah berpindah.
- Identitas Manager KDKMP yang sah adalah `roles.domain = kdkmp` dan
  `roles.slug = manager`.
- `manager-wilayah` adalah peran terpisah. Cakupan Manager KDKMP berasal dari
  `users.sdm_kdkmp_entry_id`.

## Temuan yang Terselesaikan di Sprint 5

- Scope telah ditegaskan: workflow dashboard harian Manager KDKMP
  (`ebitdamax_kdkmp`) masuk scope; modul APN corporate tetap di luar scope.
- Prasyarat task kini dapat disimpan dari web: kehadiran operasional dan pilihan
  task per bundle BMC, dengan task in-progress tetap terkunci.
- Endpoint dashboard memakai gate ketat `roles.domain = kdkmp` dan
  `roles.slug = manager`; navigasi web menggunakan kondisi yang sama.
- Finish task menyinkronkan actual revenue/cost, variable cost, durasi, margin,
  dan scoring ke data harian KDKMP.

## Temuan yang Perlu Ditindaklanjuti

### P1 — Verifikasi referensi formula pada data produksi

Formula legacy masih mengenali revenue dan biaya dengan nama task/field. Sebelum
cutover, cocokkan `Penyetoran Struk dan Uang.rekonsiliasi_uang_masuk` serta
`Pencatatan Pengeluaran Operasional Harian` dengan data produksi agar perubahan
master data tidak mengubah hasil perhitungan diam-diam.

### P1 — Putuskan status Customer Analysis

Sidebar Manager KDKMP di aplikasi lama menyediakan `Customer Analysis`, tetapi
fitur ini belum tercatat dalam Sprint 5–8. Putuskan salah satu:

- masukkan sebagai fitur parity Manager KDKMP pada backlog; atau
- nyatakan eksplisit sebagai fitur yang tidak ikut direfactor.

### P2 — Tunda integrasi pelengkap sampai alur inti stabil

LMS KDKMP, Lumbung Chat, Meeting Minutes, dan Lark SSO sudah ada di aplikasi
lama, tetapi masuk Sprint 8 pada refactor. Lark juga membutuhkan kolom dan alur
migrasi tersendiri. Jangan menggabungkannya dengan Sprint 5 karena tidak
diperlukan untuk menutup alur task dan dashboard inti.

## Urutan Rekomendasi Setelah Sprint Aktif

1. Verifikasi referensi formula terhadap data produksi tanpa mengubah data.
2. Putuskan apakah Customer Analysis ikut parity atau eksplisit dikecualikan.
3. Audit akses Manager KDKMP, Manager Wilayah, dan Superadmin sebelum membuka
   monitoring Sprint 6.
4. POS Revenue tetap ditunda sebagai `D-1` di `BACKLOG.md` sampai semua sprint
   selesai, sesuai arahan user.

## Referensi Kode

- Legacy role dan cakupan: `../ebitdamax-apn/app/Models/User.php`,
  `../ebitdamax-apn/app/Policies/EbitdamaxKdkmpPolicy.php`
- Legacy dashboard dan task: `../ebitdamax-apn/app/Http/Controllers/KdkmpDashboardController.php`,
  `../ebitdamax-apn/app/Http/Controllers/TaskReportController.php`
- Refactor route saat ini: `internal/server/router.go`
- Refactor task dashboard: `internal/server/task_dashboard_handlers.go`
