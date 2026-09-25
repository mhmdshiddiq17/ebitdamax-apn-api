# Sprint 9 — Rehearsal Cutover Lokal

Rehearsal ini **bukan** clone penuh `ebitdamax-apn`. Legacy hanya dibaca dan
database refactor utama tidak disentuh. Target tunggalnya adalah database Docker
`ebitdamax_apn_rehearsal`.

## Data yang dibawa

- Seluruh data relasional Manager KDKMP yang sudah berada di scope refactor:
  manager, KDKMP, task, laporan, dashboard harian, Meeting Minutes, item, dan
  riwayat status.
- SK Manager, foto/dokumen laporan, nilai field file, serta lampiran meeting
  tidak dibawa dan metadatanya dibersihkan.
- Satu Manager hasil migrasi dengan dashboard, laporan, dan meeting paling
  lengkap diubah menjadi akun QA lokal. Perubahan identitas ini hanya ada di
  database rehearsal agar relasi data nyata dapat diuji tanpa kredensial legacy.

## Jalankan rehearsal

1. Pastikan Docker Desktop aktif dan `.env` berisi koneksi legacy baca-saja.
2. Jalankan `bash scripts/s9-rehearsal.sh`.
3. Bila database rehearsal lama perlu dibuat ulang, jalankan
   `bash scripts/s9-rehearsal.sh --reset`. Script membuat backup logical lama
   di `tmp/s9-rehearsal/<timestamp>/` sebelum menghapus database tersebut.
4. Jalankan API dengan environment terisolasi:

   ```sh
   DB_NAME=ebitdamax_apn_rehearsal APP_PORT=4001 \
   SESSION_COOKIE=ebitda_rehearsal_session \
   CORS_ALLOWED_ORIGINS=http://localhost:3001 \
   MINIO_BUCKET=ebitdamax-rehearsal go run .
   ```

5. Jalankan frontend dengan API rehearsal:

   ```sh
   NEXT_PUBLIC_API_URL=http://localhost:4001/api/v1 \
   NEXT_PUBLIC_SESSION_COOKIE=ebitda_rehearsal_session PORT=3001 npm run dev
   ```

## Verifikasi dan rollback

- Konfirmasi output migrator dan pastikan metadata berkas berjumlah nol.
- Login akun QA lokal, lalu buka dashboard KDKMP, task, riwayat, Meeting
  Minutes, dan Action Items. Preview/download berkas harus tidak tersedia.
- Pastikan Superadmin menerima 403 saat memanggil endpoint manager-only.
- Rollback initial rehearsal cukup dengan menghapus database
  `ebitdamax_apn_rehearsal`; database legacy dan refactor utama tetap utuh.
- Untuk hasil `--reset`, buat kembali database kosong lalu restore dump terakhir
  menggunakan `pg_restore` dari folder backup yang dicetak script.
