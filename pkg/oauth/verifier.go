package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OAuthUserInfo contains verified user information from OAuth providers
type OAuthUserInfo struct {
	ID    string
	Email string
	Name  string
	Valid bool
}

// VerifyGoogleToken verifies a Google OAuth access token and returns user info
func VerifyGoogleToken(accessToken string) (*OAuthUserInfo, error) {
	// Call Google's userinfo endpoint
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token verification failed: status %d, response: %s", resp.StatusCode, string(body))
	}

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if googleUser.ID == "" || googleUser.Email == "" {
		return nil, errors.New("invalid user info from Google")
	}

	return &OAuthUserInfo{
		ID:    googleUser.ID,
		Email: googleUser.Email,
		Name:  googleUser.Name,
		Valid: true,
	}, nil
}

// VerifyMicrosoftToken verifies a Microsoft/Azure AD OAuth access token and returns user info
func VerifyMicrosoftToken(accessToken string) (*OAuthUserInfo, error) {
	// Call Microsoft Graph API to get user info
	req, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token verification failed: status %d, response: %s", resp.StatusCode, string(body))
	}

	var msUser struct {
		ID                string `json:"id"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName       string `json:"displayName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&msUser); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Use mail if available, otherwise userPrincipalName
	email := msUser.Mail
	if email == "" {
		email = msUser.UserPrincipalName
	}

	if msUser.ID == "" || email == "" {
		return nil, errors.New("invalid user info from Microsoft")
	}

	return &OAuthUserInfo{
		ID:    msUser.ID,
		Email: email,
		Name:  msUser.DisplayName,
		Valid: true,
	}, nil
}

