package main

import (
	"context"
	"flag"
	"io"
	"log"
	"path"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/db"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/ayukumar261/hackathon/go-api/internal/storage"
	"github.com/ayukumar261/hackathon/go-api/internal/supermemory"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	var only string
	flag.StringVar(&only, "applicant", "", "optional applicant ID to ingest (default: all with non-empty resume)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.SupermemoryAPIKey == "" {
		log.Fatalf("SUPERMEMORY_API_KEY not set")
	}
	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	s3Client, _, err := storage.NewR2Client(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Endpoint:        cfg.R2Endpoint,
	})
	if err != nil {
		log.Fatalf("r2: %v", err)
	}
	sm := supermemory.New(cfg.SupermemoryAPIKey)

	q := gdb.Model(&models.Applicant{}).Where("resume <> ''")
	if only != "" {
		q = q.Where("id = ?", only)
	}
	var applicants []models.Applicant
	if err := q.Find(&applicants).Error; err != nil {
		log.Fatalf("query: %v", err)
	}
	log.Printf("backfilling %d applicants", len(applicants))

	for _, a := range applicants {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(cfg.R2Bucket),
			Key:    aws.String(a.Resume),
		})
		if err != nil {
			log.Printf("r2 get %s: %v", a.ID, err)
			cancel()
			continue
		}
		data, err := io.ReadAll(obj.Body)
		obj.Body.Close()
		if err != nil {
			log.Printf("r2 read %s: %v", a.ID, err)
			cancel()
			continue
		}
		ct := "application/pdf"
		if obj.ContentType != nil && *obj.ContentType != "" {
			ct = *obj.ContentType
		}
		filename := path.Base(a.Resume)
		tag := "applicant:" + a.ID.String()
		docID, err := sm.IngestFile(ctx, tag, filename, ct, data, map[string]string{
			"applicantId": a.ID.String(),
			"resumeKey":   a.Resume,
			"name":        a.Name,
			"email":       a.Email,
		})
		cancel()
		if err != nil {
			log.Printf("ingest %s: %v", a.ID, err)
			continue
		}
		log.Printf("ok applicant=%s name=%q doc=%s", a.ID, a.Name, docID)
	}
	log.Printf("done")
}
