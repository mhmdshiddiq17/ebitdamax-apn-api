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

## Temuan yang Perlu Ditindaklanjuti

### P0 — Selaraskan keputusan scope Dashboard KDKMP

`BACKLOG.md` menyebut workflow `ebitda_kdkmp` di luar scope, tetapi Sprint 5
merencanakan dashboard harian KDKMP yang memakai `ebitdamax_kdkmp`. Tegaskan
apakah alur harian Manager KDKMP memang bagian dari refactor. Jangan mulai Sprint
5 sebelum istilah dan batas scope ini konsisten.

### P0 — Lengkapi prasyarat eksekusi task Manager KDKMP

Sprint 4 sudah memeriksa kehadiran operasional dan pemilihan task ketika manager
memulai task. Namun endpoint dan halaman untuk menyimpan keduanya belum ada di
router refactor. Tanpa data yang dibuat manual, Manager KDKMP tidak dapat
menyelesaikan alur task mandiri.

Tindak lanjut Sprint 5:

1. Simpan kehadiran operasional harian dengan transaksi dan penguncian baris.
2. Simpan pemilihan task harian serta ekspansi bundle BMC.
3. Pertahankan task wajib dan task yang sedang dikerjakan sebagai pilihan yang
   tidak dapat dilepas.
4. Baru buka alur start task sebagai perjalanan pengguna yang lengkap.

### P0 — Ketatkan otorisasi berdasarkan domain dan slug

Legacy membedakan Manager KDKMP dengan kombinasi domain dan slug. Refactor masih
memakai `RequireLevels` pada grup task, sementara navigasi web mengenali manager
hanya dari slug. Role APN dengan level atau slug yang sama berpotensi melihat
menu atau mengakses endpoint yang bukan scope-nya.

Tindak lanjut:

- Gunakan gate bersama untuk Manager KDKMP: `domain=kdkmp` + `slug=manager`.
- Tetapkan jalur Manager Wilayah dan Superadmin secara eksplisit, bukan sebagai
  efek samping dari level `manager`.
- Sinkronkan gate backend dengan kondisi navigasi frontend.

### P1 — Porting Dashboard KDKMP harus mengikuti sumber data lama

Alur Laravel untuk dashboard harian adalah:

`task_reports` + `task_report_values` → metrik harian → `ebitdamax_kdkmp` →
financial matrix dan performance scoring.

Finish task di Laravel ikut menyinkronkan actual revenue. Refactor secara sadar
menundanya ke Sprint 5; jangan menganggap data dashboard sudah otomatis berubah
setelah task diselesaikan sebelum sinkronisasi tersebut tersedia.

Saat memindahkan metrik, verifikasi referensi task dan field dari data produksi.
Aplikasi lama masih mengenali beberapa nilai melalui nama task/field, sehingga
perubahan seed atau label dapat mengubah hasil perhitungan.

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

1. Selesaikan dan review Sprint 4 yang sedang aktif.
2. Selesaikan keputusan scope pada temuan P0 pertama.
3. Kerjakan Sprint 5 dengan urutan: kehadiran → pemilihan task/BMC → input
   harian → metrik/scoring → financial matrix.
4. Audit ulang akses Manager KDKMP, Manager Wilayah, dan Superadmin sebelum
   membuka monitoring Sprint 6.
5. Perbarui `BACKLOG.md` dan `SPRINT.md` hanya setelah setiap keputusan scope
   disetujui.

## Referensi Kode

- Legacy role dan cakupan: `../ebitdamax-apn/app/Models/User.php`,
  `../ebitdamax-apn/app/Policies/EbitdamaxKdkmpPolicy.php`
- Legacy dashboard dan task: `../ebitdamax-apn/app/Http/Controllers/KdkmpDashboardController.php`,
  `../ebitdamax-apn/app/Http/Controllers/TaskReportController.php`
- Refactor route saat ini: `internal/server/router.go`
- Refactor task dashboard: `internal/server/task_dashboard_handlers.go`
