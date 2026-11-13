package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const accessTokenDuration = 15 * time.Minute

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) Register(
	ctx context.Context,
	req RegisterRequest,
) (*RegisterResponse, error) {

	fullName := strings.TrimSpace(req.FullName)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if len(fullName) < 2 || len(fullName) > 100 {
		return nil, fmt.Errorf("full name must be between 2 and 100 characters")
	}

	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	if len(password) < 8 || len(password) > 72 {
		return nil, fmt.Errorf("password must be between 8 and 72 bytes")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &User{
		FullName:     fullName,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return &RegisterResponse{
		ID:        user.ID,
		FullName:  user.FullName,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *Service) Login(
	ctx context.Context,
	req LoginRequest,
) (*LoginResponse, error) {

	// 1. Normalize email.
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// 2. Retrieve user from PostgreSQL.
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("retrieve user: %w", err)
	}

	// 3. Verify submitted password against stored bcrypt hash.
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 4. Check account status.
	if user.Status != "ACTIVE" {
		return nil, ErrAccountInactive
	}

	// 5. Generate JWT access token.
	now := time.Now()
	expiresAt := now.Add(accessTokenDuration)

	claims := jwt.RegisteredClaims{
		Issuer:    "tradex",
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// 6. Return safe user information and token.
	return &LoginResponse{
		AccessToken: signedToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(accessTokenDuration.Seconds()),
		User: RegisterResponse{
			ID:        user.ID,
			FullName:  user.FullName,
			Email:     user.Email,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}
