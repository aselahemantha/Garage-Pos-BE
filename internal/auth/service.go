package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"garage-pos-be/internal/config"
	"garage-pos-be/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Service struct {
	queries *store.Queries
	pool    *pgxpool.Pool
	cfg     *config.Config
}

func NewService(queries *store.Queries, pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{
		queries: queries,
		pool:    pool,
		cfg:     cfg,
	}
}

func (s *Service) SeedOwner(ctx context.Context) error {
	count, err := s.queries.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.SeedOwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	_, err = s.queries.CreateUser(ctx, store.CreateUserParams{
		Name:         "System Owner",
		Username:     s.cfg.SeedOwnerUsername,
		PasswordHash: string(hash),
		Role:         "owner",
		IsActive:     true,
	})
	if err != nil {
		return fmt.Errorf("seed initial owner: %w", err)
	}

	return nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*LoginResponse, string, error) {
	u, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if !u.IsActive {
		return nil, "", ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	accessToken, err := s.generateAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.createRefreshToken(ctx, u.ID)
	if err != nil {
		return nil, "", fmt.Errorf("create refresh token: %w", err)
	}

	resp := &LoginResponse{
		AccessToken: accessToken,
		User: UserResponse{
			ID:        u.ID,
			Name:      u.Name,
			Username:  u.Username,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Time,
			UpdatedAt: u.UpdatedAt.Time,
		},
	}

	return resp, refreshToken, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (string, string, error) {
	tokenHash := hashToken(rawRefreshToken)

	row, err := s.queries.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	// Revoke current token (rotation)
	if err := s.queries.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return "", "", fmt.Errorf("revoke old refresh token: %w", err)
	}

	u, err := s.queries.GetUserByID(ctx, row.UserID)
	if err != nil || !u.IsActive {
		return "", "", ErrInvalidToken
	}

	newAccessToken, err := s.generateAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, err := s.createRefreshToken(ctx, u.ID)
	if err != nil {
		return "", "", fmt.Errorf("create refresh token: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	tokenHash := hashToken(rawRefreshToken)
	return s.queries.RevokeRefreshToken(ctx, tokenHash)
}

func (s *Service) GetUserByID(ctx context.Context, userID int64) (*UserResponse, error) {
	u, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Username:  u.Username,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}, nil
}

func (s *Service) generateAccessToken(userID int64, username, role string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *Service) createRefreshToken(ctx context.Context, userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	rawToken := hex.EncodeToString(b)
	tokenHash := hashToken(rawToken)

	expiresAt := time.Now().Add(s.cfg.RefreshTokenTTL)

	_, err := s.queries.CreateRefreshToken(ctx, store.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", err
	}

	return rawToken, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
