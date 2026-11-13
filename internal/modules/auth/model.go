package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrSessionNotFound    = errors.New("session not found")
)

type User struct {
	ID           uuid.UUID `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type RegisterResponse struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginResponse struct {
	AccessToken           string           `json:"access_token"`
	RefreshToken          string           `json:"refresh_token"`
	TokenType             string           `json:"token_type"`
	ExpiresIn             int64            `json:"expires_in"`
	RefreshTokenExpiresIn int64            `json:"refresh_token_expires_in"`
	User                  RegisterResponse `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
