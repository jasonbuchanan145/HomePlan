package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"homeplan/api/internal/httpapi"
	"homeplan/api/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)

	port := env("PORT", "8080")
	cookieName := env("SESSION_COOKIE_NAME", "homeplan_session")
	devMode := env("HOMEPLAN_DEV_MODE", "false") == "true"
	ttl := 14 * 24 * time.Hour

	log.Printf("homeplan api starting on port %s devMode=%t", port, devMode)

	var houseStore httpapi.HouseStore
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Println("DATABASE_URL is not set; using in-memory store")
		houseStore = store.NewMemoryStore()
	} else {
		log.Println("DATABASE_URL is set; opening Postgres connection")
		db, err := sql.Open("pgx", databaseURL)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer db.Close()

		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)

		if err := waitForDatabase(db, 90*time.Second, 2*time.Second); err != nil {
			log.Fatalf("database unavailable: %v", err)
		}

		log.Println("Postgres connection is ready")
		houseStore = store.NewPostgresStore(db)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewRouter(houseStore, cookieName, ttl, devMode),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("homeplan api listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func waitForDatabase(db *sql.DB, timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	attempt := 1

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for Postgres: %w", timeout, err)
		}

		log.Printf("waiting for Postgres attempt %d failed: %v", attempt, err)
		attempt++
		time.Sleep(interval)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
