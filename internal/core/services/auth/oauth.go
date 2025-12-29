package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/pkg/auth"
	"github.com/aci/backend/pkg/oauth"
	"github.com/google/uuid"
)

type OAuthService interface {
	OAuthLogin(ctx context.Context, provider, accessToken, email, name, providerUserID string) (*auth.TokenPair, *UserInfo, error)
}

type UserInfo struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	LastName  string
	Role      string
}

type oauthService struct {
	userRepo repository.UserRepository
	cfg      config.Config
}

func NewOAuthService(userRepo repository.UserRepository, cfg config.Config) OAuthService {
	return &oauthService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *oauthService) OAuthLogin(ctx context.Context, provider, accessToken, email, name, providerUserID string) (*auth.TokenPair, *UserInfo, error) {
	// Step 1: Verify the OAuth token with the provider
	var verifiedUser *oauth.OAuthUserInfo
	var verifyErr error

	switch provider {
	case "google":
		verifiedUser, verifyErr = oauth.VerifyGoogleToken(accessToken)
	case "microsoft":
		verifiedUser, verifyErr = oauth.VerifyMicrosoftToken(accessToken)
	default:
		return nil, nil, errors.New("invalid OAuth provider. Only 'google' and 'microsoft' are supported")
	}

	if verifyErr != nil || verifiedUser == nil || !verifiedUser.Valid {
		return nil, nil, fmt.Errorf("invalid or expired OAuth token. Please try signing in again")
	}

	// Step 2: Verify the provider user ID matches
	verifiedID := verifiedUser.ID
	providedID := providerUserID

	if provider == "microsoft" {
		// Microsoft Graph API returns only the last 16 characters
		// Normalize both IDs for comparison
		providedIDNormalized := strings.ReplaceAll(providedID, "-", "")
		if len(providedIDNormalized) >= 16 {
			providedID = providedIDNormalized[len(providedIDNormalized)-16:]
		} else {
			providedID = providedIDNormalized
		}
	}

	if verifiedID != providedID {
		return nil, nil, fmt.Errorf("OAuth user ID mismatch. Expected: %s, Got: %s. Please try signing in again", verifiedUser.ID, providerUserID)
	}

	// Step 3: Verify email matches
	if verifiedUser.Email != email {
		return nil, nil, errors.New("email mismatch. The email from OAuth provider doesn't match the provided email")
	}

	// Step 4: Check if user exists by provider ID
	var user *dbgen.AccountUser
	var err error

	switch provider {
	case "google":
		user, err = s.userRepo.GetByGoogleID(ctx, verifiedUser.ID)
	case "microsoft":
		user, err = s.userRepo.GetByMicrosoftID(ctx, verifiedUser.ID)
	}

	// Step 5: If user doesn't exist by provider ID, check if user exists by email and auto-link
	if err != nil {
		if !verifiedUser.Valid {
			return nil, nil, fmt.Errorf("cannot auto-link account: OAuth token verification failed. Please try signing in again")
		}

		verifiedEmail := verifiedUser.Email
		if verifiedEmail == "" {
			return nil, nil, fmt.Errorf("cannot auto-link account: verified email is empty")
		}

		// Check if user exists by verified email
		existingUser, emailErr := s.userRepo.GetByEmail(ctx, verifiedEmail)
		if emailErr == nil && existingUser != nil {
			// User exists with this verified email - auto-link the OAuth provider
			verifiedProviderID := verifiedUser.ID

			// For Microsoft, use the full UUID format from frontend for storage
			if provider == "microsoft" {
				providedIDNormalized := strings.ReplaceAll(providerUserID, "-", "")
				if len(providedIDNormalized) == 32 {
					verifiedProviderID = providedIDNormalized
				}
			}

			existingGoogleID := ""
			if existingUser.GoogleID.Valid {
				existingGoogleID = existingUser.GoogleID.String
			}
			existingMicrosoftID := ""
			if existingUser.MicrosoftID.Valid {
				existingMicrosoftID = existingUser.MicrosoftID.String
			}

			if provider == "google" && existingGoogleID == "" {
				// Double-check: Verify Google ID is not already used by another account
				existingGoogleUser, _ := s.userRepo.GetByGoogleID(ctx, verifiedProviderID)
				if existingGoogleUser != nil {
					return nil, nil, fmt.Errorf("this Google account is already linked to another user")
				}

				// Auto-link Google account
				_, updateErr := s.userRepo.UpdateOAuthIDs(ctx, existingUser.ID, &verifiedProviderID, nil)
				if updateErr != nil {
					return nil, nil, fmt.Errorf("failed to link Google account: %v", updateErr)
				}
				user = existingUser
			} else if provider == "microsoft" && existingMicrosoftID == "" {
				// Double-check: Verify Microsoft ID is not already used by another account
				existingMicrosoftUser, _ := s.userRepo.GetByMicrosoftID(ctx, verifiedProviderID)
				if existingMicrosoftUser != nil {
					return nil, nil, fmt.Errorf("this Microsoft account is already linked to another user")
				}

				// Auto-link Microsoft account
				_, updateErr := s.userRepo.UpdateOAuthIDs(ctx, existingUser.ID, nil, &verifiedProviderID)
				if updateErr != nil {
					return nil, nil, fmt.Errorf("failed to link Microsoft account: %v", updateErr)
				}
				user = existingUser
			} else {
				return nil, nil, fmt.Errorf("account with email %s is already linked to %s, but provider ID doesn't match. Please contact an administrator", verifiedEmail, provider)
			}
		} else {
			// User doesn't exist at all
			return nil, nil, fmt.Errorf("no account found with this %s account. Please contact an administrator to create an account", provider)
		}
	}

	// Step 6: Verify email matches
	if user.Email != verifiedUser.Email {
		return nil, nil, errors.New("email mismatch. The account email doesn't match the OAuth email")
	}

	// Step 7: Check if user is active
	if user.AccountStatus != "active" {
		return nil, nil, errors.New("your account has been deactivated. Please contact an administrator")
	}

	// Step 8: Generate tokens
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Email, user.Role, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil, errors.New("failed to generate authentication tokens. Please try again")
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

