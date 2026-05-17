package main

import (
	"flag"
	"log"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/db"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
)

func main() {
	down := flag.Bool("down", false, "drop tables instead of migrating up")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	if *down {
		if err := gdb.Migrator().DropTable(&models.Applicant{}, &models.Position{}); err != nil {
			log.Fatalf("drop: %v", err)
		}
		log.Println("rollback complete")
		return
	}

	if err := gdb.AutoMigrate(&models.User{}, &models.Session{}, &models.Resume{}, &models.Position{}, &models.Applicant{}, &models.Template{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migration complete")
}
