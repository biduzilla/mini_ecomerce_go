package auth

import (
	"context"
	"errors"
	"fmt"
	"ms_auth/internal/core/domain"
	"ms_auth/internal/core/domain/apiError"
	"ms_auth/internal/core/security"
	"ms_auth/internal/core/validator"
	"ms_auth/internal/features/user"
	"time"
)

type AuthService struct {
	userService
	jwtService
}

type userService interface {
	FindByEmail(
		ctx context.Context,
		email string,
	) (*user.User, error)
}

type jwtService interface {
	GetAccessTokenExpiration() time.Duration
	CreateToken(
		user domain.UserDetails,
		tokenType security.TokenType,
	) (string, error)
	ValidateToken(
		tokenString string,
		expectedType security.TokenType,
	) (*security.TokenClaims, error)
}

type authService interface {
	Login(ctx context.Context, email, password string) (*TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, error)
}

func NewService(
	userService userService,
	jwtService jwtService,
) *AuthService {
	return &AuthService{
		userService: userService,
		jwtService:  jwtService,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	email, password string,
) (*TokenResponse, error) {
	tokenExpiration := s.jwtService.GetAccessTokenExpiration()

	v := validator.New()
	user.ValidatePasswordPlaintext(v, password)
	if !v.Valid() {
		return nil, apiError.NewValidationError(v.Errors)
	}

	user, err := s.userService.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if !user.Activated {
		return nil, apiError.ErrInactiveAccount
	}

	match, err := user.Senha.Matches(password)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, apiError.ErrInvalidCredentials
	}

	accessToken, err := s.jwtService.CreateToken(
		user,
		security.TokenTypeAccess,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	refreshToken, err := s.jwtService.CreateToken(
		user,
		security.TokenTypeRefresh,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	token := NewTokenResponse(accessToken, refreshToken, tokenExpiration)
	return &token, nil
}

func (s *AuthService) RefreshToken(
	ctx context.Context,
	refreshToken string,
) (string, error) {
	claims, err := s.ValidateToken(refreshToken, security.TokenTypeRefresh)
	if err != nil {
		return "", err
	}

	user, err := s.FindByEmail(ctx, claims.Username)
	if err != nil {
		if errors.Is(err, apiError.ErrRecordNotFound) {
			return "", apiError.ErrInvalidCredentials
		}
		return "", err
	}

	if !user.Activated {
		return "", apiError.ErrInactiveAccount
	}

	return s.CreateToken(user, security.TokenTypeRefresh)

}
