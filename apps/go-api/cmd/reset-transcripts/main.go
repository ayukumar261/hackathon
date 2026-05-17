package main

import (
	"context"
	"log"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/redis/go-redis/v9"
)

const matchPattern = "call:*:transcript"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse REDIS_URL: %v", err)
	}
	client := redis.NewClient(opts)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	var (
		cursor  uint64
		deleted int64
		batch   []string
	)
	for {
		keys, next, err := client.Scan(ctx, cursor, matchPattern, 500).Result()
		if err != nil {
			log.Fatalf("scan: %v", err)
		}
		batch = append(batch, keys...)
		if len(batch) >= 500 {
			n, err := client.Del(ctx, batch...).Result()
			if err != nil {
				log.Fatalf("del: %v", err)
			}
			deleted += n
			batch = batch[:0]
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if len(batch) > 0 {
		n, err := client.Del(ctx, batch...).Result()
		if err != nil {
			log.Fatalf("del: %v", err)
		}
		deleted += n
	}

	log.Printf("deleted %d transcript streams matching %q", deleted, matchPattern)
}
