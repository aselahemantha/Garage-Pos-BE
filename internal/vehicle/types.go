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

type VehicleHistoryJobItem struct {
	Description string  `json:"description"`
	ItemType    string  `json:"item_type"`
	Qty         float64 `json:"qty"`
	LineTotal   int64   `json:"line_total"`
}

type VehicleHistoryJob struct {
	ID          int64                   `json:"id"`
	JobNo       string                  `json:"job_no"`
	Status      string                  `json:"status"`
	MileageIn   int32                   `json:"mileage_in"`
	Complaint   *string                 `json:"complaint,omitempty"`
	Notes       *string                 `json:"notes,omitempty"`
	AdvisorName string                  `json:"advisor_name"`
	TotalAmount int64                   `json:"total_amount"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
	Items       []VehicleHistoryJobItem `json:"items"`
}

type VehicleHistoryInvoice struct {
	ID        int64     `json:"id"`
	InvoiceNo string    `json:"invoice_no"`
	Total     int64     `json:"total"`
	PaidTotal int64     `json:"paid_total"`
	Balance   int64     `json:"balance"`
	Status    string    `json:"status"`
	IssuedAt  time.Time `json:"issued_at"`
}

type VehicleCustomerInfo struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`
}

type VehicleHistoryResponse struct {
	Vehicle  VehicleResponse         `json:"vehicle"`
	Customer VehicleCustomerInfo     `json:"customer"`
	Jobs     []VehicleHistoryJob     `json:"jobs"`
	Invoices []VehicleHistoryInvoice `json:"invoices"`
}
