package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionHandler struct {
	DB *gorm.DB
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: "session_id", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_id")
	if err != nil || c.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sid, err := uuid.Parse(c.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var session models.Session
	if err := h.DB.First(&session, "id = ?", sid).Error; err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if time.Now().After(session.ExpiresAt) {
		h.DB.Delete(&session)
		clearSessionCookie(w)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", session.UserID).Error; err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"user": user})
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session_id"); err == nil && c.Value != "" {
		if sid, err := uuid.Parse(c.Value); err == nil {
			h.DB.Delete(&models.Session{}, "id = ?", sid)
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
