package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.R2Bucket == "" || cfg.R2Endpoint == "" {
		log.Fatal("R2_BUCKET and R2_ENDPOINT must be set")
	}

	client, presign, err := storage.NewR2Client(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Endpoint:        cfg.R2Endpoint,
	})
	if err != nil {
		log.Fatalf("r2 client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := fmt.Sprintf("smoketest/%d.txt", time.Now().UnixNano())
	payload := []byte("hello from smoketest")

	fmt.Printf("→ PutObject bucket=%s key=%s\n", cfg.R2Bucket, key)
	if _, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.R2Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String("text/plain"),
	}); err != nil {
		log.Fatalf("PutObject: %v", err)
	}
	fmt.Println("  ok")

	fmt.Println("→ PresignGetObject")
	req, err := presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.R2Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		log.Fatalf("Presign: %v", err)
	}
	fmt.Printf("  url=%s\n", req.URL)

	fmt.Println("→ GET via presigned URL")
	resp, err := http.Get(req.URL)
	if err != nil {
		log.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("GET status=%d body=%s", resp.StatusCode, body)
	}
	if !bytes.Equal(body, payload) {
		log.Fatalf("payload mismatch: got %q want %q", body, payload)
	}
	fmt.Printf("  ok (%d bytes match)\n", len(body))

	fmt.Println("→ DeleteObject")
	if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(cfg.R2Bucket),
		Key:    aws.String(key),
	}); err != nil {
		log.Fatalf("DeleteObject: %v", err)
	}
	fmt.Println("  ok")

	fmt.Println("→ GET after delete (expect 404/403)")
	resp2, err := http.Get(req.URL)
	if err != nil {
		log.Fatalf("GET: %v", err)
	}
	resp2.Body.Close()
	fmt.Printf("  status=%d\n", resp2.StatusCode)

	fmt.Println("\nR2 smoke test PASSED")
}
