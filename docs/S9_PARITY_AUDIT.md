# Sprint 9 — Audit Paritas Manager KDKMP

Tanggal audit: 25 September 2026. Sumber perilaku adalah `../ebitdamax-apn`.
Refactor ini sengaja hanya mencakup Manager KDKMP
(`roles.domain=kdkmp` dan `roles.slug=manager`).

## Hasil audit in-scope

| Alur legacy | Refactor API | Halaman refactor | Gate |
|---|---|---|---|
| Redirect dashboard Manager KDKMP | `GET /auth/me` | `/dashboard` ke `/dashboard/kdkmp` | role domain+slug exact |
| Dashboard KDKMP dan input harian | `/kdkmp-dashboard`, `/input`, `/today` | `/dashboard/kdkmp`, `/dashboard/kdkmp/input` | Manager KDKMP |
| Task harian, laporan, riwayat, dan berkas | `/task-dashboard`, `/tasks/*`, `/task-reports/*` | `/dashboard/tasks`, `/dashboard/tasks/completed` | Manager KDKMP |
| Meeting Minutes, Action Items, dan lampiran | `/meeting-minutes/*` | `/meeting-minutes`, `/meeting-minutes/action-items` | Manager KDKMP + owner |
| Profil, SK Manager, onboarding | `/profile`, `/users/:id/manager-sk-document`, `/users/complete-onboarding` | `/settings/profile` | autentikasi; SK dan onboarding memeriksa Manager KDKMP |
| Customer Analysis | `/customer-analyses` | `/customer-analyses` | Manager KDKMP + owner |
| Pengumuman dan notifikasi | `/announcements`, `/notifications` | `/announcements`, `/notifications` | Superadmin kirim; Manager KDKMP baca milik sendiri |

Task Dashboard legacy masih memakai middleware level yang lebih luas. Refactor
mempersempit seluruh endpoint task ke `RequireKdkmpManager`. Menu task tidak
lagi ditampilkan untuk Manager Wilayah. Meeting memakai gate dan query
kepemilikan; akun lain tidak dapat membaca atau mengubah data Manager KDKMP.

## Sengaja di luar paritas aktif

- POS Revenue: `blocked` hingga kontrak API eksternal tervalidasi.
- Monitoring regional/superadmin, APN corporate, import, dan peta bukan alur
  navigasi Manager KDKMP aktif.
- LMS dan Lumbung Chat dihentikan karena tidak lagi digunakan; SSO Lark masih
  menunggu kontrak eksternal.
- Plan EBITDA Matrix: dibatalkan dan tidak diaudit sebagai fitur aktif.

## Matriks verifikasi akses

| Identitas | Endpoint Manager | Data milik Manager lain |
|---|---|---|
| Tanpa sesi | 401 | 401 |
| Manager KDKMP | 200 sesuai kepemilikan | 404 / tidak tampil |
| Manager Wilayah | 403 | 403 |
| Superadmin | 403 | 403 |

Status 403 adalah keputusan produk refactor. Superadmin tetap hanya mengelola
master data pendukung.

## Performance dan validasi

- Login dan challenge mengganti ID sesi yang disajikan. Ganti kata sandi
  mencabut seluruh sesi akun dari Redis sebelum menerbitkan sesi baru.
- Query task memakai indeks baseline `task_roles(role_id, task_id)` serta
  `task_reports` per task/user/periode/status. Dashboard KDKMP memakai indeks
  tanggal dan constraint `(sdm_kdkmp_entry_id, report_date)`.
- Action Items sudah dipaginasi 15 baris. Pencarian Meeting dan Action Items
  kini dibatasi 255 karakter sebelum membentuk pola `ILIKE`.
- Tidak ada indeks atau perubahan schema baru karena baseline Sprint 7–9
  dilarang berubah dan query in-scope sudah memiliki indeks yang diperlukan.
