package repository

import (
	"context"
	"errors"
	"time"
	"web-server/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, login, hash string) error
	GetByLogin(ctx context.Context, login string) (*models.User, error)
	CreateSession(ctx context.Context, token, userID string, expires time.Time) error

	GetLoginByToken(ctx context.Context, token string) (string, error)
	DeleteSession(ctx context.Context, token string) error
	GetActiveToken(ctx context.Context, userID string) (string, error)
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, login, hash string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1,$2)`, login, hash)
	return err
}

func (r *userRepo) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	var u models.User
	row := r.db.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at FROM users WHERE login=$1`, login)
	if err := row.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) CreateSession(ctx context.Context, token, userID string, expires time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO sessions (token,user_id,expires_at) VALUES ($1,$2,$3)`,
		token, userID, expires)
	return err
}

func (r *userRepo) GetLoginByToken(ctx context.Context, token string) (string, error) {
	var login string
	var expires time.Time
	err := r.db.QueryRow(ctx, `
        SELECT u.login, s.expires_at
        FROM sessions s
        JOIN users u ON u.id = s.user_id
        WHERE s.token = $1
    `, token).Scan(&login, &expires)
	if err != nil {
		return "", err
	}
	if time.Now().After(expires) {
		_, _ = r.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
		return "", errors.New("session expired")
	}
	return login, nil
}

func (r *userRepo) DeleteSession(ctx context.Context, token string) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *userRepo) GetActiveToken(ctx context.Context, userID string) (string, error) {
	var token string
	var expires time.Time
	err := r.db.QueryRow(ctx, `
        SELECT token, expires_at 
        FROM sessions 
        WHERE user_id=$1 AND expires_at > NOW()
        ORDER BY expires_at DESC
        LIMIT 1
    `, userID).Scan(&token, &expires)
	if err != nil {
		return "", err
	}
	return token, nil
}
