# Sprint 15 — Operasional Sinkronisasi Sarpras & Data SDM

Runbook untuk alur dua tahap: portal pembangunan → `koperasi_sarpras_status_points`
(sinkron berkala) → `sdm_kdkmp_entries` (derive manual). `jumlah_karyawan` tetap
diisi manual oleh superadmin lewat halaman `/sdm-data`.

## Variabel environment

| Variabel | Wajib | Default | Catatan |
|---|---|---|---|
| `PORTAL_PEMBANGUNAN_BASE_URL` | tidak | `https://portalkdkmp.id` | Endpoint yang dipanggil: `{base}/api/koperasi-sarpras-status` |
| `PORTAL_PEMBANGUNAN_SARPRAS_TOKEN` | ya (untuk sync) | — | Bearer token portal; **jangan commit**, hanya di `.env` lokal/secret manager |
| `SARPRAS_SCHEDULER_ENABLED` | tidak | `false` | `true` hanya pada instance yang menjalankan cron (dev: biarkan `false`) |
| `SARPRAS_SYNC_INTERVAL` | tidak | `@every 15m` | Format `robfig/cron` (`@every 15m`, `0 */15 * * * *`, dst.) |

## CLI

### `cmd/sync-sarpras` — tarik titik dari portal

```sh
# pratinjau tanpa menulis (tetap memanggil portal)
go run ./cmd/sync-sarpras --dry-run --max-pages 2

# uji parsial 2 halaman (500 baris/halaman)
go run ./cmd/sync-sarpras --max-pages 2

# sinkronisasi penuh (≈73 halaman, 36.359 titik, 3–4 menit)
go run ./cmd/sync-sarpras
```

- Paginasi 500 baris; baris tanpa koordinat valid dilewati dan dihitung.
- Upsert `ON CONFLICT (nik, lat, lng)`; baris yang tidak muncul lagi pada sync
  ini dihapus (`synced_at < waktu mulai sync`).
- Retry otomatis 3× (backoff 2s/4s) untuk error jaringan/timeout/5xx; error 4xx
  (mis. token salah) langsung gagal tanpa retry.
- `--max-pages` aman dipakai berulang: upsert bersifat idempoten.

### `cmd/sync-sdm` — derive ke data SDM

```sh
# default dry-run: hanya ringkasan
go run ./cmd/sync-sdm

# tulis hasil
go run ./cmd/sync-sdm --apply
```

- Sumber: titik `sarpras_primary_lengkap = true`, unik per NIK (baris terbaru
  menang), NIK wajib terisi.
- Kolom `sdm_kdkmp_entries` yang **ditimpa**: `nama_koperasi`, `provinsi`,
  `nama_kodim`, `desa`, `kecamatan`, `kota_kabupaten`, `batch`, `updated_at`.
- Kolom **terlindungi** (tidak pernah ditimpa): `jumlah_karyawan`, `catatan`,
  `created_by`, `updated_by`, `created_at`, `nik`.
- Dibatalkan tanpa menulis bila ada sumber dengan nama koperasi kosong.

## Jadwal (cron di proses API)

Aktifkan di `.env` instance yang berjalan terus:

```
SARPRAS_SCHEDULER_ENABLED=true
SARPRAS_SYNC_INTERVAL=@every 15m
```

- Job memakai `SkipIfStillRunning` (satu siklus tidak tumpang tindih) dan
  advisory lock PostgreSQL `pg_try_advisory_lock` — pada deployment
  multi-instance hanya satu proses yang benar-benar sync.
- Log: `scheduler sarpras aktif (...)` saat start; ringkasan
  `sync sarpras: N halaman, M titik, K stale dihapus` setiap selesai; kegagalan
  dicatat `sync sarpras gagal: ...` dan siklus berikutnya mencoba lagi.
- Matikan dengan `SARPRAS_SCHEDULER_ENABLED=false` lalu restart API. Derive
  **tidak** dijadwalkan (manual by design).

## Prosedur operasional rutin

1. Pastikan dependency sehat: `GET /healthz` → database/redis/minio `up`.
2. Sync titik (cron sudah jalan di produksi; manual bila perlu):
   `go run ./cmd/sync-sarpras`.
3. Verifikasi cepat:

   ```sql
   SELECT count(*) AS titik,
          count(*) FILTER (WHERE sarpras_primary_lengkap) AS siap_derive,
          count(*) FILTER (WHERE nik IS NULL OR nik = '') AS tanpa_nik
   FROM koperasi_sarpras_status_points;
   ```

4. Derive: `go run ./cmd/sync-sdm` (dry-run) → periksa ringkasan → `--apply`.
5. Isi `jumlah_karyawan` lewat `/sdm-data` (superadmin). Entri hasil derive
   belum punya akun manager sehingga tidak tampil di monitoring sampai
   manager ditautkan.

## Troubleshooting

| Gejala | Penyebab | Tindakan |
|---|---|---|
| `PORTAL_PEMBANGUNAN_SARPRAS_TOKEN belum di-set` | token kosong | isi `.env`; restart API bila memakai cron |
| `portal mengembalikan HTTP 401` | token salah/kedaluwarsa | perbarui token; command tidak mengulang 4xx |
| `portal gagal setelah 3 percobaan` | portal timeout/5xx berulang | ulangi command (idempoten); jangan jalankan dua sync bersamaan |
| `membaca respons portal: stream error ... INTERNAL_ERROR` | portal flaky di HTTP/2 | sudah ditangani: klien memakai HTTP/1.1 + retry |
| Jumlah titik naik lalu turun setelah sync berikutnya | baris NIK kosong ter-insert ulang (unique `(nik,lat,lng)` tidak menahan NULL) | normal (parity lama); dibersihkan oleh stale-delete pada sync sukses berikutnya |
| `derivasi dibatalkan: N data sumber tidak memiliki nama koperasi` | data portal tidak lengkap | perbaiki di portal; tidak ada data yang ditulis |
| `jumlah_karyawan` tidak berubah setelah derive | kolom terlindungi | by design; isi manual via `/sdm-data` |

## Batas & keamanan

- Token portal tidak pernah di-commit (`.env.example` hanya placeholder).
- Halaman `/sdm-data` dan update `jumlah_karyawan` hanya untuk superadmin;
  manager wilayah/manager KDKMP mendapat 403.
- Objek berkas legacy tidak disalin dan tidak diperlukan alur ini.
- Sinkronisasi penuh menulis ±36 ribu baris; jalankan di luar jam sibuk portal
  bila memungkinkan.

## Runbook E2E (bukti 27 Sep 2026)

1. `go run ./cmd/sync-sarpras --dry-run --max-pages 2` → 2 halaman, 1.000 titik.
2. `go run ./cmd/sync-sarpras --max-pages 2` → 1.000 baris tersimpan.
3. `go run ./cmd/sync-sarpras` → **36.359 titik / 73 halaman / 3m10s**; ulangi
   sync → jumlah tetap, stale-delete membersihkan duplikat (`dup = 0`).
4. `go run ./cmd/sync-sdm` → sumber unik 6.449, diperbarui 2.002, ditambahkan
   4.447, total SDM 6.456.
5. `go run ./cmd/sync-sdm --apply` → marker uji (`jumlah_karyawan`, `catatan`)
   tetap utuh; entri demo `DEMO-KDKMP-LEGACY` tidak tersentuh; derive ulang → 0
   insert.
6. Scheduler smoke: jalankan API dengan `SARPRAS_SCHEDULER_ENABLED=true
   SARPRAS_SYNC_INTERVAL='@every 10s'` di port terpisah → log
   `scheduler sarpras aktif`, job menulis ribuan baris dalam satu menit.
7. UI: superadmin membuka `/sdm-data` → 6.456 baris, 259 halaman, edit inline
   `jumlah_karyawan` + ringkasan berubah; manager wilayah → menu tidak tampil
   dan akses langsung dialihkan ke `/dashboard`.
