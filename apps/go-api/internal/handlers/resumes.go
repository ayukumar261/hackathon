package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	mw "github.com/ayukumar261/hackathon/go-api/internal/middleware"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxUploadBytes = 10 << 20

var allowedContentTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

type ResumesHandler struct {
	DB      *gorm.DB
	S3      *s3.Client
	Presign *s3.PresignClient
	Bucket  string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *ResumesHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !allowedContentTypes[contentType] {
		http.Error(w, "unsupported file type", http.StatusUnsupportedMediaType)
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	objectID := uuid.New()
	key := fmt.Sprintf("resumes/%s/%s%s", user.ID, objectID, ext)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if _, err := h.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(h.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}); err != nil {
		http.Error(w, "upload failed", http.StatusBadGateway)
		return
	}

	resume := models.Resume{
		ID:          objectID,
		UserID:      user.ID,
		Key:         key,
		Filename:    header.Filename,
		Size:        header.Size,
		ContentType: contentType,
	}
	if err := h.DB.Create(&resume).Error; err != nil {
		_, _ = h.S3.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(h.Bucket), Key: aws.String(key),
		})
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resume)
}

func (h *ResumesHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var resumes []models.Resume
	if err := h.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&resumes).Error; err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if resumes == nil {
		resumes = []models.Resume{}
	}
	writeJSON(w, http.StatusOK, resumes)
}

func (h *ResumesHandler) DownloadURL(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var resume models.Resume
	if err := h.DB.First(&resume, "id = ? AND user_id = ?", id, user.ID).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	req, err := h.Presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(h.Bucket),
		Key:    aws.String(resume.Key),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		http.Error(w, "presign failed", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": req.URL})
}

func (h *ResumesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var resume models.Resume
	if err := h.DB.First(&resume, "id = ? AND user_id = ?", id, user.ID).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if _, err := h.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(h.Bucket),
		Key:    aws.String(resume.Key),
	}); err != nil {
		http.Error(w, "delete failed", http.StatusBadGateway)
		return
	}
	if err := h.DB.Delete(&resume).Error; err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
