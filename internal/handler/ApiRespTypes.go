package handler

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Answer struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Mime    string         `json:"mime,omitempty"`
	File    bool           `json:"file"`
	Public  bool           `json:"public"`
	Created string         `json:"created"`
	Grant   []string       `json:"grant"`
	Json    map[string]any `json:"json,omitempty"`
}
type APIError struct {
	Code int    `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}
type APIResponse struct {
	Error    *APIError   `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, payload *APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func getTokenFromHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.Fields(auth)
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}
