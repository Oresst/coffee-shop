package service

import (
	"errors"
	"fmt"
	domain "github.com/yourusername/user-service/internal/domains"
	"github.com/yourusername/user-service/internal/repository/postgres"
	"github.com/yourusername/user-service/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo       postgres.UserRepositoryInt
	jwtManager *jwt.JWTManager
}

func NewAuthService(repo postgres.UserRepositoryInt, jwtManager *jwt.JWTManager) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

func (s *AuthService) Register(email, name, password string) (*domain.User, error) {
	existing, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Email:    email,
		Name:     name,
		Password: string(hashedPassword),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.Password = ""

	return user, nil
}

func (s *AuthService) Login(req *domain.LoginRequest) (*domain.TokenResponse, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	accessToken, refreshToken, err := s.jwtManager.GenerateTokens(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &domain.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) RefreshTokens(refreshToken string) (*domain.TokenResponse, error) {
	claims, err := s.jwtManager.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	accessToken, newRefreshToken, err := s.jwtManager.GenerateTokens(claims.UserID, claims.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &domain.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) VerifyToken(token string) (*jwt.Claims, error) {
	return s.jwtManager.VerifyToken(token)
}
