package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"
	"web-server/internal/models"
	"web-server/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DocumentService interface {
	CreateDocument(ctx context.Context, owner string, meta models.DocumentMeta, jsonData map[string]any) (string, error)
	ListDocuments(ctx context.Context, requester, login, key, value string, limit int) ([]models.Document, error)
	GetDocument(ctx context.Context, requester, id string) (*models.Document, string, string, map[string]any, error)
	DeleteDocument(ctx context.Context, requester, id string) error
}

type documentService struct {
	repo       repository.DocumentRepository
	cache      *redis.Client
	ttl        time.Duration
	storageDir string
}

func NewDocumentService(repo repository.DocumentRepository, cache *redis.Client, ttl time.Duration, storageDir string) DocumentService {
	return &documentService{repo: repo, cache: cache, ttl: ttl, storageDir: storageDir}
}

func cacheKey(viewer, key, value string, limit int) string {
	return fmt.Sprintf("docs:%s:%s:%s:%d", viewer, key, value, limit)
}

func (s *documentService) CreateDocument(ctx context.Context, owner string, meta models.DocumentMeta, jsonData map[string]any) (string, error) {
	id := uuid.NewString()
	doc := &models.Document{
		ID:        id,
		Owner:     owner,
		Name:      meta.Name,
		Mime:      meta.Mime,
		File:      meta.File,
		Public:    meta.Public,
		CreatedAt: time.Now(),
		Grants:    meta.Grants,
		JSONRaw:   nil,
	}
	if jsonData != nil {
		if b, err := json.Marshal(jsonData); err == nil {
			doc.JSONRaw = b
		}
	}
	if err := s.repo.Upload(ctx, doc); err != nil {
		return "", err
	}
	pattern := fmt.Sprintf("docs:%s:*", owner)
	keys, _ := s.cache.Keys(ctx, pattern).Result()
	if len(keys) > 0 {
		_, _ = s.cache.Del(ctx, keys...).Result()
	}
	return id, nil
}

func (s *documentService) ListDocuments(ctx context.Context, requester, login, key, value string, limit int) ([]models.Document, error) {
	viewer := requester
	if login != "" {
		viewer = login
	}
	k := cacheKey(viewer, key, value, limit)
	if val, err := s.cache.Get(ctx, k).Result(); err == nil {
		var docs []models.Document
		if json.Unmarshal([]byte(val), &docs) == nil {
			return docs, nil
		}
	}
	docs, err := s.repo.List(ctx, viewer, key, value, limit)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(docs); err == nil {
		_ = s.cache.Set(ctx, k, b, s.ttl).Err()
	}
	return docs, nil
}

func (s *documentService) GetDocument(ctx context.Context, requester, id string) (*models.Document, string, string, map[string]any, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", "", nil, err
	}
	allowed := false
	if d.Owner == requester {
		allowed = true
	}
	if d.Public {
		allowed = true
	}
	for _, g := range d.Grants {
		if g == requester {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, "", "", nil, errors.New("forbidden")
	}

	filePath := ""
	if d.File && d.Name != "" {
		filePath = filepath.Join(s.storageDir, d.Name)
	}

	var jsonData map[string]any
	if len(d.JSONRaw) > 0 {
		_ = json.Unmarshal(d.JSONRaw, &jsonData)
	}

	return d, filePath, d.Mime, jsonData, nil
}

func (s *documentService) DeleteDocument(ctx context.Context, requester, id string) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.Owner != requester {
		return errors.New("forbidden")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	pattern := fmt.Sprintf("docs:%s:*", d.Owner)
	keys, _ := s.cache.Keys(ctx, pattern).Result()
	if len(keys) > 0 {
		_, _ = s.cache.Del(ctx, keys...).Result()
	}
	return nil
}
