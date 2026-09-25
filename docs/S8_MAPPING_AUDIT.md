# Sprint 8 — Audit Mapping Revenue dan Biaya

Tanggal audit: 25 September 2026. Audit ini hanya membaca kode refactor dan
data lokal aplikasi legacy `../ebitdamax-apn`; tidak ada perubahan skema,
formula, atau data produksi.

## Mapping yang dipakai refactor

| Nilai | Task | Field | Aturan |
|---|---|---|---|
| Actual revenue harian | `Penyetoran Struk dan Uang` | `rekonsiliasi_uang_masuk` | Menjumlahkan nilai numerik laporan selesai pada hari berjalan. |
| Actual cost harian | `Pencatatan Pengeluaran Operasional Harian` | seluruh field bernilai numerik | Hanya task yang masuk daftar task manager pada hari itu. |
| Variable cost 30 hari | `Pencatatan Pengeluaran Operasional Harian` | `token_listrik`, `bahan_bakar_kendaraan` | Hanya kelebihan di atas batas masing-masing Rp3.000.000 dan Rp2.000.000. |

Implementasi sumber berada pada `internal/kdkmp/dashboard.go`: `dailyRevenueAndCost`
dan `expenseFieldTotals`. Nilai non-numerik diabaikan sehingga field unggahan
tidak dapat masuk ke perhitungan.

## Hasil pembandingan data legacy

- Dataset legacy memiliki task dan field kanonis di atas, termasuk
  `Penyetoran Struk dan Uang / rekonsiliasi_uang_masuk` dan task pengeluaran
  dengan dua field biaya kanonis.
- Ada nama lama yang berbeda, misalnya `Catat pengeluaran harian`,
  `bbm_kdkmp`, dan `listrik_token`. Laporan dengan nama lama tersebut tidak
  dihitung oleh refactor saat ini.
- Satu task penyetoran lama memakai field unggahan, bukan field angka revenue;
  nilainya memang tidak boleh dihitung sebagai revenue.

## Keputusan

Mapping tetap **read-only dan strict** agar hasil tidak mencampur data dari
task atau field yang berbeda arti. Normalisasi alias task/field lama, jika
dibutuhkan setelah cutover, dicatat sebagai pekerjaan Sprint 9 dan perlu
persetujuan perubahan data terlebih dahulu.

## Query verifikasi ulang

Jalankan hanya terhadap database legacy dengan akses baca:

```sql
SELECT t.id, t.name AS task_name, f.field_name
FROM tasks t
LEFT JOIN task_additional_fields f ON f.task_id = t.id
WHERE t.name IN ('Penyetoran Struk dan Uang', 'Pencatatan Pengeluaran Operasional Harian')
   OR t.name ILIKE '%pengeluaran%'
   OR t.name ILIKE '%setor%'
ORDER BY t.id, f.id;
```

```sql
SELECT t.name AS task_name, f.field_name, COUNT(*) AS completed_values
FROM task_report_values v
JOIN task_reports r ON r.id = v.task_report_id AND r.status = 'completed'
JOIN tasks t ON t.id = r.task_id
JOIN task_additional_fields f ON f.id = v.task_additional_field_id
GROUP BY t.name, f.field_name
ORDER BY completed_values DESC, task_name, f.field_name;
```
