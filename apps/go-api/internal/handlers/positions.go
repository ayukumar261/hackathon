package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PositionsHandler struct {
	DB *gorm.DB
}

type positionCreateInput struct {
	Title       string  `json:"title"`
	Company     *string `json:"company"`
	Description string  `json:"description"`
}

type positionUpdateInput struct {
	Title       *string `json:"title"`
	Company     *string `json:"company"`
	Description *string `json:"description"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *PositionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in positionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if in.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}
	company := "Helix Compute"
	if in.Company != nil {
		if c := strings.TrimSpace(*in.Company); c != "" {
			company = c
		}
	}
	p := models.Position{
		UserID:      user.ID,
		Title:       in.Title,
		Company:     company,
		Description: in.Description,
	}
	if err := h.DB.Create(&p).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *PositionsHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		if n > 100 {
			n = 100
		}
		limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = n
	}
	var positions []models.Position
	if err := h.DB.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&positions).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if positions == nil {
		positions = []models.Position{}
	}
	writeJSON(w, http.StatusOK, positions)
}

func (h *PositionsHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	var p models.Position
	if err := h.DB.First(&p, "id = ? AND user_id = ?", id, user.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PositionsHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var p models.Position
	if err := h.DB.First(&p, "id = ? AND user_id = ?", id, user.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	var in positionUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	updates := map[string]any{}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		updates["title"] = t
	}
	if in.Company != nil {
		c := strings.TrimSpace(*in.Company)
		if c == "" {
			writeError(w, http.StatusBadRequest, "company cannot be empty")
			return
		}
		updates["company"] = c
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if d == "" {
			writeError(w, http.StatusBadRequest, "description cannot be empty")
			return
		}
		updates["description"] = d
	}
	if len(updates) == 0 {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	if err := h.DB.Model(&p).Updates(updates).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PositionsHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	var p models.Position
	if err := h.DB.First(&p, "id = ? AND user_id = ?", id, user.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := h.DB.Delete(&p).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
