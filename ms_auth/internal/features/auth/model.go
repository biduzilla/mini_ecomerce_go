package auth

import "time"

type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func NewTokenResponse(access, refresh string, expiresIn time.Duration) TokenResponse {
	return TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(expiresIn),
	}
}
