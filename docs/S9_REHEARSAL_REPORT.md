# Sprint 9 — Laporan Rehearsal Lokal

Dilaksanakan pada 25 September 2026 terhadap database Docker
`ebitdamax_apn_rehearsal`. Legacy hanya dibaca dan database refactor utama
tidak dijadikan target perintah.

## Hasil rebuild

- `scripts/s9-rehearsal.sh --reset` membuat backup logical sebelum target
  rehearsal dibangun ulang.
- Migrasi relasional selesai: 2.009 Manager KDKMP, 2.008 entry KDKMP, 43 task,
  127 laporan, 36 dashboard, dan 2 Meeting Minutes berada dalam scope target.
- Akun QA lokal memiliki 14 dashboard, 49 laporan, dan 1 Meeting Minute.
- Metadata SK, foto/dokumen task, field file, dan attachment Meeting Minutes
  berjumlah 0.

## Verifikasi runtime

- `GET /healthz` mengembalikan 200.
- Login akun QA serta endpoint dashboard KDKMP, task, riwayat, Meeting
  Minutes, dan Action Items mengembalikan 200.
- Akun Superadmin menerima 403 pada endpoint task dashboard manager.
- Render SSR halaman dashboard KDKMP, task, dan Meeting Minutes melalui web
  rehearsal mengembalikan 200.

Tidak ada uji preview/download berkas karena artefak legacy sengaja tidak
dibawa pada rehearsal ini. Lihat `docs/S9_REHEARSAL_RUNBOOK.md` untuk menjalankan
ulang atau memulihkan backup rehearsal.
