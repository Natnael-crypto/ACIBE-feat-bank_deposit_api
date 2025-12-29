package auth

import (
	"context"
	"errors"

	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/auth"
)

// UserInfo is defined in oauth.go to avoid duplication

type Service interface {
	Login(ctx context.Context, email, password string) (*auth.TokenPair, *UserInfo, error)
	ValidateToken(ctx context.Context, tokenString string) (*auth.JWTClaims, error)
}

type service struct {
	userRepo repository.UserRepository
	cfg      config.Config
}

func NewService(userRepo repository.UserRepository, cfg config.Config) Service {
	return &service{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *service) Login(ctx context.Context, email, password string) (*auth.TokenPair, *UserInfo, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := auth.VerifyPassword(string(user.UserPassword), password); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if user.AccountStatus != "active" {
		return nil, nil, errors.New("user account is inactive")
	}

	// Generate token pair
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Email, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil, errors.New("failed to generate token")
	}

	userInfo := &UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.Fname,
		LastName:  user.Lname,
		Role:      user.Role,
	}

	return tokenPair, userInfo, nil
}

func (s *service) ValidateToken(ctx context.Context, tokenString string) (*auth.JWTClaims, error) {
	return auth.ValidateToken(tokenString, s.cfg.JWTSecret)
}

