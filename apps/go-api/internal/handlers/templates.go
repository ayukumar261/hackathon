package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/ayukumar261/hackathon/go-api/internal/templates"
	"gorm.io/gorm"
)

type TemplatesHandler struct {
	DB *gorm.DB
}

func (h *TemplatesHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var t models.Template
	if err := h.DB.Where("user_id = ?", user.ID).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t = models.Template{UserID: user.ID, Content: templates.DefaultContent}
			if err := h.DB.Create(&t).Error; err != nil {
				http.Error(w, "create template failed", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (h *TemplatesHandler) UpdateMine(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var t models.Template
	err := h.DB.Where("user_id = ?", user.ID).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t = models.Template{UserID: user.ID, Content: body.Content}
		if err := h.DB.Create(&t).Error; err != nil {
			http.Error(w, "create failed", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	} else {
		t.Content = body.Content
		if err := h.DB.Save(&t).Error; err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}
