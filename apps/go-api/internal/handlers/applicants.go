package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/ayukumar261/hackathon/go-api/internal/templates"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicantsHandler struct {
	DB                *gorm.DB
	AgentPhoneAPIKey  string
	AgentPhoneAgentID string
	Templates         *templates.RedisStore
}

type applicantCreateInput struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Resume     string `json:"resume"`
	PositionID string `json:"positionId"`
}

type applicantUpdateInput struct {
	Name       *string `json:"name"`
	Email      *string `json:"email"`
	Phone      *string `json:"phone"`
	Resume     *string `json:"resume"`
	PositionID *string `json:"positionId"`
}

func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}

func (h *ApplicantsHandler) ownsPosition(userID, positionID uuid.UUID) (bool, error) {
	var p models.Position
	err := h.DB.Select("id").First(&p, "id = ? AND user_id = ?", positionID, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (h *ApplicantsHandler) loadOwned(userID, id uuid.UUID) (*models.Applicant, error) {
	var a models.Applicant
	err := h.DB.
		Joins("JOIN positions ON positions.id = applicants.position_id").
		Where("applicants.id = ? AND positions.user_id = ?", id, userID).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (h *ApplicantsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in applicantCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Resume = strings.TrimSpace(in.Resume)
	in.PositionID = strings.TrimSpace(in.PositionID)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if in.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if in.Phone == "" {
		writeError(w, http.StatusBadRequest, "phone is required")
		return
	}
	if in.PositionID == "" {
		writeError(w, http.StatusBadRequest, "positionId is required")
		return
	}
	positionID, err := uuid.Parse(in.PositionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid positionId")
		return
	}
	owns, err := h.ownsPosition(user.ID, positionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if !owns {
		writeError(w, http.StatusNotFound, "position not found")
		return
	}
	a := models.Applicant{
		PositionID: positionID,
		Name:       in.Name,
		Email:      in.Email,
		Phone:      in.Phone,
		Resume:     in.Resume,
	}
	if err := h.DB.Create(&a).Error; err != nil {
		if isDuplicateErr(err) {
			writeError(w, http.StatusConflict, "email or phone already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *ApplicantsHandler) List(w http.ResponseWriter, r *http.Request) {
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
	q := h.DB.Model(&models.Applicant{}).
		Joins("JOIN positions ON positions.id = applicants.position_id").
		Where("positions.user_id = ?", user.ID)

	if v := r.URL.Query().Get("positionId"); v != "" {
		positionID, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid positionId")
			return
		}
		owns, err := h.ownsPosition(user.ID, positionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		if !owns {
			writeError(w, http.StatusNotFound, "position not found")
			return
		}
		q = q.Where("applicants.position_id = ?", positionID)
	}

	var applicants []models.Applicant
	if err := q.Order("applicants.created_at DESC").
		Limit(limit).Offset(offset).
		Find(&applicants).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if applicants == nil {
		applicants = []models.Applicant{}
	}
	writeJSON(w, http.StatusOK, applicants)
}

func (h *ApplicantsHandler) Get(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, a)
}

func (h *ApplicantsHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	var in applicantUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	updates := map[string]any{}
	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" {
			writeError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
		updates["name"] = v
	}
	if in.Email != nil {
		v := strings.TrimSpace(*in.Email)
		if v == "" {
			writeError(w, http.StatusBadRequest, "email cannot be empty")
			return
		}
		if _, err := mail.ParseAddress(v); err != nil {
			writeError(w, http.StatusBadRequest, "invalid email")
			return
		}
		updates["email"] = v
	}
	if in.Phone != nil {
		v := strings.TrimSpace(*in.Phone)
		if v == "" {
			writeError(w, http.StatusBadRequest, "phone cannot be empty")
			return
		}
		updates["phone"] = v
	}
	if in.Resume != nil {
		updates["resume"] = strings.TrimSpace(*in.Resume)
	}
	if in.PositionID != nil {
		v := strings.TrimSpace(*in.PositionID)
		positionID, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid positionId")
			return
		}
		owns, err := h.ownsPosition(user.ID, positionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		if !owns {
			writeError(w, http.StatusNotFound, "position not found")
			return
		}
		updates["position_id"] = positionID
	}
	if len(updates) == 0 {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	if err := h.DB.Model(a).Updates(updates).Error; err != nil {
		if isDuplicateErr(err) {
			writeError(w, http.StatusConflict, "email or phone already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *ApplicantsHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	if err := h.DB.Delete(a).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
