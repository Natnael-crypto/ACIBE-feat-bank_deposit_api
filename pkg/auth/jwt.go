package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// GenerateTokenPair generates a new access and refresh token pair
// secretKey must be provided and cannot be empty
func GenerateTokenPair(userID uuid.UUID, email, role string, secretKey string) (*TokenPair, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("JWT secret key is required and cannot be empty")
	}

	// Access token (expires in 15 minutes)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})

	accessTokenString, err := accessToken.SignedString([]byte(secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Refresh token (expires in 7 days)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})

	refreshTokenString, err := refreshToken.SignedString([]byte(secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    15 * 60, // 15 minutes in seconds
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
// secretKey must be provided and cannot be empty
func ValidateToken(tokenString string, secretKey string) (*JWTClaims, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("JWT secret key is required and cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshToken validates a refresh token and generates a new token pair
// secretKey must be provided and cannot be empty
func RefreshToken(refreshTokenString string, secretKey string) (*TokenPair, error) {
	claims, err := ValidateToken(refreshTokenString, secretKey)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	return GenerateTokenPair(claims.UserID, claims.Email, claims.Role, secretKey)
}

