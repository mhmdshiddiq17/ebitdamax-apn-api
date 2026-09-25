# Sprint 9 — Migrasi Data Manager KDKMP

Perintah `go run ./cmd/migrate-legacy-kdkmp` membaca database legacy dan
menghasilkan preflight **tanpa menulis target**. Gunakan kredensial legacy
melalui variabel lingkungan `LEGACY_DB_*`; jangan simpan kredensial di dokumen
atau command history.

## Scope

Migrasi hanya membawa data Manager KDKMP dan relasi yang dipakai alur manager:

- akun Manager KDKMP dan `sdm_kdkmp_entries` yang terhubung;
- kategori, task Manager, pivot role, field, laporan, nilai laporan, dan
  `ebitdamax_kdkmp`;
- Meeting Minutes, item, dan riwayat status.

Role Manager KDKMP dipakai jika sudah ada pada target. Akun target dengan
email sama hanya boleh merupakan Manager KDKMP; akun atau tabel scope yang
sudah berisi data tidak ditimpa.

Secret 2FA, passkey, sesi, dan metadata dokumen tidak dibawa. Pengguna perlu
enroll ulang faktor keamanan setelah cutover.

## Hasil preflight lokal

Preflight 25 September 2026 menemukan 2.009 Manager, 2.008 entry KDKMP, 6
kategori, 43 task, 127 laporan, 36 dashboard harian, 2 meeting, 2 item, dan 1
riwayat status. Ia juga menemukan objek eksternal: 1 SK, 247 foto, 5 dokumen
task, dan 1 lampiran meeting.

Karena metadata tanpa objek akan menghasilkan tautan rusak, `--apply` ditolak
selama referensi objek tersebut masih ada. Guard ini mencegah kehilangan data.

## Rehearsal Sprint 9-4

Rehearsal memakai `bash scripts/s9-rehearsal.sh`, database khusus
`ebitdamax_apn_rehearsal`, dan mode
`go run ./cmd/migrate-legacy-kdkmp --apply --rehearsal --without-files`.
Kedua flag tersebut wajib dipakai bersama dan ditolak untuk database lain.

Mode rehearsal membersihkan metadata file, memilih satu Manager hasil migrasi
sebagai akun QA lokal, dan tetap menulis semua relasi non-file dalam satu
transaksi. Runbook lengkap ada pada `docs/S9_REHEARSAL_RUNBOOK.md`.

Cutover produksi tetap membutuhkan salin/validasi seluruh objek MinIO sebelum
`--apply` normal dapat digunakan. Itu bukan bagian rehearsal ini.
