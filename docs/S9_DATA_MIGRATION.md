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

## Prosedur cutover (Sprint 9-4)

1. Simpan backup source dan target, lalu salin/validasi seluruh objek MinIO
   yang tercatat preflight.
2. Pastikan seluruh tabel scope target kosong; akun/role seed boleh tetap ada.
3. Jalankan kembali preflight dan cocokkan jumlahnya.
4. Jalankan `go run ./cmd/migrate-legacy-kdkmp --apply` terhadap target
   cutover yang sudah disetujui.
5. Cocokkan jumlah target yang dicetak migrator, login satu Manager hasil
   migrasi, lalu cek dashboard, task, dan meeting miliknya.

Penulisan target berjalan dalam satu transaksi. Jika relasi atau hitungan
akhir tidak cocok, transaksi dibatalkan.
