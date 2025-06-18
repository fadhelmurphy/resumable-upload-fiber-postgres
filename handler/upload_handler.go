package handler

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AbortUpload(c *fiber.Ctx, db *sql.DB) error {
	filename := c.Query("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing filename",
		})
	}

	uploadDir := "uploads"
	filePath := filepath.Join(uploadDir, filename)

	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete file: " + err.Error(),
			})
		}
	}

	_, err := db.Exec(`DELETE FROM uploads WHERE filename = $1`, filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete db record: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Upload aborted and cleaned up",
	})
}


func UploadChunk(c *fiber.Ctx, db *sql.DB) error {
	filename := c.Get("Upload-File-Name")
	offset, _ := strconv.ParseInt(c.Get("Upload-Offset"), 10, 64)
	totalSize, _ := strconv.ParseInt(c.Get("Upload-Total-Size"), 10, 64)

	ext := strings.ToLower(filepath.Ext(filename))
	allowed := []string{".pdf", ".jpg", ".jpeg", ".png"}
	valid := false
	for _, a := range allowed {
		if ext == a {
			valid = true
			break
		}
	}
	if !valid {
		return c.Status(fiber.StatusBadRequest).SendString("File type not allowed")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Missing file in request")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	path := filepath.Join("uploads", filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(offset, 0); err != nil {
		return err
	}
	if _, err := io.Copy(f, src); err != nil {
		return err
	}

	info, err := f.Stat()
	if err != nil {
		return err
	}

	status := "in-progress"
	var md5sum, sha256sum string
	if info.Size() >= totalSize {
		status = "complete"
		md5sum, sha256sum = computeChecksums(path)
		db.Exec(`UPDATE uploads SET md5=$1, sha256=$2 WHERE filename=$3`, md5sum, sha256sum, filename)
	}

	_, err = db.Exec(`
        INSERT INTO uploads (filename, size, status, md5, sha256, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        ON CONFLICT (filename) DO UPDATE SET size=$2, status=$3, md5=$4, sha256=$5, updated_at=NOW()
    `, filename, info.Size(), status, md5sum, sha256sum)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"uploaded": info.Size(),
		"status":   status,
	})
}

func CheckStatus(c *fiber.Ctx, db *sql.DB) error {
	filename := c.Query("filename")
	var size int64
	var status, md5sum, sha256sum string
	err := db.QueryRow(`SELECT size, status, md5, sha256 FROM uploads WHERE filename=$1`, filename).
		Scan(&size, &status, &md5sum, &sha256sum)
	if err != nil {
		return c.Status(404).SendString("Not found")
	}
	return c.JSON(fiber.Map{
		"filename": filename,
		"size":     size,
		"status":   status,
		"md5":      md5sum,
		"sha256":   sha256sum,
	})
}

func computeChecksums(path string) (string, string) {
	f, _ := os.Open(path)
	defer f.Close()
	md5h := md5.New()
	sha := sha256.New()
	io.Copy(io.MultiWriter(md5h, sha), f)
	return hex.EncodeToString(md5h.Sum(nil)), hex.EncodeToString(sha.Sum(nil))
}
