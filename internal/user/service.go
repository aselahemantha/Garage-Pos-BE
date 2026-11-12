package user

import (
	"context"
	"errors"
	"strings"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	queries *store.Queries
	pool    *pgxpool.Pool
}

func NewService(queries *store.Queries, pool *pgxpool.Pool) *Service {
	return &Service{
		queries: queries,
		pool:    pool,
	}
}

func (s *Service) List(ctx context.Context) ([]UserResponse, error) {
	rows, err := s.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]UserResponse, len(rows))
	for i, u := range rows {
		result[i] = UserResponse{
			ID:        u.ID,
			Name:      u.Name,
			Username:  u.Username,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Time,
			UpdatedAt: u.UpdatedAt.Time,
		}
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*UserResponse, error) {
	u, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
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

func (s *Service) Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	username := strings.ToLower(strings.TrimSpace(req.Username))

	// Check if username already exists
	_, err := s.queries.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, ErrUsernameTaken
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u, err := s.queries.CreateUser(ctx, store.CreateUserParams{
		Name:         strings.TrimSpace(req.Name),
		Username:     username,
		PasswordHash: string(hash),
		Role:         req.Role,
		IsActive:     true,
	})
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

func (s *Service) Update(ctx context.Context, id int64, req UpdateUserRequest) (*UserResponse, error) {
	// Verify user exists first
	_, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var nameText pgtype.Text
	if req.Name != nil {
		nameText = pgtype.Text{String: strings.TrimSpace(*req.Name), Valid: true}
	}

	var roleText pgtype.Text
	if req.Role != nil {
		roleText = pgtype.Text{String: *req.Role, Valid: true}
	}

	var isActiveBool pgtype.Bool
	if req.IsActive != nil {
		isActiveBool = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	u, err := s.queries.UpdateUser(ctx, store.UpdateUserParams{
		ID:       id,
		Name:     nameText,
		Role:     roleText,
		IsActive: isActiveBool,
	})
	if err != nil {
		return nil, err
	}

	// If account was deactivated, revoke all its refresh tokens immediately
	if req.IsActive != nil && !*req.IsActive {
		_ = s.queries.RevokeAllUserRefreshTokens(ctx, id)
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

func (s *Service) ChangePassword(ctx context.Context, id int64, newPassword string) error {
	// Verify user exists
	_, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.queries.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: string(hash),
	}); err != nil {
		return err
	}

	// Invalidate all active sessions for this user so they must re-authenticate with the new password
	return s.queries.RevokeAllUserRefreshTokens(ctx, id)
}
