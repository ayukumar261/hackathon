package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/config"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB        *gorm.DB
	Cfg       *config.Config
	OAuthCfg  *oauth2.Config
}

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func randomState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *AuthHandler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	state := randomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.OAuthCfg.AuthCodeURL(state), http.StatusFound)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "oauth_state", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	token, err := h.OAuthCfg.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	client := h.OAuthCfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "userinfo failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Error(w, "userinfo decode failed", http.StatusBadGateway)
		return
	}
	if info.ID == "" {
		http.Error(w, "missing google id", http.StatusBadGateway)
		return
	}

	var user models.User
	res := h.DB.Where("google_id = ?", info.ID).First(&user)
	if res.Error == gorm.ErrRecordNotFound {
		user = models.User{
			GoogleID: info.ID,
			Email:    info.Email,
			Name:     info.Name,
			Picture:  info.Picture,
		}
		if err := h.DB.Create(&user).Error; err != nil {
			http.Error(w, "create user failed", http.StatusInternalServerError)
			return
		}
	} else if res.Error != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	} else {
		user.Email = info.Email
		user.Name = info.Name
		user.Picture = info.Picture
		h.DB.Save(&user)
	}

	session := models.Session{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := h.DB.Create(&session).Error; err != nil {
		http.Error(w, "create session failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID.String(),
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, h.Cfg.FrontendURL+"/", http.StatusFound)
}
