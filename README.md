# K12-Reverse ChatGPT Creator

> Otomatisasi Registrasi Akun ChatGPT Skala Besar dengan Fitur K12 Invite dan Multi-Gmail IMAP.

K12-Reverse adalah *tool* berbasis Go (Golang) untuk melakukan registrasi akun ChatGPT secara massal. Dibuat dengan antarmuka CLI yang interaktif, *tool* ini memanfaatkan teknik Dot-Trick pada Gmail dan integrasi IMAP untuk mengekstrak OTP secara otomatis tanpa intervensi manual.

---

## 🚀 Fitur Utama

- **Multi-Gmail Dot-Trick**: Menghasilkan ribuan variasi email unik dari satu akun Gmail dasar tanpa memicu sistem anti-spam.
- **IMAP Auto-Read**: Membaca kotak masuk Gmail secara *headless* via protokol IMAP untuk memvalidasi OTP (One Time Password) secepat kilat.
- **K12 Auto-Invite**: Menggabungkan akun baru ke dalam *workspace* Edukasi (K12) secara instan, lengkap dengan ekstraksi *Token Session*.
- **Multi-Threading / Workers**: Mendukung registrasi konkurensi (berjalan bersamaan) untuk kecepatan maksimal.
- **Anti-Zombie System**: Deteksi otomatis untuk melewati (*skip*) email yang sebelumnya gagal atau menggantung di tengah pendaftaran.
- **Smart Proxy Support**: Mendukung SOCKS5 / HTTP Proxy dengan *auth* URL (contoh: `socks5://user:pass@host:port`).
- **Auto-Resume**: Melanjutkan registrasi yang tertunda akibat kegagalan proxy atau terputusnya koneksi, tepat di titik berhentinya.

---

## 🛠️ Persyaratan Sistem

Sebelum menjalankan program ini, pastikan mesin atau server Anda telah terinstal:

- **Go (Golang)**: Versi 1.20 atau yang lebih baru.
- **Koneksi Internet Stabil**: Disarankan menggunakan Proxy berkualitas (Residential/Static) untuk menghindari limitasi *rate-limit* dari Cloudflare/OpenAI.
- **Akun Gmail**: Akun Gmail utama (*base email*) beserta **App Password**-nya (Sandi Aplikasi).

### Cara Mendapatkan App Password Gmail
Demi keamanan, Anda tidak bisa menggunakan kata sandi asli Gmail. Anda harus membuat Sandi Aplikasi (App Password):
1. Aktifkan **Verifikasi 2 Langkah (2FA)** di akun Google Anda.
2. Masuk ke setelan **Keamanan** akun Google.
3. Cari menu **Sandi Aplikasi** (App Passwords).
4. Buat sandi baru (Pilih "Lainnya", beri nama misalnya "K12-Bot").
5. Salin 16 digit huruf yang muncul (tanpa spasi). Ini adalah kredensial yang akan digunakan dalam *tool*.

---

## 📦 Instalasi & Penggunaan

1. **Kloning Repositori**
   ```bash
   git clone https://github.com/ahmadd4vd/k12-reverse.git
   cd k12-reverse
   ```

2. **Jalankan Program**
   Anda tidak perlu mengatur konfigurasi manual. Program memiliki asisten pengaturan (*wizard*) interaktif:
   ```bash
   go run cmd/register/main.go
   ```

3. **Konfigurasi via CLI**
   Pilih opsi **[2] Edit Configuration & Gmail Accounts**. Anda akan dipandu untuk:
   - Memasukkan *Base Email* (contoh: `nama.email@gmail.com`).
   - Memasukkan *App Password* yang baru saja Anda buat.
   - Mengatur URL Proxy (opsional).
   - *Tool* akan otomatis menghasilkan variasi Dot-Trick dan menyimpannya di direktori `data/`.

4. **Mulai Registrasi**
   Pilih opsi **[1] Start Registration** dari menu utama. Tentukan jumlah *worker* (konkurensi) yang diinginkan, dan program akan berjalan sepenuhnya otomatis.

---

## ⚠️ Memahami Akun "Zombie"

Saat menjalankan program, Anda mungkin menemui peringatan seperti ini di terminal:
`SKIP: Account is a zombie (partially registered, can't login or create).`

**Apa itu Akun Zombie?**
Sistem registrasi saat ini **hanya berfokus pada Sign-Up (Pembuatan Akun)** dan melewati fitur Login. Jika dalam eksekusi sebelumnya program terhenti/gagal (misalnya proxy terputus atau *rate-limit*) di tengah-tengah proses pendaftaran, email tersebut sudah tercatat di sistem *backend* OpenAI, tetapi profil dan *password*-nya belum terbentuk sempurna.

Akibatnya, email tersebut menjadi "menggantung" (Zombie). Saat *tool* mencoba mendaftarkannya ulang, sistem akan merespon `user_already_exists`, namun email tersebut tidak memiliki kredensial Login yang valid. 
Sistem K12-Reverse telah diprogram untuk otomatis mendeteksi dan mengabaikan email-email ini, sehingga antrean Anda tetap berjalan mulus ke variasi Dot-Trick berikutnya.

---

## 🔮 Rencana Pengembangan Selanjutnya

Fitur utama yang sedang dalam tahap riset dan pengembangan selanjutnya:
- **Sistem Login Universal**: Modul baru untuk melakukan autentikasi penuh (Login) pada akun yang sudah jadi, memungkinkan sinkronisasi profil, perbaikan akun zombie, dan pembaruan Token secara berkala tanpa harus mendaftar ulang.

---

## 🤝 Kontribusi

Kontribusi selalu terbuka! Jika Anda memiliki perbaikan kode, optimasi *bypass*, atau fitur baru:
1. *Fork* repositori ini.
2. Buat *branch* fitur Anda (`git checkout -b fitur/NamaFitur`).
3. Lakukan *commit* perubahan (`git commit -m "Menambahkan fitur X"`).
4. *Push* ke *branch* (`git push origin fitur/NamaFitur`).
5. Buka **Pull Request**.

Pastikan kode Anda rapi dan mematuhi konvensi bahasa Go (`gofmt`).

---

## 📄 Lisensi

Didistribusikan di bawah Lisensi MIT. Lihat file `LICENSE` untuk informasi selengkapnya.

> Dibuat oleh **Ahmadd4vd**
