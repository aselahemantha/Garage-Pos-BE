package vehicle

import "time"

type VehicleResponse struct {
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customer_id"`
	PlateNo     string    `json:"plate_no"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Year        *int32    `json:"year,omitempty"`
	VIN         *string   `json:"vin,omitempty"`
	FuelType    *string   `json:"fuel_type,omitempty"`
	Color       *string   `json:"color,omitempty"`
	LastMileage *int32    `json:"last_mileage,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateVehicleRequest struct {
	CustomerID  int64   `json:"customer_id" validate:"required"`
	PlateNo     string  `json:"plate_no" validate:"required,min=3"`
	Make        string  `json:"make" validate:"required"`
	Model       string  `json:"model" validate:"required"`
	Year        *int32  `json:"year,omitempty"`
	VIN         *string `json:"vin,omitempty"`
	FuelType    *string `json:"fuel_type,omitempty"`
	Color       *string `json:"color,omitempty"`
	LastMileage *int32  `json:"last_mileage,omitempty"`
}

type UpdateVehicleRequest struct {
	Make        *string `json:"make,omitempty"`
	Model       *string `json:"model,omitempty"`
	Year        *int32  `json:"year,omitempty"`
	VIN         *string `json:"vin,omitempty"`
	FuelType    *string `json:"fuel_type,omitempty"`
	Color       *string `json:"color,omitempty"`
	LastMileage *int32  `json:"last_mileage,omitempty"`
}
