package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ayukumar261/hackathon/go-api/internal/aigateway"
	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/db"
	"github.com/ayukumar261/hackathon/go-api/internal/handlers"
	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/ayukumar261/hackathon/go-api/internal/oauth"
	"github.com/ayukumar261/hackathon/go-api/internal/storage"
	"github.com/ayukumar261/hackathon/go-api/internal/templates"
	"github.com/ayukumar261/hackathon/go-api/internal/transcripts"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
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
	applicants := &handlers.ApplicantsHandler{DB: gdb, AgentPhoneAPIKey: cfg.AgentPhoneAPIKey, AgentPhoneAgentID: cfg.AgentPhoneAgentID}
	templatesHandler := &handlers.TemplatesHandler{DB: gdb}

	aiClient := aigateway.New(cfg.AIGatewayAPIKey, cfg.AgentPhoneLLMModel, cfg.AIGatewayBaseURL)

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}
	transcriptStore := transcripts.New(rdb)
	transcriptStream := &handlers.TranscriptStreamHandler{Transcripts: transcriptStore}
	templateStore := templates.NewRedisStore(rdb)
	templateStream := &handlers.TemplateStreamHandler{Templates: templateStore}
	applicants.Templates = templateStore

	apHook := &handlers.AgentPhoneWebhookHandler{
		DB:          gdb,
		Secret:      cfg.AgentPhoneWebhookSecret,
		AI:          aiClient,
		Transcripts: transcriptStore,
		StreamMode:  cfg.AgentPhoneWebhookStream,
		MaxTurns:    cfg.AgentPhoneMaxTurns,
		ToolLoopMax: cfg.AgentPhoneToolLoopMax,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Post("/webhooks/agentphone", apHook.Handle)

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

	r.Route("/api/templates", func(r chi.Router) {
		r.Use(mw.RequireUser(gdb))
		r.Get("/me", templatesHandler.GetMine)
		r.Put("/me", templatesHandler.UpdateMine)
	})

	r.Route("/api/applicants", func(r chi.Router) {
		r.Use(mw.RequireUser(gdb))
		r.Post("/", applicants.Create)
		r.Get("/", applicants.List)
		r.Get("/{id}", applicants.Get)
		r.Patch("/{id}", applicants.Update)
		r.Delete("/{id}", applicants.Delete)
		r.Post("/{id}/screen", applicants.Screen)
		r.Get("/{id}/transcript/stream", transcriptStream.Stream)
		r.Delete("/{id}/transcript", transcriptStream.Reset)
		r.Get("/{id}/template/stream", templateStream.Stream)
	})

	addr := ":" + cfg.Port
	log.Printf("go-api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
