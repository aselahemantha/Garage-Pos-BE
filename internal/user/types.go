package user

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrInvalidRole       = errors.New("role must be owner, advisor, or mechanic")
	ErrCannotDeactivateSelf = errors.New("cannot deactivate your own account")
)

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=owner advisor mechanic"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=2,max=100"`
	Role     *string `json:"role" validate:"omitempty,oneof=owner advisor mechanic"`
	IsActive *bool   `json:"is_active"`
}

type ChangePasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=6"`
}
