package billing

type InvoiceItemInput struct {
	ItemType     string  `json:"item_type"` // labor, part, custom
	ServiceID    *int64  `json:"service_id,omitempty"`
	PartID       *int64  `json:"part_id,omitempty"`
	Description  string  `json:"description"`
	Qty          float64 `json:"qty"`
	UnitPrice    int64   `json:"unit_price"`
	Discount     int64   `json:"discount"`
	MechanicName *string `json:"mechanic_name,omitempty"`
}

type IssueInvoiceRequest struct {
	JobCardID *int64             `json:"job_card_id,omitempty"`
	CustomerID *int64            `json:"customer_id,omitempty"`
	VehicleID  *int64            `json:"vehicle_id,omitempty"`
	TaxBps     *int32            `json:"tax_bps,omitempty"`
	Notes      *string           `json:"notes,omitempty"`
	Items      []InvoiceItemInput `json:"items,omitempty"`
}

type RecordPaymentRequest struct {
	PaymentMethod string  `json:"payment_method"` // cash, card, bank_transfer, cheque, other
	Amount        int64   `json:"amount"`         // amount to apply towards balance (minor units)
	Tendered      *int64  `json:"tendered,omitempty"` // cash handed over (for change calc)
	ReferenceNo   *string `json:"reference_no,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	MarkDelivered *bool   `json:"mark_delivered,omitempty"`
}

type VoidInvoiceRequest struct {
	Reason string `json:"reason"`
}

type ApplyDiscountRequest struct {
	DiscountAmount  *int64   `json:"discount_amount,omitempty"`
	DiscountPercent *float64 `json:"discount_percent,omitempty"`
	Reason          *string  `json:"reason,omitempty"`
}

type OpenCashSessionRequest struct {
	OpeningFloat int64   `json:"opening_float"`
	Notes        *string `json:"notes,omitempty"`
}

type CloseCashSessionRequest struct {
	ClosingCounted int64   `json:"closing_counted"`
	Notes          *string `json:"notes,omitempty"`
}

type InvoiceItemResponse struct {
	ID           int64   `json:"id"`
	InvoiceID    int64   `json:"invoice_id"`
	ItemType     string  `json:"item_type"`
	ServiceID    *int64  `json:"service_id,omitempty"`
	PartID       *int64  `json:"part_id,omitempty"`
	Description  string  `json:"description"`
	Qty          float64 `json:"qty"`
	UnitPrice    int64   `json:"unit_price"`
	Discount     int64   `json:"discount"`
	LineTotal    int64   `json:"line_total"`
	MechanicName *string `json:"mechanic_name,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type PaymentResponse struct {
	ID            int64   `json:"id"`
	InvoiceID     int64   `json:"invoice_id"`
	CashSessionID *int64  `json:"cash_session_id,omitempty"`
	PaymentMethod string  `json:"payment_method"`
	Amount        int64   `json:"amount"`
	Tendered      *int64  `json:"tendered,omitempty"`
	ChangeGiven   int64   `json:"change_given"`
	ReferenceNo   *string `json:"reference_no,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	CreatedByName string  `json:"created_by_name"`
	CreatedAt     string  `json:"created_at"`
}

type InvoiceSummaryResponse struct {
	ID            int64      `json:"id"`
	InvoiceNo     string     `json:"invoice_no"`
	JobCardID     *int64     `json:"job_card_id,omitempty"`
	JobNo         *string    `json:"job_no,omitempty"`
	CustomerID    int64      `json:"customer_id"`
	CustomerName  string     `json:"customer_name"`
	CustomerPhone string     `json:"customer_phone"`
	VehicleID     *int64     `json:"vehicle_id,omitempty"`
	PlateNo       *string    `json:"plate_no,omitempty"`
	Status        string     `json:"status"`
	Subtotal      int64      `json:"subtotal"`
	DiscountTotal int64      `json:"discount_total"`
	TaxBps        int32      `json:"tax_bps"`
	TaxTotal      int64      `json:"tax_total"`
	Total         int64      `json:"total"`
	PaidTotal     int64      `json:"paid_total"`
	Balance       int64      `json:"balance"`
	IssuedAt      string     `json:"issued_at"`
	PaidAt        *string    `json:"paid_at,omitempty"`
	VoidedAt      *string    `json:"voided_at,omitempty"`
	CreatedAt     string     `json:"created_at"`
}

type InvoiceDetailResponse struct {
	ID              int64                 `json:"id"`
	InvoiceNo       string                `json:"invoice_no"`
	JobCardID       *int64                `json:"job_card_id,omitempty"`
	JobNo           *string               `json:"job_no,omitempty"`
	CustomerID      int64                 `json:"customer_id"`
	CustomerName    string                `json:"customer_name"`
	CustomerPhone   string                `json:"customer_phone"`
	CustomerAddress *string               `json:"customer_address,omitempty"`
	VehicleID       *int64                `json:"vehicle_id,omitempty"`
	PlateNo         *string               `json:"plate_no,omitempty"`
	VehicleMake     *string               `json:"vehicle_make,omitempty"`
	VehicleModel    *string               `json:"vehicle_model,omitempty"`
	VehicleColor    *string               `json:"vehicle_color,omitempty"`
	Status          string                `json:"status"`
	Subtotal        int64                 `json:"subtotal"`
	BillDiscount    int64                 `json:"bill_discount"`
	DiscountTotal   int64                 `json:"discount_total"`
	TaxBps          int32                 `json:"tax_bps"`
	TaxTotal        int64                 `json:"tax_total"`
	Total           int64                 `json:"total"`
	PaidTotal       int64                 `json:"paid_total"`
	Balance         int64                 `json:"balance"`
	IssuedAt        string                `json:"issued_at"`
	DueAt           *string               `json:"due_at,omitempty"`
	PaidAt          *string               `json:"paid_at,omitempty"`
	VoidedAt        *string               `json:"voided_at,omitempty"`
	VoidReason      *string               `json:"void_reason,omitempty"`
	Notes           *string               `json:"notes,omitempty"`
	CreatedByName   string                `json:"created_by_name"`
	CreatedAt       string                `json:"created_at"`
	Items           []InvoiceItemResponse `json:"items"`
	Payments        []PaymentResponse     `json:"payments"`
}

type CashSessionResponse struct {
	ID             int64   `json:"id"`
	SessionNo      string  `json:"session_no"`
	UserID         int64   `json:"user_id"`
	UserName       string  `json:"user_name"`
	OpenedAt       string  `json:"opened_at"`
	ClosedAt       *string `json:"closed_at,omitempty"`
	OpeningFloat   int64   `json:"opening_float"`
	ClosingCounted *int64  `json:"closing_counted,omitempty"`
	ExpectedCash   int64   `json:"expected_cash"`
	Difference     *int64  `json:"difference,omitempty"`
	Status         string  `json:"status"` // open, closed
	Notes          *string `json:"notes,omitempty"`
}

type ListInvoicesResponse struct {
	Invoices []InvoiceSummaryResponse `json:"invoices"`
	Total    int64                    `json:"total"`
	Limit    int32                    `json:"limit"`
	Offset   int32                    `json:"offset"`
}
