package handlers

import (
	"backend/src/auth"
	"backend/src/db"
	"backend/src/models"
	"backend/src/utils"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// userResponse is the safe JSON view of a user (never exposes pass_hash).
type userResponse struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
}

// authResponse is returned by register/login.
type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

// handleRegister creates a user account and returns a JWT (auto-login).
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)

	switch {
	case !strings.Contains(req.Email, "@"):
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "valid email is required"})
		return
	case req.Username == "":
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	case len(req.Password) < 6:
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	exists, err := db.UserExists(req.Email)
	if err != nil {
		slog.Error("register: user exists check failed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "registration failed"})
		return
	}
	if exists {
		utils.WriteJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("register: hash password failed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "registration failed"})
		return
	}

	user := models.User{
		ID:             uuid.New(),
		Email:          req.Email,
		Username:       req.Username,
		HashedPassword: hash,
		CreatedAt:      time.Now(),
	}
	if err := db.InsertUser(user); err != nil {
		slog.Error("register: insert user failed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "registration failed"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		slog.Error("register: token generation failed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "registration failed"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  userResponse{ID: user.ID, Email: user.Email, Username: user.Username},
	})
}

// handleLogin verifies email + password and returns a JWT.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	user, err := db.GetUser(req.Email)
	if err != nil {
		// Generic 401: do not reveal whether the email exists.
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if !auth.VerifyPassword(user.HashedPassword, req.Password) {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		slog.Error("login: token generation failed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "login failed"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userResponse{ID: user.ID, Email: user.Email, Username: user.Username},
	})
}
