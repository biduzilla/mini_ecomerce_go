package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"ms_auth/internal/core/domain"
	"ms_auth/internal/core/domain/apiError"
	"ms_auth/internal/core/security"
	"ms_auth/internal/features/user"

	"github.com/google/uuid"
)

type mockUserService struct {
	findByEmailFn func(ctx context.Context, email string) (*user.User, error)
}

func (m *mockUserService) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return m.findByEmailFn(ctx, email)
}

type mockJwtService struct {
	getAccessExpFn  func() time.Duration
	createTokenFn   func(user domain.UserDetails, tokenType security.TokenType) (string, error)
	validateTokenFn func(tokenString string, expectedType security.TokenType) (*security.TokenClaims, error)
}

func (m *mockJwtService) GetAccessTokenExpiration() time.Duration {
	return m.getAccessExpFn()
}

func (m *mockJwtService) CreateToken(user domain.UserDetails, tokenType security.TokenType) (string, error) {
	return m.createTokenFn(user, tokenType)
}

func (m *mockJwtService) ValidateToken(tokenString string, expectedType security.TokenType) (*security.TokenClaims, error) {
	return m.validateTokenFn(tokenString, expectedType)
}

func TestLogin_InvalidPasswordFormat_ShortCircuit(t *testing.T) {
	jwtMock := &mockJwtService{
		getAccessExpFn: func() time.Duration {
			return 0
		},
	}

	svc := NewService(&mockUserService{}, jwtMock)

	_, err := svc.Login(context.Background(), "test@test.com", "123")

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}
	var valErr *apiError.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}

func TestLogin_UserNotFound_ShortCircuit(t *testing.T) {
	userMock := &mockUserService{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return nil, apiError.ErrRecordNotFound
		},
	}
	jwtMock := &mockJwtService{
		getAccessExpFn: func() time.Duration {
			return 0
		},
	}

	svc := NewService(userMock, jwtMock)

	_, err := svc.Login(context.Background(), "test@test.com", "12345678")

	if !errors.Is(err, apiError.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

func TestLogin_InactiveAccount_ShortCircuit(t *testing.T) {
	fakeInactiveUser := createFakeUser(true, false)

	userMock := &mockUserService{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return fakeInactiveUser, nil
		},
	}
	jwtMock := &mockJwtService{
		getAccessExpFn: func() time.Duration {
			return 0
		},
	}

	svc := NewService(userMock, jwtMock)

	_, err := svc.Login(context.Background(), "test@test.com", "12345678")

	if !errors.Is(err, apiError.ErrInactiveAccount) {
		t.Errorf("Expected ErrInactiveAccount, got %v", err)
	}
}

func TestLogin_WrongPassword_ShortCircuit(t *testing.T) {
	fakeUser := createFakeUser(true, true)

	userMock := &mockUserService{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return fakeUser, nil
		},
	}
	jwtMock := &mockJwtService{
		getAccessExpFn: func() time.Duration {
			return 0
		},
	}

	svc := NewService(userMock, jwtMock)

	_, err := svc.Login(context.Background(), "test@test.com", "senha_errada")

	if !errors.Is(err, apiError.ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	fakeUser := createFakeUser(true, true)
	mockAccess := "access_token_mock"
	mockRefresh := "refresh_token_mock"
	expTime := 1 * time.Hour
	expectedTime := time.Now().Add(expTime)

	userMock := &mockUserService{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return fakeUser, nil
		},
	}
	jwtMock := &mockJwtService{
		getAccessExpFn: func() time.Duration { return expTime },
		createTokenFn: func(user domain.UserDetails, tokenType security.TokenType) (string, error) {
			if tokenType == security.TokenTypeAccess {
				return mockAccess, nil
			}
			if tokenType == security.TokenTypeRefresh {
				return mockRefresh, nil
			}
			return "", nil
		},
	}

	svc := NewService(userMock, jwtMock)

	resp, err := svc.Login(context.Background(), "test@test.com", "12345678")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.AccessToken != mockAccess {
		t.Error("Access token mismatch")
	}
	if resp.RefreshToken != mockRefresh {
		t.Error("Refresh token mismatch")
	}

	diff := resp.ExpiresAt.Sub(expectedTime)
	if diff < 0 {
		diff = -diff
	}

	if diff > time.Second {
		t.Errorf("Expiration time mismatch. Diff: %v", diff)
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	userMock := &mockUserService{}
	jwtMock := &mockJwtService{
		validateTokenFn: func(tokenString string, expectedType security.TokenType) (*security.TokenClaims, error) {
			return nil, apiError.ErrTokenExpired
		},
	}

	svc := NewService(userMock, jwtMock)

	_, err := svc.RefreshToken(context.Background(), "token_invalido")

	if !errors.Is(err, apiError.ErrTokenExpired) {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestRefreshToken_UserNotFoundBecomesInvalidCredentials(t *testing.T) {
	jwtMock := &mockJwtService{
		validateTokenFn: func(tokenString string, expectedType security.TokenType) (*security.TokenClaims, error) {
			return &security.TokenClaims{Username: "test@test.com"}, nil
		},
	}
	userMock := &mockUserService{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return nil, apiError.ErrRecordNotFound
		},
	}

	svc := NewService(userMock, jwtMock)

	_, err := svc.RefreshToken(context.Background(), "token_valido")

	if !errors.Is(err, apiError.ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials for masked not found, got %v", err)
	}
}

func createFakeUser(matchPassword bool, activated bool) *user.User {
	id := uuid.New()

	fakeUser := &user.User{
		ID:        id,
		Email:     "test@test.com",
		Activated: activated,
	}

	if matchPassword {
		fakeUser.Senha.Set("12345678")
	} else {
		fakeUser.Senha.Set("senha_errada")
	}

	return fakeUser
}
