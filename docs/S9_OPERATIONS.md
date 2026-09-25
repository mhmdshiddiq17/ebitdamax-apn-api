# Sprint 9 — Operasional dan Monitoring Rehearsal

## Pemeriksaan layanan

| Komponen | Pemeriksaan | Kondisi sehat |
|---|---|---|
| API | `GET /healthz` | HTTP 200; database, Redis, dan MinIO `up` |
| PostgreSQL | `pg_isready` | menerima koneksi |
| Redis | `redis-cli ping` | `PONG` |
| MinIO | `/minio/health/live` | HTTP 200 |
| Docker | `docker compose ps` | seluruh dependency `healthy` |

Gunakan log `go run .` dan `npm run dev` untuk diagnosis lokal. Tidak ada
vendor observability atau schema monitoring baru pada rehearsal ini.

## Respons insiden

- Healthz gagal: hentikan verifikasi, cek dependency yang ditandai `down`, lalu
  ulangi dari health check; jangan melakukan migrasi ulang pada database yang
  sudah berisi data tanpa `--reset`.
- Count atau verifikasi metadata berkas gagal: hapus database rehearsal saja
  atau pulihkan backup `--reset`; jangan mengubah legacy untuk memperbaiki hasil.
- Login QA gagal: validasi akun seed environment dan ulangi rehearsal dari
  database kosong; jangan memakai password Manager legacy.

## Bukti selesai S9-4/S9-5

- Output preflight dan migrator tersimpan pada catatan eksekusi lokal.
- Count target, metadata file nol, health check, dan boundary Superadmin 403
  tercatat lulus.
- Database refactor utama dan legacy terbukti tidak menjadi target command.
