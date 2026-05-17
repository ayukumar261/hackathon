package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func normalizeE164(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	hasPlus := strings.HasPrefix(s, "+")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return ""
	}
	if !hasPlus && len(digits) == 10 {
		digits = "1" + digits
	}
	return "+" + digits
}

type agentPhoneCallRequest struct {
	AgentID         string `json:"agentId"`
	ToNumber        string `json:"toNumber"`
	InitialGreeting string `json:"initialGreeting,omitempty"`
}

func (h *ApplicantsHandler) Screen(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	a, err := h.loadOwned(user.ID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	toNumber := normalizeE164(a.Phone)
	if toNumber == "" {
		writeError(w, http.StatusBadRequest, "applicant has no phone number")
		return
	}
	if h.AgentPhoneAPIKey == "" || h.AgentPhoneAgentID == "" {
		writeError(w, http.StatusInternalServerError, "agentphone not configured")
		return
	}

	body, err := json.Marshal(agentPhoneCallRequest{
		AgentID:         h.AgentPhoneAgentID,
		ToNumber:        toNumber,
		InitialGreeting: "Hi " + a.Name + ", this is a screening call. Please hold.",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encode error")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://api.agentphone.ai/v1/calls", bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "request error")
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.AgentPhoneAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "agentphone request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeError(w, resp.StatusCode, "agentphone error: "+string(respBody))
		return
	}

	var parsed map[string]any
	_ = json.Unmarshal(respBody, &parsed)
	out := map[string]any{"status": "calling"}
	if id, ok := parsed["id"]; ok {
		out["callId"] = id
	}
	writeJSON(w, http.StatusOK, out)
}
