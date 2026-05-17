package main

import (
	"log"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/db"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := gdb.AutoMigrate(&models.User{}, &models.Session{}, &models.Resume{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migration complete")
}
