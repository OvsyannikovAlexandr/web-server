package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"web-server/internal/config"
	"web-server/internal/logger"
	"web-server/internal/service"
)

type UserHandler struct {
	log     *logger.Logger
	cfg     *config.Config
	service service.UserService
}

func NewUserHandler(log *logger.Logger, cfg *config.Config, s service.UserService) *UserHandler {
	return &UserHandler{log: log, cfg: cfg, service: s}
}

// POST /api/register
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
		Login string `json:"login"`
		Pswd  string `json:"pswd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, 400, &APIResponse{Error: &APIError{Code: 400, Text: "invalid json"}})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.service.Register(ctx, req.Token, req.Login, req.Pswd, h.cfg.Server.AdminToken); err != nil {
		h.log.Error("register", "err", err)
		writeJSON(w, r, 400, &APIResponse{Error: &APIError{Code: 400, Text: err.Error()}})
		return
	}
	writeJSON(w, r, 200, &APIResponse{Response: map[string]string{"login": req.Login}})
}

// POST /api/auth
func (h *UserHandler) Auth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Login string `json:"login"`
		Pswd  string `json:"pswd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, 400, &APIResponse{Error: &APIError{Code: 400, Text: "invalid json"}})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	token, err := h.service.Auth(ctx, req.Login, req.Pswd,
		time.Duration(h.cfg.Security.TokenTTLSeconds)*time.Second)
	if err != nil {
		writeJSON(w, r, 401, &APIResponse{Error: &APIError{Code: 401, Text: err.Error()}})
		return
	}
	writeJSON(w, r, 200, &APIResponse{Response: map[string]string{"token": token}})
}

// DELETE /api/auth
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := getTokenFromHeader(r)
	if token == "" {
		writeJSON(w, r, http.StatusUnauthorized, &APIResponse{Error: &APIError{Code: 401, Text: "not authorized"}})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.service.Logout(ctx, token); err != nil {
		writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: "failed to logout"}})
		return
	}

	writeJSON(w, r, http.StatusOK, &APIResponse{Response: map[string]bool{token: true}})
}
