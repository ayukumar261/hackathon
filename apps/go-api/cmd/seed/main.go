package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/db"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/ayukumar261/hackathon/go-api/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	applicantName  = "Ayush Kumar"
	applicantEmail = "ayukumar261@gmail.com"
	applicantPhone = "+1 (945) 241-2002"
)

func expandHome(p string) (string, error) {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	return p, nil
}

const forwardDeployedDesc = "# **Forward Deployed ML Engineer**\n" +
	"**Helix Compute** — Bay Area (frequent customer interaction)\n" +
	"**Team:** Applied Inference & Customer Engineering\n\n" +
	"## About Helix\n" +
	"Helix Compute provides high-performance GPU infrastructure for frontier AI labs, ML-first startups, and enterprise AI teams. We run large H100/H200/B200 clusters and a full serving + post-training stack on top of them. Our customers ship some of the most demanding AI workloads in production today—and we work shoulder-to-shoulder with them to make it happen.\n\n" +
	"## About the Role\n" +
	"We're hiring a Forward Deployed ML Engineer to embed with Helix customers during their first 90 days on the platform—turning ambitious model launches into production-grade deployments on our clusters.\n\n" +
	"You'll be the technical face of Helix in the room when a customer is debugging a flaky multi-node training run at 2am, or trying to squeeze 30% more throughput out of their inference stack before a product launch. This is a hybrid role that sits at the intersection of platform engineering, applied ML, and customer success.\n\n" +
	"If you enjoy being close to users, debugging real systems, and shipping results fast (not just writing docs), this role is for you.\n\n" +
	"## What You'll Do\n" +
	"**Own customer POCs end-to-end**\n" +
	"- Architect and tune deployments using vLLM, SGLang, TensorRT-LLM, and Helix's serving runtime\n" +
	"- Translate customer requirements into concrete system designs and experiments\n\n" +
	"**Forward-deploy with customers**\n" +
	"- Work hands-on with research teams, startups, and enterprise customers\n" +
	"- Debug performance, stability, and correctness issues in real environments\n\n" +
	"**Performance & reliability**\n" +
	"- Profile distributed training and inference jobs across multi-GPU / multi-node setups\n" +
	"- Diagnose bottlenecks across NCCL, InfiniBand, and storage layers\n\n" +
	"**Feedback loop to product**\n" +
	"- Build reference implementations, cookbooks, and customer-facing SDK examples\n" +
	"- Channel field learnings into the Helix platform roadmap and tooling\n\n" +
	"## What We're Looking For\n" +
	"**Core Requirements**\n" +
	"- Strong software engineering background (Python required; Go / Rust a plus)\n" +
	"- Production experience with LLM inference or large-scale training\n" +
	"- Familiarity with distributed systems and GPUs (multi-GPU, multi-node)\n" +
	"- Comfort debugging across the full stack: code, infra, networking, performance\n" +
	"- High agency, low ego, fast iteration\n\n" +
	"**Nice to Have**\n" +
	"- Prior FDE / Solutions Engineer / Applied Research Engineer experience\n" +
	"- LLM inference frameworks (vLLM, SGLang, Ray Serve, Triton)\n" +
	"- Kernel-level GPU optimization (Triton, CUTLASS)\n" +
	"- RLHF, DPO, GRPO pipeline experience\n" +
	"- Kubernetes-based ML platforms\n\n" +
	"## Why Helix\n" +
	"- You're close to real users and real GPUs—not abstract roadmaps\n" +
	"- You'll work on cutting-edge inference and training workloads, not toy demos\n" +
	"- You'll influence product direction through direct customer feedback\n" +
	"- Fast iteration, high ownership, and visible impact\n\n" +
	"## Who Thrives Here\n" +
	"- Engineers who like shipping over theorizing\n" +
	"- People who enjoy being the \"last mile\" problem solver\n" +
	"- Builders who want exposure to both deep systems and applied ML\n" +
	"- Those excited by early-stage POCs that turn into real production systems\n"

func main() {
	email := flag.String("email", "ayukumar261@gmail.com", "email of the user to own the seeded positions")
	resumePath := flag.String("resume", "~/Desktop/Resume_04_26_2026.pdf", "path to the resume PDF to upload to R2 and link to the seeded applicant")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	var user models.User
	if err := gdb.Where("email = ?", *email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Fatalf("user with email %q not found — sign in via Google OAuth first", *email)
		}
		log.Fatalf("lookup user: %v", err)
	}

	positions := []models.Position{
		{UserID: user.ID, Title: "Software Engineer", Description: forwardDeployedDesc},
	}

	inserted, skipped := 0, 0
	var softwareEng models.Position
	for _, p := range positions {
		var existing models.Position
		err := gdb.Where("user_id = ? AND title = ?", p.UserID, p.Title).First(&existing).Error
		if err == nil {
			skipped++
			if existing.Title == "Software Engineer" {
				softwareEng = existing
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Fatalf("check existing %q: %v", p.Title, err)
		}
		if err := gdb.Create(&p).Error; err != nil {
			log.Fatalf("insert %q: %v", p.Title, err)
		}
		if p.Title == "Software Engineer" {
			softwareEng = p
		}
		inserted++
	}

	applicantStatus := seedApplicant(gdb, cfg, user, softwareEng, *resumePath)

	log.Printf("seed complete for %s: positions inserted %d, skipped %d; applicant %s", user.Email, inserted, skipped, applicantStatus)
}

func seedApplicant(gdb *gorm.DB, cfg *config.Config, user models.User, position models.Position, resumePath string) string {
	if position.ID == uuid.Nil {
		log.Fatalf("software engineer position not available; cannot seed applicant")
	}

	var existing models.Applicant
	err := gdb.Where("email = ?", applicantEmail).First(&existing).Error
	if err == nil {
		return "skipped (already exists)"
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Fatalf("check existing applicant: %v", err)
	}

	expanded, err := expandHome(resumePath)
	if err != nil {
		log.Fatalf("expand resume path: %v", err)
	}
	f, err := os.Open(expanded)
	if err != nil {
		log.Fatalf("open resume %q: %v", expanded, err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		log.Fatalf("stat resume: %v", err)
	}

	if cfg.R2Bucket == "" || cfg.R2Endpoint == "" {
		log.Fatalf("R2 config missing (R2_BUCKET / R2_ENDPOINT)")
	}
	s3Client, _, err := storage.NewR2Client(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Endpoint:        cfg.R2Endpoint,
	})
	if err != nil {
		log.Fatalf("r2 client: %v", err)
	}

	filename := filepath.Base(expanded)
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".pdf"
	}
	objectID := uuid.New()
	key := fmt.Sprintf("resumes/%s/%s%s", user.ID, objectID, ext)
	contentType := "application/pdf"

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.R2Bucket),
		Key:         aws.String(key),
		Body:        f,
		ContentType: aws.String(contentType),
	}); err != nil {
		log.Fatalf("r2 upload: %v", err)
	}

	resume := models.Resume{
		ID:          objectID,
		UserID:      user.ID,
		Key:         key,
		Filename:    filename,
		Size:        stat.Size(),
		ContentType: contentType,
	}
	if err := gdb.Create(&resume).Error; err != nil {
		log.Fatalf("insert resume row: %v", err)
	}

	applicant := models.Applicant{
		PositionID: position.ID,
		Name:       applicantName,
		Email:      applicantEmail,
		Phone:      applicantPhone,
		Resume:     key,
	}
	if err := gdb.Create(&applicant).Error; err != nil {
		log.Fatalf("insert applicant: %v", err)
	}
	return fmt.Sprintf("inserted (resume key=%s)", key)
}
