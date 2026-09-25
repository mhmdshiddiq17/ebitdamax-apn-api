# Audit Gap Legacy Manager KDKMP

Tanggal audit: 25 September 2026. Sumber perilaku adalah
`../ebitdamax-apn`; refactor hanya mengejar alur Manager KDKMP
(`roles.domain=kdkmp`, `roles.slug=manager`).

## Status paritas

| Alur legacy Manager | Status refactor | Keputusan |
|---|---|---|
| Dashboard, input harian, task, laporan, dan riwayat | selesai | Sprint 4–5 |
| Meeting Minutes dan Action Items | selesai | Sprint 7 |
| Profil, SK, onboarding, 2FA | selesai | Sprint 1–2, 8 |
| Customer Analysis | selesai | Sprint 10; tabel baru, data mulai kosong |
| Pengumuman dan notifikasi | selesai | Sprint 11; pengumuman hanya ke Manager KDKMP |
| POS Revenue read-only | belum | Sprint 12 blocked oleh kontrak POS |
| Plan EBITDA Matrix | tidak dikerjakan | canceled |
| UI passkeys | tidak dikerjakan | skip; backend tetap tersedia |
| Verifikasi email dan reset password | tidak dikerjakan | skip |
| Monitoring melalui URL legacy | tidak dikerjakan | di luar alur navigasi Manager |
| LMS KDKMP dan Lumbung Chat | tidak dikerjakan | tidak lagi digunakan |
| SSO Lark | belum | blocked oleh akses dan kontrak eksternal |

## Data dan schema

`migrations/00002_customer_analyses.sql` menambahkan Customer Analysis sesuai
struktur legacy. Migration ini diterapkan pada database refactor dan oleh
`scripts/s9-rehearsal.sh` saat rehearsal dibangun ulang. Riwayat Customer
Analysis maupun notifikasi legacy tidak diimpor sesuai keputusan produk.

Tabel `notifications` sudah ada pada baseline. Pengumuman baru disimpan dengan
penerima bertipe user dan dibatasi pada role Manager KDKMP; Superadmin tidak
dapat membaca notifikasi Manager melalui endpoint tersebut.

Smoke rehearsal 25 September 2026 membuat satu Customer Analysis uji dan satu
pengumuman uji hanya di `ebitdamax_apn_rehearsal`; keduanya hilang saat
rehearsal dibangun ulang oleh skrip S9.

## Prasyarat POS Revenue

Sebelum Sprint 12 diaktifkan, validasi base URL, client credential, pemetaan
NIK ke `companyId`, format respons, kebijakan timeout, dan cache API POS.
Tanpa itu aplikasi tidak boleh menampilkan atau menyimpan pendapatan manual
sebagai pengganti nilai POS legacy.
