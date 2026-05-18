// Package transcripts streams AgentPhone call transcripts into Redis Streams.
//
// One stream per call, keyed by applicant ID. Writes are best-effort: failures
// are logged but do not surface to the webhook caller, so a Redis hiccup never
// 500s the agent platform.
package transcripts

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	Client *redis.Client
}

func New(client *redis.Client) *Store {
	return &Store{Client: client}
}

func streamKey(applicantID uuid.UUID) string {
	return "call:" + applicantID.String() + ":transcript"
}

func (s *Store) xadd(ctx context.Context, applicantID uuid.UUID, fields map[string]any) {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return
	}
	fields["ts"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	if err := s.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey(applicantID),
		Values: fields,
	}).Err(); err != nil {
		log.Printf("transcripts: XADD failed for %s: %v", applicantID, err)
	}
}

func (s *Store) AppendUserUtterance(ctx context.Context, applicantID uuid.UUID, text string) {
	if text == "" {
		return
	}
	s.xadd(ctx, applicantID, map[string]any{
		"role": "user",
		"kind": "utterance",
		"text": text,
	})
}

func (s *Store) AppendAssistantToken(ctx context.Context, applicantID uuid.UUID, token string) {
	if token == "" {
		return
	}
	s.xadd(ctx, applicantID, map[string]any{
		"role": "assistant",
		"kind": "token",
		"text": token,
	})
}

// Entry is one stream record returned by ReadFrom.
type Entry struct {
	ID   string
	Role string
	Kind string
	Text string
	TS   int64
}

// ReadFrom blocks up to `block` waiting for entries with ID > lastID.
// Returns the new entries and the ID of the last one (caller passes back as lastID).
// On block timeout returns (nil, lastID, nil).
func (s *Store) ReadFrom(ctx context.Context, applicantID uuid.UUID, lastID string, block time.Duration) ([]Entry, string, error) {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return nil, lastID, nil
	}
	res, err := s.Client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{streamKey(applicantID), lastID},
		Count:   500,
		Block:   block,
	}).Result()
	if err == redis.Nil {
		return nil, lastID, nil
	}
	if err != nil {
		return nil, lastID, err
	}
	var out []Entry
	newLast := lastID
	for _, stream := range res {
		for _, msg := range stream.Messages {
			e := Entry{ID: msg.ID}
			if v, ok := msg.Values["role"].(string); ok {
				e.Role = v
			}
			if v, ok := msg.Values["kind"].(string); ok {
				e.Kind = v
			}
			if v, ok := msg.Values["text"].(string); ok {
				e.Text = v
			}
			if v, ok := msg.Values["ts"].(string); ok {
				if n, perr := strconv.ParseInt(v, 10, 64); perr == nil {
					e.TS = n
				}
			}
			out = append(out, e)
			newLast = msg.ID
		}
	}
	return out, newLast, nil
}

func (s *Store) AppendAssistantTurnEnd(ctx context.Context, applicantID uuid.UUID, fullText string) {
	s.xadd(ctx, applicantID, map[string]any{
		"role": "assistant",
		"kind": "turn_end",
		"text": fullText,
	})
}

func (s *Store) Delete(ctx context.Context, applicantID uuid.UUID) error {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return nil
	}
	return s.Client.Del(ctx, streamKey(applicantID)).Err()
}

func (s *Store) AppendSubAgentInvoked(ctx context.Context, applicantID uuid.UUID, task string) {
	s.xadd(ctx, applicantID, map[string]any{
		"role": "system",
		"kind": "sub_agent_invoked",
		"text": task,
	})
}

func (s *Store) AppendSubAgentCompleted(ctx context.Context, applicantID uuid.UUID, summary string) {
	s.xadd(ctx, applicantID, map[string]any{
		"role": "system",
		"kind": "sub_agent_completed",
		"text": summary,
	})
}

// ReadRecent returns up to `count` most-recent entries (non-blocking).
func (s *Store) ReadRecent(ctx context.Context, applicantID uuid.UUID, count int64) ([]Entry, error) {
	if s == nil || s.Client == nil || applicantID == uuid.Nil {
		return nil, nil
	}
	msgs, err := s.Client.XRevRangeN(ctx, streamKey(applicantID), "+", "-", count).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		e := Entry{ID: msg.ID}
		if v, ok := msg.Values["role"].(string); ok {
			e.Role = v
		}
		if v, ok := msg.Values["kind"].(string); ok {
			e.Kind = v
		}
		if v, ok := msg.Values["text"].(string); ok {
			e.Text = v
		}
		if v, ok := msg.Values["ts"].(string); ok {
			if n, perr := strconv.ParseInt(v, 10, 64); perr == nil {
				e.TS = n
			}
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) AppendCallEnded(ctx context.Context, applicantID uuid.UUID, text string) {
	s.xadd(ctx, applicantID, map[string]any{
		"role": "system",
		"kind": "call_ended",
		"text": text,
	})
}
