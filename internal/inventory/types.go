package inventory

import "time"

type PartResponse struct {
	ID           int64     `json:"id"`
	SKU          string    `json:"sku"`
	Name         string    `json:"name"`
	Category     *string   `json:"category,omitempty"`
	Unit         string    `json:"unit"`
	CostPrice    int64     `json:"cost_price"` // minor units
	SellPrice    int64     `json:"sell_price"` // minor units
	ReorderLevel float64   `json:"reorder_level"`
	StockOnHand  float64   `json:"stock_on_hand"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreatePartRequest struct {
	SKU          string  `json:"sku" validate:"required,min=2"`
	Name         string  `json:"name" validate:"required,min=2"`
	Category     *string `json:"category,omitempty"`
	Unit         string  `json:"unit" validate:"required"` // pcs, litre, set, can
	CostPrice    int64   `json:"cost_price" validate:"gte=0"`
	SellPrice    int64   `json:"sell_price" validate:"gte=0"`
	ReorderLevel float64 `json:"reorder_level" validate:"gte=0"`
}

type UpdatePartRequest struct {
	Name         *string  `json:"name,omitempty" validate:"omitempty,min=2"`
	Category     *string  `json:"category,omitempty"`
	Unit         *string  `json:"unit,omitempty"`
	CostPrice    *int64   `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	SellPrice    *int64   `json:"sell_price,omitempty" validate:"omitempty,gte=0"`
	ReorderLevel *float64 `json:"reorder_level,omitempty" validate:"omitempty,gte=0"`
	IsActive     *bool    `json:"is_active,omitempty"`
}

type StockInRequest struct {
	Qty      float64 `json:"qty" validate:"gt=0"`
	UnitCost int64   `json:"unit_cost" validate:"gte=0"`
	Note     *string `json:"note,omitempty"`
}

type StockMovementResponse struct {
	ID        int64     `json:"id"`
	PartID    int64     `json:"part_id"`
	Qty       float64   `json:"qty"`
	Type      string    `json:"type"`
	RefType   *string   `json:"ref_type,omitempty"`
	RefID     *int64    `json:"ref_id,omitempty"`
	UnitCost  *int64    `json:"unit_cost,omitempty"`
	Note      *string   `json:"note,omitempty"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
