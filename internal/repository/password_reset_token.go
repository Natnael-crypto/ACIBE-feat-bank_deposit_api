package repository

import (
	"context"
	"errors"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenUsed     = errors.New("token already used")
)

const PasswordResetPurpose = "password_reset"

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*dbgen.AccountToken, error)
	MarkAsUsed(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type passwordResetTokenRepository struct {
	store *Store
}

func NewPasswordResetTokenRepository(store *Store) PasswordResetTokenRepository {
	return &passwordResetTokenRepository{store: store}
}

func (r *passwordResetTokenRepository) Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	// Get user email for the token
	user, err := r.store.queries.GetUserById(ctx, userID)
	if err != nil {
		return err
	}

	expiresAtPG := pgtype.Timestamptz{
		Time:  expiresAt,
		Valid: true,
	}

	return r.store.queries.StoreToken(ctx, dbgen.StoreTokenParams{
		Token:     token,
		UserID:    userID,
		Email:     user.Email,
		Purpose:   PasswordResetPurpose,
		Used:      false,
		Revoked:   false,
		ExpiresAt: expiresAtPG,
	})
}

func (r *passwordResetTokenRepository) GetByToken(ctx context.Context, token string) (*dbgen.AccountToken, error) {
	tokenData, err := r.store.queries.GetTokenByPurpose(ctx, dbgen.GetTokenByPurposeParams{
		Token:   token,
		Purpose: PasswordResetPurpose,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	// Check if token is expired
	if tokenData.ExpiresAt.Valid && time.Now().After(tokenData.ExpiresAt.Time) {
		return nil, ErrTokenExpired
	}

	// Check if token is used
	if tokenData.Used {
		return nil, ErrTokenUsed
	}

	// Check if token is revoked
	if tokenData.Revoked {
		return nil, ErrTokenNotFound
	}

	return &tokenData, nil
}

func (r *passwordResetTokenRepository) MarkAsUsed(ctx context.Context, token string) error {
	return r.store.queries.MarkTokenAsUsed(ctx, token)
}

func (r *passwordResetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.store.queries.DeleteUserTokensByPurpose(ctx, dbgen.DeleteUserTokensByPurposeParams{
		UserID:  userID,
		Purpose: PasswordResetPurpose,
	})
}

