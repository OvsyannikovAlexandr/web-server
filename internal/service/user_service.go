package service

import (
	"context"
	"errors"
	"regexp"
	"time"
	"web-server/internal/repository"
	"web-server/internal/util"

	"github.com/google/uuid"
)

type UserService interface {
	Register(ctx context.Context, adminToken, login, password, configToken string) error
	Auth(ctx context.Context, login, password string, ttl time.Duration) (string, error)
	ValidateToken(ctx context.Context, token string) (string, error)
	Logout(ctx context.Context, token string) error
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

var loginRe = regexp.MustCompile(`^[A-Za-z0-9]{8,}$`)
var pwUpper = regexp.MustCompile(`[A-Z]`)
var pwLower = regexp.MustCompile(`[a-z]`)
var pwDigit = regexp.MustCompile(`[0-9]`)
var pwSymbol = regexp.MustCompile(`[^A-Za-z0-9]`)

func validatePassword(p string) bool {
	return len(p) >= 8 &&
		pwUpper.MatchString(p) &&
		pwLower.MatchString(p) &&
		pwDigit.MatchString(p) &&
		pwSymbol.MatchString(p)
}

func (s *userService) Register(ctx context.Context,
	adminToken, login, password, configToken string) error {

	if adminToken != configToken {
		return errors.New("invalid admin token")
	}
	if !loginRe.MatchString(login) {
		return errors.New("login must be >=8 letters/digits")
	}
	if !validatePassword(password) {
		return errors.New("password complexity not met")
	}
	hash, err := util.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.Create(ctx, login, hash)
}

func (s *userService) Auth(ctx context.Context,
	login, password string, ttl time.Duration) (string, error) {

	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	if err := util.CheckPassword(user.PasswordHash, password); err != nil {
		return "", errors.New("invalid credentials")
	}

	if token, err := s.repo.GetActiveToken(ctx, user.ID); err == nil && token != "" {
		return token, nil
	}
	token := uuid.NewString()
	expires := time.Now().Add(ttl)
	if err := s.repo.CreateSession(ctx, token, user.ID, expires); err != nil {
		return "", err
	}
	return token, nil
}

func (s *userService) ValidateToken(ctx context.Context, token string) (string, error) {
	return s.repo.GetLoginByToken(ctx, token)
}

func (s *userService) Logout(ctx context.Context, token string) error {
	return s.repo.DeleteSession(ctx, token)
}
