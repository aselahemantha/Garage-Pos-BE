package inventory

import "time"

type PartResponse struct {
	ID           int64     `json:"id"`
	SKU          string    `json:"sku"`
	Barcode      *string   `json:"barcode,omitempty"`
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
	Barcode      *string `json:"barcode,omitempty"`
	Name         string  `json:"name" validate:"required,min=2"`
	Category     *string `json:"category,omitempty"`
	Unit         string  `json:"unit" validate:"required"` // pcs, litre, set, can
	CostPrice    int64   `json:"cost_price" validate:"gte=0"`
	SellPrice    int64   `json:"sell_price" validate:"gte=0"`
	ReorderLevel float64 `json:"reorder_level" validate:"gte=0"`
}

type UpdatePartRequest struct {
	Barcode      *string  `json:"barcode,omitempty"`
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

type AdjustStockRequest struct {
	NewStockOnHand float64 `json:"new_stock_on_hand" validate:"gte=0"`
	Reason         string  `json:"reason" validate:"required,min=3"`
}

type BatchReorderItem struct {
	PartID   int64   `json:"part_id" validate:"required"`
	Qty      float64 `json:"qty" validate:"gt=0"`
	UnitCost int64   `json:"unit_cost" validate:"gte=0"`
	Note     *string `json:"note,omitempty"`
}

type BatchReorderRequest struct {
	Items        []BatchReorderItem `json:"items" validate:"required,min=1,dive"`
	SupplierName *string            `json:"supplier_name,omitempty"`
	InvoiceRef   *string            `json:"invoice_ref,omitempty"`
}

type LowStockItem struct {
	Part                PartResponse `json:"part"`
	Deficit             float64      `json:"deficit"`
	SuggestedReorderQty float64      `json:"suggested_reorder_qty"`
	EstimatedCost       int64        `json:"estimated_cost"`
}

type LowStockSummaryResponse struct {
	LowStockCount      int            `json:"low_stock_count"`
	OutOfStockCount    int            `json:"out_of_stock_count"`
	TotalEstimatedCost int64          `json:"total_estimated_cost"`
	Items              []LowStockItem `json:"items"`
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
