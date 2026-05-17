package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/db"
	"github.com/ayukumar261/hackathon/go-api/internal/handlers"
	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/ayukumar261/hackathon/go-api/internal/oauth"
	"github.com/ayukumar261/hackathon/go-api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

	auth := &handlers.AuthHandler{DB: gdb, Cfg: cfg, OAuthCfg: oauth.GoogleConfig(cfg)}
	sess := &handlers.SessionHandler{DB: gdb}

	s3Client, presign, err := storage.NewR2Client(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Endpoint:        cfg.R2Endpoint,
	})
	if err != nil {
		log.Fatalf("r2: %v", err)
	}
	resumes := &handlers.ResumesHandler{DB: gdb, S3: s3Client, Presign: presign, Bucket: cfg.R2Bucket}
	positions := &handlers.PositionsHandler{DB: gdb}
	applicants := &handlers.ApplicantsHandler{DB: gdb}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Get("/api/users/google", auth.GoogleStart)
	r.Get("/api/users/google/callback", auth.GoogleCallback)
	r.Get("/sessions", sess.Get)
	r.Delete("/sessions", sess.Delete)

	r.Route("/api/resumes", func(r chi.Router) {
		r.Use(mw.RequireUser(gdb))
		r.Post("/", resumes.Upload)
		r.Get("/", resumes.List)
		r.Get("/{id}", resumes.DownloadURL)
		r.Delete("/{id}", resumes.Delete)
	})

	r.Route("/api/positions", func(r chi.Router) {
		r.Use(mw.RequireUser(gdb))
		r.Post("/", positions.Create)
		r.Get("/", positions.List)
		r.Get("/{id}", positions.Get)
		r.Patch("/{id}", positions.Update)
		r.Delete("/{id}", positions.Delete)
	})

	r.Route("/api/applicants", func(r chi.Router) {
		r.Use(mw.RequireUser(gdb))
		r.Post("/", applicants.Create)
		r.Get("/", applicants.List)
		r.Get("/{id}", applicants.Get)
		r.Patch("/{id}", applicants.Update)
		r.Delete("/{id}", applicants.Delete)
	})

	addr := ":" + cfg.Port
	log.Printf("go-api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
