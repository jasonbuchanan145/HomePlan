package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"homeplan/api/internal/httpapi"
	"homeplan/api/internal/store"
)

func main() {
	ctx := context.Background()
	port := env("PORT", "8080")
	cookieName := env("SESSION_COOKIE_NAME", "homeplan_session")
	ttl := 14 * 24 * time.Hour

	var houseStore httpapi.HouseStore
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Println("DATABASE_URL is not set; using in-memory store")
		houseStore = store.NewMemoryStore()
	} else {
		db, err := sql.Open("pgx", databaseURL)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer db.Close()
		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("ping database: %v", err)
		}
		houseStore = store.NewPostgresStore(db)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewRouter(houseStore, cookieName, ttl),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("homeplan api listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
