package report

type DateRange struct {
	Period    string `json:"period"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type SummaryResponse struct {
	Period             string `json:"period"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	TotalCollected     int64  `json:"total_collected"`
	CashCollected      int64  `json:"cash_collected"`
	CardCollected      int64  `json:"card_collected"`
	BankCollected      int64  `json:"bank_collected"`
	PaidInvoicesCount  int64  `json:"paid_invoices_count"`
	TotalInvoiced      int64  `json:"total_invoiced"`
	InvoiceCount       int64  `json:"invoice_count"`
	TotalSubtotal      int64  `json:"total_subtotal"`
	TotalDiscount      int64  `json:"total_discount"`
	TotalTax           int64  `json:"total_tax"`
	TotalOutstanding   int64  `json:"total_outstanding"`
	OutstandingCount   int64  `json:"outstanding_count"`
	CompletedJobsCount int64  `json:"completed_jobs_count"`
	AverageJobValue    int64  `json:"average_job_value"`
	LaborTotal         int64  `json:"labor_total"`
	PartsTotal         int64  `json:"parts_total"`
	CustomTotal        int64  `json:"custom_total"`
}

type RevenueTrendItem struct {
	Day         string `json:"day"`
	TotalAmount int64  `json:"total_amount"`
	CashAmount  int64  `json:"cash_amount"`
	CardAmount  int64  `json:"card_amount"`
	BankAmount  int64  `json:"bank_amount"`
}

type PaymentMethodItem struct {
	PaymentMethod string  `json:"payment_method"`
	TotalAmount   int64   `json:"total_amount"`
	PaymentCount  int64   `json:"payment_count"`
	Percentage    float64 `json:"percentage"`
}

type TopServiceItem struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	TimesPerformed int64  `json:"times_performed"`
	TotalRevenue   int64  `json:"total_revenue"`
}

type TopPartItem struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Sku          string  `json:"sku"`
	QtySold      float64 `json:"qty_sold"`
	TotalRevenue int64   `json:"total_revenue"`
}

type TopOutstandingInvoiceItem struct {
	ID            int64   `json:"id"`
	InvoiceNo     string  `json:"invoice_no"`
	CustomerID    int64   `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	CustomerPhone string  `json:"customer_phone"`
	PlateNo       *string `json:"plate_no,omitempty"`
	Total         int64   `json:"total"`
	PaidTotal     int64   `json:"paid_total"`
	Balance       int64   `json:"balance"`
	IssuedAt      string  `json:"issued_at"`
}

type CashSessionSummaryItem struct {
	ID             int64   `json:"id"`
	SessionNo      string  `json:"session_no"`
	UserName       string  `json:"user_name"`
	OpenedAt       string  `json:"opened_at"`
	ClosedAt       *string `json:"closed_at,omitempty"`
	OpeningFloat   int64   `json:"opening_float"`
	ClosingCounted *int64  `json:"closing_counted,omitempty"`
	ExpectedCash   int64   `json:"expected_cash"`
	Difference     *int64  `json:"difference,omitempty"`
	Status         string  `json:"status"`
}

type EndOfDayReportResponse struct {
	Date                string                   `json:"date"`
	TotalCollected      int64                    `json:"total_collected"`
	CashCollected       int64                    `json:"cash_collected"`
	CardCollected       int64                    `json:"card_collected"`
	BankCollected       int64                    `json:"bank_collected"`
	InvoicesPaidCount   int64                    `json:"invoices_paid_count"`
	JobsCompletedCount  int64                    `json:"jobs_completed_count"`
	TotalOpeningFloat   int64                    `json:"total_opening_float"`
	TotalClosingCounted int64                    `json:"total_closing_counted"`
	TotalExpectedCash   int64                    `json:"total_expected_cash"`
	TotalDifference     int64                    `json:"total_difference"`
	Sessions            []CashSessionSummaryItem `json:"sessions"`
}
