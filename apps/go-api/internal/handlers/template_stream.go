package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/templates"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TemplateStreamHandler struct {
	Templates *templates.RedisStore
}

type templateSSEEvent struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

func (h *TemplateStreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
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

	send := func(kind, content string) {
		payload, _ := json.Marshal(templateSSEEvent{Kind: kind, Content: content})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}

	if v, found, err := h.Templates.Get(ctx, applicantID); err == nil && found {
		send("snapshot", v)
	} else {
		send("snapshot", "")
	}

	updates, cancel, err := h.Templates.Subscribe(ctx, applicantID)
	if err != nil {
		fmt.Fprintf(w, ": subscribe error %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer cancel()

	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case content, ok := <-updates:
			if !ok {
				return
			}
			send("update", content)
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
