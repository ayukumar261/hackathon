package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/transcripts"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TranscriptStreamHandler struct {
	Transcripts *transcripts.Store
}

type sseEntry struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Kind string `json:"kind"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

func (h *TranscriptStreamHandler) Reset(w http.ResponseWriter, r *http.Request) {
	applicantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid applicant id", http.StatusBadRequest)
		return
	}
	if err := h.Transcripts.Delete(r.Context(), applicantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TranscriptStreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	applicantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid applicant id", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	lastID := "0"

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		entries, newLast, err := h.Transcripts.ReadFrom(ctx, applicantID, lastID, 15*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Fprintf(w, ": error %s\n\n", err.Error())
			flusher.Flush()
			time.Sleep(time.Second)
			continue
		}
		if len(entries) == 0 {
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
			continue
		}
		for _, e := range entries {
			payload, _ := json.Marshal(sseEntry{ID: e.ID, Role: e.Role, Kind: e.Kind, Text: e.Text, TS: e.TS})
			fmt.Fprintf(w, "data: %s\n\n", payload)
		}
		flusher.Flush()
		lastID = newLast
	}
}
