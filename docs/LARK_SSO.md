# SSO Lark untuk Manager KDKMP

Integrasi ini memakai **Custom App Lark baru**. Hanya user dengan role `domain=kdkmp` dan `slug=manager` yang dapat masuk lewat Lark. Akun harus sudah ada di aplikasi dengan email yang sama; SSO tidak membuat user baru. Login email/kata sandi tetap tersedia.

## Konfigurasi

1. Buat Custom App di Lark Developer Console, aktifkan Web App/H5, dan beri izin `contact:user.email:readonly`. Terbitkan versi aplikasi dan pastikan izin disetujui tenant.
2. Daftarkan redirect URI browser **persis** sama dengan `LARK_REDIRECT_URI`, yaitu `{URL_API}/api/v1/auth/lark/callback`. Daftarkan URL frontend HTTPS sebagai alamat Web App/H5. URL `localhost` hanya contoh dev; gunakan HTTPS publik/tunnel jika Lark tidak menerima localhost.
3. Jalankan migrasi `00003_lark_sso.sql` melalui Goose setelah migrasi sebelumnya. Migrasi menambah `users.lark_open_id` nullable dan unique index. Jangan menjalankan seluruh file SQL langsung karena bagian `Down` ada di bawahnya.
4. Isi `.env` backend dari `.env.example`: `LARK_APP_ID`, `LARK_APP_SECRET`, `LARK_REDIRECT_URI`, `LARK_FRONTEND_URL`, lalu set `LARK_SSO_ENABLED=true`. `LARK_BASE_URL`, `LARK_AUTHORIZATION_URL`, dan `LARK_SCOPES` sudah memiliki default Lark global. Set `CORS_ALLOWED_ORIGINS` ke origin frontend. Simpan secret hanya di backend.
5. Pastikan `NEXT_PUBLIC_API_URL` frontend menunjuk API publik yang sama, termasuk `/api/v1`. Cookie autentikasi saat ini host-only, jadi sajikan FE dan BE pada **hostname yang sama** (misalnya melalui reverse proxy) agar redirect SSO menghasilkan sesi yang dapat dibaca Next.js. Port lokal 3000/4000 boleh berbeda karena cookie tidak dibatasi port. Restart BE/FE.

Alur browser: tombol **Masuk dengan Lark** → OAuth authorization code + PKCE/state → callback API → sesi cookie aplikasi → `/dashboard`. Alur H5: SDK resmi `h5-js-sdk-1.5.44.js` memanggil `tt.requestAccess`, fallback `tt.requestAuthCode` bila API belum tersedia/errno 103 → kode ditukar di backend → sesi aplikasi → `/dashboard`.

Kode dan state sekali pakai dengan TTL 5 menit untuk browser dan 3 menit untuk H5. `App Secret` serta token Lark tidak dikirim ke frontend. Pada kegagalan, cookie autentikasi dihapus dan halaman aplikasi tetap dijaga oleh `/auth/me`. SSO melewati 2FA lokal; jalur kata sandi tetap mengikuti aturan 2FA yang ada.

## Pemetaan akun dan pemulihan

- Login pertama mencari **tepat satu** user berdasarkan `LOWER(email)` dan mengikat `open_id`. Jika email duplikat, akun tidak ada, bukan Manager KDKMP, atau `open_id` sudah berbeda, login ditolak.
- Login berikutnya harus cocok pada **email dan `open_id`**. Perubahan email di Lark atau aplikasi perlu diselaraskan sebelum login ulang.
- Superadmin dapat memilih aksi **Lepas koneksi Lark** pada halaman User. Aksi ini menghapus binding dan mencabut refresh token user; access token yang sudah terbit tetap berlaku sampai TTL-nya (default 1 jam). Setelahnya, user dapat ditautkan kembali saat login dengan email yang cocok.

## Verifikasi

Jalankan `go test -mod=readonly ./...`, `go vet ./...`, `npm run lint`, dan `npm run build`. Dengan Custom App aktif, uji Manager KDKMP di H5 dan browser, lalu coba user tanpa role manager, email tak cocok, dan callback dengan state salah—semuanya harus gagal tanpa akses halaman. Uji ulang login email/kata sandi, 2FA, logout, dan reset binding superadmin.

Rujukan resmi: [H5 introduction](https://open.larksuite.com/document/client-docs/h5/introduction), [Step 2 JSAPI](https://open.larksuite.com/document/client-docs/h5/development-guide/step-2:-call-jsapi(optional)), [Step 3 login](https://open.larksuite.com/document/client-docs/h5/development-guide/step-3), [Step 4 server](https://open.larksuite.com/document/client-docs/h5/development-guide/step-4).
