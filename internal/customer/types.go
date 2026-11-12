package customer

import (
	"time"

	"garage-pos-be/internal/vehicle"
)

type CustomerResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     *string   `json:"email,omitempty"`
	Address   *string   `json:"address,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomerDetailResponse struct {
	CustomerResponse
	Vehicles []vehicle.VehicleResponse `json:"vehicles"`
}

type CreateCustomerRequest struct {
	Name    string  `json:"name" validate:"required,min=2"`
	Phone   string  `json:"phone" validate:"required,min=9"`
	Email   *string `json:"email,omitempty" validate:"omitempty,email"`
	Address *string `json:"address,omitempty"`
	Notes   *string `json:"notes,omitempty"`
}

type UpdateCustomerRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,min=9"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Address  *string `json:"address,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}
