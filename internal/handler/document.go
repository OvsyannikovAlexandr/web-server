package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"web-server/internal/models"
	"web-server/internal/service"

	"github.com/gorilla/mux"
)

type DocumentHandler struct {
	svc         service.DocumentService
	userService service.UserService
}

func NewDocumentHandler(svc service.DocumentService, us service.UserService) *DocumentHandler {
	return &DocumentHandler{svc: svc, userService: us}
}

// UploadDoc (POST /api/docs)
func (h *DocumentHandler) UploadDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, r, http.StatusMethodNotAllowed, &APIResponse{Error: &APIError{Code: 405, Text: "method not allowed"}})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, r, http.StatusBadRequest, &APIResponse{Error: &APIError{Code: 400, Text: "invalid multipart"}})
		return
	}

	token := getTokenFromHeader(r)

	metaRaw := r.FormValue("meta")
	if metaRaw == "" {
		writeJSON(w, r, http.StatusBadRequest, &APIResponse{Error: &APIError{Code: 400, Text: "meta required"}})
		return
	}

	var meta models.DocumentMeta
	if err := json.Unmarshal([]byte(metaRaw), &meta); err != nil {
		writeJSON(w, r, http.StatusBadRequest, &APIResponse{Error: &APIError{Code: 400, Text: "invalid meta"}})
		return
	}

	userLogin, err := h.userService.ValidateToken(r.Context(), token)
	if err != nil {
		writeJSON(w, r, http.StatusUnauthorized, &APIResponse{Error: &APIError{Code: 401, Text: "unauthorized"}})
		return
	}

	var jsonData map[string]any
	if j := r.FormValue("json"); j != "" {
		if err := json.Unmarshal([]byte(j), &jsonData); err != nil {
			writeJSON(w, r, http.StatusBadRequest, &APIResponse{Error: &APIError{Code: 400, Text: "invalid json"}})
			return
		}
	}

	var fileName string
	if meta.File {
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, r, http.StatusBadRequest, &APIResponse{Error: &APIError{Code: 400, Text: "file required"}})
			return
		}
		defer file.Close()

		if err := os.MkdirAll("uploads", 0755); err != nil {
			writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: "storage error"}})
			return
		}
		dst := filepath.Join("uploads", meta.Name)
		out, err := os.Create(dst)
		if err != nil {
			writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: "storage error"}})
			return
		}
		defer out.Close()
		if _, err := io.Copy(out, file); err != nil {
			writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: "storage error"}})
			return
		}
		fileName = meta.Name
	}

	docID, err := h.svc.CreateDocument(r.Context(), userLogin, meta, jsonData)
	if err != nil {
		writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: "create error"}})
		return
	}

	writeJSON(w, r, http.StatusOK, &APIResponse{
		Data: map[string]any{
			"json": jsonData,
			"file": fileName,
			"id":   docID,
		},
	})
}

// ListDocs (GET|HEAD /api/docs)
func (h *DocumentHandler) ListDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, r, http.StatusMethodNotAllowed, &APIResponse{Error: &APIError{Code: 405, Text: "method not allowed"}})
		return
	}

	token := getTokenFromHeader(r)
	userLogin, err := h.userService.ValidateToken(r.Context(), token)
	if err != nil {
		writeJSON(w, r, http.StatusUnauthorized, &APIResponse{
			Error: &APIError{Code: 401, Text: "unauthorized"},
		})
		return
	}

	loginFilter := r.URL.Query().Get("login")
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	docs, err := h.svc.ListDocuments(r.Context(), userLogin, loginFilter, key, value, limit)
	if err != nil {
		writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: "list error"}})
		return
	}

	answers := []Answer{}
	for _, doc := range docs {
		answer := Answer{
			ID:      doc.ID,
			Name:    doc.Name,
			Mime:    doc.Mime,
			File:    doc.File,
			Public:  doc.Public,
			Created: doc.CreatedAt.Format(time.DateTime),
			Grant:   doc.Grants,
		}
		if len(doc.JSONRaw) > 0 {
			err := json.Unmarshal(doc.JSONRaw, &answer.Json)
			if err != nil {
				writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: err.Error()}})
			}
		}
		answers = append(answers, answer)

	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	writeJSON(w, r, http.StatusOK, &APIResponse{
		Data: map[string]any{"docs": answers},
	})
}

// GetDoc (GET|HEAD /api/docs/{id})
func (h *DocumentHandler) GetDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, r, http.StatusMethodNotAllowed, &APIResponse{Error: &APIError{Code: 405, Text: "method not allowed"}})
		return
	}

	id := mux.Vars(r)["id"]
	token := getTokenFromHeader(r)

	userLogin, err := h.userService.ValidateToken(r.Context(), token)
	if err != nil {
		writeJSON(w, r, http.StatusUnauthorized, &APIResponse{Error: &APIError{Code: 401, Text: "unauthorized"}})
		return
	}

	doc, filePath, mime, jsonData, err := h.svc.GetDocument(r.Context(), userLogin, id)
	if err != nil {
		if err.Error() == "forbidden" {
			writeJSON(w, r, http.StatusForbidden, &APIResponse{Error: &APIError{Code: 403, Text: "access denied"}})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: err.Error()}})
		return
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	if doc.File && filePath != "" {
		w.Header().Set("Content-Type", mime)
		http.ServeFile(w, r, filePath)
		return
	}

	writeJSON(w, r, http.StatusOK, &APIResponse{Data: jsonData})
}

// DeleteDoc (DELETE /api/docs/{id})
func (h *DocumentHandler) DeleteDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, r, http.StatusMethodNotAllowed, &APIResponse{Error: &APIError{Code: 405, Text: "method not allowed"}})
		return
	}

	id := mux.Vars(r)["id"]
	token := getTokenFromHeader(r)

	userLogin, err := h.userService.ValidateToken(r.Context(), token)
	if err != nil {
		writeJSON(w, r, http.StatusUnauthorized, &APIResponse{Error: &APIError{Code: 401, Text: "unauthorized"}})
		return
	}

	if err := h.svc.DeleteDocument(r.Context(), userLogin, id); err != nil {
		if err.Error() == "forbidden" {
			writeJSON(w, r, http.StatusForbidden, &APIResponse{Error: &APIError{Code: 403, Text: "cannot delete"}})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, &APIResponse{Error: &APIError{Code: 500, Text: err.Error()}})
		return
	}

	writeJSON(w, r, http.StatusOK, &APIResponse{
		Response: map[string]bool{id: true},
	})
}
