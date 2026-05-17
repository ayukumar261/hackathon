// Redis-backed live store for per-call template markdown.
//
// Acts as the source of truth during an in-flight screening call: snapshotted
// from Postgres when the call starts, mutated freely by the agent loop, and
// read by the frontend over SSE. Updates are broadcast over Pub/Sub so SSE
// subscribers can re-render in real time.
package templates

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	Client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{Client: client}
}

func valueKey(applicantID uuid.UUID) string {
	return "call:" + applicantID.String() + ":template"
}

func channel(applicantID uuid.UUID) string {
	return "call:" + applicantID.String() + ":template:updates"
}

func (s *RedisStore) Set(ctx context.Context, applicantID uuid.UUID, content string) error {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return errors.New("template store: not configured")
	}
	if err := s.Client.Set(ctx, valueKey(applicantID), content, 0).Err(); err != nil {
		return err
	}
	if err := s.Client.Publish(ctx, channel(applicantID), content).Err(); err != nil {
		log.Printf("templates: PUBLISH failed for %s: %v", applicantID, err)
	}
	return nil
}

// Get returns the current value. (value, found, error).
func (s *RedisStore) Get(ctx context.Context, applicantID uuid.UUID) (string, bool, error) {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return "", false, nil
	}
	v, err := s.Client.Get(ctx, valueKey(applicantID)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// Subscribe returns a channel of update payloads and a cancel func that closes
// the subscription. The caller MUST invoke cancel when done.
func (s *RedisStore) Subscribe(ctx context.Context, applicantID uuid.UUID) (<-chan string, func(), error) {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return nil, func() {}, errors.New("template store: not configured")
	}
	ps := s.Client.Subscribe(ctx, channel(applicantID))
	if _, err := ps.Receive(ctx); err != nil {
		_ = ps.Close()
		return nil, func() {}, err
	}
	out := make(chan string, 16)
	go func() {
		defer close(out)
		ch := ps.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				select {
				case out <- m.Payload:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	cancel := func() { _ = ps.Close() }
	return out, cancel, nil
}
