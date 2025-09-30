package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"web-server/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentRepository interface {
	Upload(ctx context.Context, d *models.Document) error
	List(ctx context.Context, viewer, key, value string, limit int) ([]models.Document, error)
	GetByID(ctx context.Context, id string) (*models.Document, error)
	Delete(ctx context.Context, id string) error
}

type documentRepo struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) DocumentRepository {
	return &documentRepo{db: db}
}

func (r *documentRepo) Upload(ctx context.Context, d *models.Document) error {
	grantB, _ := json.Marshal(d.Grants)
	_, err := r.db.Exec(ctx, `
		INSERT INTO documents (id, owner, name, mime, file, public, created_at, grants, json)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, d.ID, d.Owner, d.Name, d.Mime, d.File, d.Public, d.CreatedAt, grantB, d.JSONRaw)
	return err
}

func (r *documentRepo) GetByID(ctx context.Context, id string) (*models.Document, error) {
	var d models.Document
	var grantRaw []byte
	var jsonb []byte
	err := r.db.QueryRow(ctx, `
		SELECT id, owner, name, mime, file, public, created_at, grants, json
		FROM documents WHERE id=$1
	`, id).Scan(&d.ID, &d.Owner, &d.Name, &d.Mime, &d.File, &d.Public, &d.CreatedAt, &grantRaw, &jsonb)
	if err != nil {
		return nil, err
	}
	if len(grantRaw) > 0 {
		_ = json.Unmarshal(grantRaw, &d.Grants)
	}
	d.JSONRaw = jsonb
	return &d, nil
}

func (r *documentRepo) Delete(ctx context.Context, id string) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM documents WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *documentRepo) List(ctx context.Context, viewer, key, value string, limit int) ([]models.Document, error) {
	q := `
        SELECT id, owner, name, mime, file, public, created_at, grants, json
        FROM documents
        WHERE (owner = $1 OR $1 = ANY (SELECT jsonb_array_elements_text(grants)) OR public = true)
    `
	args := []interface{}{viewer}
	i := 2

	if key != "" && value != "" {
		switch key {
		case "name", "mime":
			q += fmt.Sprintf(" AND %s = $%d", key, i)
			args = append(args, value)
			i++
		case "file":
			b := false
			if value == "true" {
				b = true
			}
			q += fmt.Sprintf(" AND file = $%d", i)
			args = append(args, b)
			i++
		case "public":
			b := false
			if value == "true" {
				b = true
			}
			q += fmt.Sprintf(" AND public = $%d", i)
			args = append(args, b)
			i++
		default:
		}
	}

	q += " ORDER BY name ASC, created_at DESC"

	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Document
	for rows.Next() {
		var d models.Document
		var grantRaw []byte
		var jsonb []byte
		if err := rows.Scan(&d.ID, &d.Owner, &d.Name, &d.Mime, &d.File, &d.Public, &d.CreatedAt, &grantRaw, &jsonb); err != nil {
			return nil, err
		}
		if len(grantRaw) > 0 {
			_ = json.Unmarshal(grantRaw, &d.Grants)
		}
		d.JSONRaw = jsonb
		out = append(out, d)
	}
	return out, nil
}
