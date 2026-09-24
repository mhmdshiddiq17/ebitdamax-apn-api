# Import Data Demo KDKMP

Command ini mengimpor satu fixture demo Manager KDKMP dari database legacy
lokal. Ia tidak mengimpor akun legacy, NIK asli, foto, dokumen, Plan EBITDA
Matrix, atau Meeting Minutes.

Prasyarat: PostgreSQL refactor berjalan melalui Docker, akun Manager seed
sudah dibuat dengan `go run ./cmd/seed`, dan variabel `LEGACY_DB_*` tersedia
di `.env` atau environment shell.

Periksa data tanpa menulis apa pun:

```sh
go run ./cmd/import-legacy-kdkmp-demo
```

Jika ringkasan sudah benar, lakukan import:

```sh
go run ./cmd/import-legacy-kdkmp-demo --apply
```

Command berhenti bila master task sudah terisi, akun Manager sudah terhubung
ke KDKMP lain, atau marker `DEMO-KDKMP-LEGACY` sudah ada. Hal ini mencegah
data kerja lokal tertimpa oleh fixture demo.
