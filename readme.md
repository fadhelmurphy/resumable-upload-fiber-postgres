# Resumable File Upload Service

Sebuah project **Resumable File Upload** menggunakan:
- **Golang + Fiber** sebagai backend
- **PostgreSQL** untuk metadata file upload
- **Redis** untuk rate limiter
- **Vanilla JavaScript (fetch API)** untuk frontend dengan progress bar dan auto-resume
- **Docker Compose** untuk environment

---

## Fitur
- Resumable upload (lanjut otomatis jika jaringan kembali)  
- Progress bar real-time  
- Validasi file type (.pdf, .jpg, .jpeg, .png)  
- MD5 + SHA256 checksum (integrity check)  
- Rate limit (Redis-based limiter)  
- Auto clean unfinished uploads  
- Check file status endpoint  
- Network-aware upload (pause saat offline dan melanjutkan saat online)

---

## Arsitektur
```
Frontend JS (Fetch + Progress Bar)
↓
Fiber Server (Upload, Status, Checksum)
↓
Postgres (store file metadata: size, status, hash)
↓
Redis (rate limiter per client)
↓
Disk (folder uploads/)
```

---

## Cara Menjalankan

### Jalankan Docker Compose
```bash
docker compose up --build
```

Ini akan menjalankan:

- Fiber app (port 3000)

- PostgreSQL (port 5432)

- pgAdmin (port 8080)

- Redis (port 6379)

## Cara Stop Project

```bash
docker compose down -v
```