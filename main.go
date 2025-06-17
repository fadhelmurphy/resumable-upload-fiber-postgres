package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"log"

	"resumable-upload/handler"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/storage/redis/v2"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	app := fiber.New()

	var store *redis.Storage
	for {
		store = redis.New(redis.Config{
			Host:     os.Getenv("REDIS_HOST"),
			Database: 0,
			Reset:    false,
		})
		if err := store.Set("ping", []byte("pong"), 1*time.Second); err == nil {
			break
		}
		log.Println("Waiting for Redis to be ready...")
		time.Sleep(1 * time.Second)
	}
	goLimiter := limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
		},
		Storage: store,
	})

	app.Static("/", "./static")

	app.Post("/upload", func(c *fiber.Ctx) error {
		return handler.UploadChunk(c, db)
	})

	app.Delete("/upload", func(c *fiber.Ctx) error {
    return handler.AbortUpload(c, db)
	})

	app.Get("/status", goLimiter, func(c *fiber.Ctx) error {
		return handler.CheckStatus(c, db)
	})

	go autoClean(db)

	app.Listen(":3000")
}

func autoClean(db *sql.DB) {
	for {
		time.Sleep(30 * time.Minute)
		rows, err := db.Query(`SELECT filename FROM uploads WHERE status='in-progress' AND updated_at < NOW() - INTERVAL '1 hour'`)
		if err != nil {
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var filename string
			rows.Scan(&filename)
			os.Remove(filepath.Join("uploads", filename))
			db.Exec(`DELETE FROM uploads WHERE filename=$1`, filename)
			fmt.Println("Auto cleaned:", filename)
		}
	}
}
