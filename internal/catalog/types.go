package catalog

import "time"

type CategoryResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	SortOrder int32     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ServiceResponse struct {
	ID           int64     `json:"id"`
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	Price        int64     `json:"price"` // minor units (e.g. cents / cents of LKR)
	EstMinutes   *int32    `json:"est_minutes,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateCategoryRequest struct {
	Name      string `json:"name" validate:"required,min=2"`
	SortOrder int32  `json:"sort_order"`
}

type CreateServiceRequest struct {
	CategoryID  int64   `json:"category_id" validate:"required"`
	Name        string  `json:"name" validate:"required,min=2"`
	Description *string `json:"description,omitempty"`
	Price       int64   `json:"price" validate:"gte=0"`
	EstMinutes  *int32  `json:"est_minutes,omitempty"`
}

type UpdateServiceRequest struct {
	CategoryID  *int64  `json:"category_id,omitempty"`
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2"`
	Description *string `json:"description,omitempty"`
	Price       *int64  `json:"price,omitempty" validate:"omitempty,gte=0"`
	EstMinutes  *int32  `json:"est_minutes,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}
