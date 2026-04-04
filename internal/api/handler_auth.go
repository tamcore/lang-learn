package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/user/lang-learn/internal/auth"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler groups authentication-related HTTP handlers.
type AuthHandler struct {
	users      store.UserStorer
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
	bcryptCost int
}

// NewAuthHandler creates an AuthHandler with the given dependencies.
func NewAuthHandler(users store.UserStorer, jwtSecret string, accessTTL, refreshTTL time.Duration, bcryptCost int) *AuthHandler {
	return &AuthHandler{
		users:      users,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		bcryptCost: bcryptCost,
	}
}

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token,omitempty"`
	User         userDTO `json:"user"`
}

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

func toUserDTO(u models.User) userDTO {
	return userDTO{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin}
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := h.users.GetByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := auth.IssueToken(h.jwtSecret, user.ID, user.IsAdmin, h.accessTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := tokenResponse{
		AccessToken: accessToken,
		User:        toUserDTO(user),
	}

	if req.RememberMe {
		refreshToken, err := auth.IssueToken(h.jwtSecret, user.ID, user.IsAdmin, h.refreshTTL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		resp.RefreshToken = refreshToken
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh handles POST /api/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	claims, err := auth.Verify(h.jwtSecret, req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	accessToken, err := auth.IssueToken(h.jwtSecret, user.ID, user.IsAdmin, h.accessTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	refreshToken, err := auth.IssueToken(h.jwtSecret, user.ID, user.IsAdmin, h.refreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserDTO(user),
	})
}

// Logout handles POST /api/auth/logout. Stateless — client discards tokens.
func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
