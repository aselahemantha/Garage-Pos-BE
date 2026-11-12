package job

import "time"

type JobItemResponse struct {
	ID           int64     `json:"id"`
	JobCardID    int64     `json:"job_card_id"`
	ItemType     string    `json:"item_type"` // service or part
	ServiceID    *int64    `json:"service_id,omitempty"`
	PartID       *int64    `json:"part_id,omitempty"`
	Description  string    `json:"description"`
	Qty          float64   `json:"qty"`
	UnitPrice    int64     `json:"unit_price"` // minor units
	Discount     int64     `json:"discount"`   // minor units
	LineTotal    int64     `json:"line_total"` // minor units
	MechanicID   *int64    `json:"mechanic_id,omitempty"`
	MechanicName *string   `json:"mechanic_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type JobCardResponse struct {
	ID            int64      `json:"id"`
	JobNo         string     `json:"job_no"`
	CustomerID    int64      `json:"customer_id"`
	CustomerName  string     `json:"customer_name"`
	CustomerPhone string     `json:"customer_phone"`
	VehicleID     int64      `json:"vehicle_id"`
	PlateNo       string     `json:"plate_no"`
	VehicleMake   string     `json:"vehicle_make"`
	VehicleModel  string     `json:"vehicle_model"`
	VehicleYear   *int32     `json:"vehicle_year,omitempty"`
	AdvisorID     int64      `json:"advisor_id"`
	AdvisorName   string     `json:"advisor_name"`
	MileageIn     int32      `json:"mileage_in"`
	Complaint     *string    `json:"complaint,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	Status        string     `json:"status"` // pending, in_progress, completed, delivered, cancelled
	PromisedBy    *time.Time `json:"promised_by,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type JobDetailResponse struct {
	JobCardResponse
	Items      []JobItemResponse `json:"items"`
	TotalMinor int64             `json:"total_minor"`
}

type CreateJobItemInput struct {
	ItemType    string  `json:"item_type" validate:"required,oneof=service part"`
	ServiceID   *int64  `json:"service_id,omitempty"`
	PartID      *int64  `json:"part_id,omitempty"`
	Description string  `json:"description" validate:"required"`
	Qty         float64 `json:"qty" validate:"gt=0"`
	UnitPrice   int64   `json:"unit_price" validate:"gte=0"`
	Discount    int64   `json:"discount" validate:"gte=0"`
	MechanicID  *int64  `json:"mechanic_id,omitempty"`
}

type CreateJobCardRequest struct {
	CustomerID int64                `json:"customer_id" validate:"required"`
	VehicleID  int64                `json:"vehicle_id" validate:"required"`
	MileageIn  int32                `json:"mileage_in" validate:"gte=0"`
	Complaint  *string              `json:"complaint,omitempty"`
	Notes      *string              `json:"notes,omitempty"`
	PromisedBy *string              `json:"promised_by,omitempty"` // RFC3339 string
	Items      []CreateJobItemInput `json:"items"`
}

type UpdateJobStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending in_progress completed delivered cancelled"`
}
