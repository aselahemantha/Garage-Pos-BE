package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrInvoiceAlreadyIssued   = errors.New("active invoice already exists for this job card")
	ErrInvoiceAlreadyPaid     = errors.New("invoice is already fully paid")
	ErrInvoiceVoided          = errors.New("cannot process payment for a voided invoice")
	ErrInsufficientTendered   = errors.New("tendered amount is less than payment amount")
	ErrInvalidPaymentAmount   = errors.New("payment amount must be greater than zero")
	ErrSessionAlreadyOpen     = errors.New("a cash session is already open")
	ErrNoActiveSession        = errors.New("no active cash session found")
	ErrUnauthorizedRole       = errors.New("unauthorized: owner permission required")
)

type Service struct {
	queries *store.Queries
	pool    *pgxpool.Pool
}

func NewService(queries *store.Queries, pool *pgxpool.Pool) *Service {
	return &Service{
		queries: queries,
		pool:    pool,
	}
}

func (s *Service) IssueInvoice(ctx context.Context, userID int64, req IssueInvoiceRequest) (*InvoiceDetailResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	var (
		customerID = int64(0)
		vehicleID  *int64
		jobCardID  *int64
	)

	// 1. Resolve Customer and Vehicle from JobCard if supplied
	if req.JobCardID != nil && *req.JobCardID > 0 {
		jobCardID = req.JobCardID
		jc, err := qtx.GetJobCardByID(ctx, *req.JobCardID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("job card not found")
			}
			return nil, fmt.Errorf("get job card: %w", err)
		}

		// Check if active invoice already exists for this job
		existingInv, err := qtx.GetInvoiceByJobCardID(ctx, pgtype.Int8{Int64: *jobCardID, Valid: true})
		if err == nil && existingInv.ID > 0 {
			return nil, ErrInvoiceAlreadyIssued
		}

		customerID = jc.CustomerID
		vehicleID = &jc.VehicleID
	}

	if req.CustomerID != nil && *req.CustomerID > 0 {
		customerID = *req.CustomerID
	}
	if req.VehicleID != nil && *req.VehicleID > 0 {
		vehicleID = req.VehicleID
	}

	if customerID == 0 {
		return nil, errors.New("customer_id is required to issue an invoice")
	}

	// 2. Prepare Items
	items := req.Items
	if len(items) == 0 && jobCardID != nil {
		jobItems, err := qtx.ListJobItems(ctx, *jobCardID)
		if err != nil {
			return nil, fmt.Errorf("list job items: %w", err)
		}

		for _, ji := range jobItems {
			qtyFloat := numericToFloat(ji.Qty)
			itemType := "labor"
			if ji.ItemType == "part" {
				itemType = "part"
			}

			var sID *int64
			if ji.ServiceID.Valid {
				v := ji.ServiceID.Int64
				sID = &v
			}
			var pID *int64
			if ji.PartID.Valid {
				v := ji.PartID.Int64
				pID = &v
			}

			var mechName *string
			if ji.MechanicName.Valid {
				v := ji.MechanicName.String
				mechName = &v
			}

			items = append(items, InvoiceItemInput{
				ItemType:     itemType,
				ServiceID:    sID,
				PartID:       pID,
				Description:  ji.Description,
				Qty:          qtyFloat,
				UnitPrice:    ji.UnitPrice,
				Discount:     ji.Discount,
				MechanicName: mechName,
			})
		}
	}

	if len(items) == 0 {
		return nil, errors.New("invoice must have at least one line item")
	}

	// 3. Calculate Totals
	var subtotal int64 = 0
	var discountTotal int64 = 0

	type calculatedLine struct {
		input     InvoiceItemInput
		lineTotal int64
	}
	calcLines := make([]calculatedLine, 0, len(items))

	for _, it := range items {
		qty := it.Qty
		if qty <= 0 {
			qty = 1
		}
		rawLine := int64(math.Round(float64(it.UnitPrice) * qty))
		lineTot := rawLine - it.Discount
		if lineTot < 0 {
			lineTot = 0
		}
		subtotal += lineTot
		discountTotal += it.Discount

		calcLines = append(calcLines, calculatedLine{
			input:     it,
			lineTotal: lineTot,
		})
	}

	// Determine Tax Bps
	taxBps := int32(0)
	if req.TaxBps != nil {
		taxBps = *req.TaxBps
	} else {
		// Read tax_bps from settings
		if taxSetting, err := qtx.GetSetting(ctx, "tax_bps"); err == nil {
			if parsed, err := strconv.Atoi(taxSetting.Value); err == nil {
				taxBps = int32(parsed)
			}
		}
	}

	taxTotal := int64(0)
	if taxBps > 0 {
		taxTotal = (subtotal*int64(taxBps) + 5000) / 10000
	}
	total := subtotal + taxTotal
	balance := total

	// 4. Generate Invoice No
	invNo, err := qtx.NextInvoiceNo(ctx)
	if err != nil {
		return nil, fmt.Errorf("next invoice no: %w", err)
	}

	// 5. Active cash session if available
	var cashSessionID pgtype.Int8
	if activeSession, err := qtx.GetActiveCashSession(ctx); err == nil && activeSession.ID > 0 {
		cashSessionID = pgtype.Int8{Int64: activeSession.ID, Valid: true}
	}

	var notesParam pgtype.Text
	if req.Notes != nil && strings.TrimSpace(*req.Notes) != "" {
		notesParam = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}

	var jobCardParam pgtype.Int8
	if jobCardID != nil {
		jobCardParam = pgtype.Int8{Int64: *jobCardID, Valid: true}
	}

	var vehicleParam pgtype.Int8
	if vehicleID != nil {
		vehicleParam = pgtype.Int8{Int64: *vehicleID, Valid: true}
	}

	// 6. Insert Invoice
	createdInv, err := qtx.CreateInvoice(ctx, store.CreateInvoiceParams{
		InvoiceNo:     invNo,
		JobCardID:     jobCardParam,
		CustomerID:    customerID,
		VehicleID:     vehicleParam,
		CashSessionID: cashSessionID,
		Status:        "issued",
		Subtotal:      subtotal,
		DiscountTotal: discountTotal,
		TaxBps:        taxBps,
		TaxTotal:      taxTotal,
		Total:         total,
		PaidTotal:     0,
		Balance:       balance,
		IssuedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Notes:         notesParam,
		CreatedBy:     userID,
	})
	if err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	// 7. Insert Line Items
	for _, cl := range calcLines {
		var sID pgtype.Int8
		if cl.input.ServiceID != nil {
			sID = pgtype.Int8{Int64: *cl.input.ServiceID, Valid: true}
		}
		var pID pgtype.Int8
		if cl.input.PartID != nil {
			pID = pgtype.Int8{Int64: *cl.input.PartID, Valid: true}
		}
		var mech pgtype.Text
		if cl.input.MechanicName != nil {
			mech = pgtype.Text{String: *cl.input.MechanicName, Valid: true}
		}

		qtyNumeric, _ := floatToNumeric(cl.input.Qty)

		_, err := qtx.CreateInvoiceItem(ctx, store.CreateInvoiceItemParams{
			InvoiceID:    createdInv.ID,
			ItemType:     cl.input.ItemType,
			ServiceID:    sID,
			PartID:       pID,
			Description:  cl.input.Description,
			Qty:          qtyNumeric,
			UnitPrice:    cl.input.UnitPrice,
			Discount:     cl.input.Discount,
			LineTotal:    cl.lineTotal,
			MechanicName: mech,
		})
		if err != nil {
			return nil, fmt.Errorf("create invoice item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetInvoice(ctx, createdInv.ID)
}

func (s *Service) RecordPayment(ctx context.Context, userID int64, invoiceID int64, req RecordPaymentRequest) (*InvoiceDetailResponse, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	inv, err := qtx.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	if inv.Status == "void" {
		return nil, ErrInvoiceVoided
	}
	if inv.Status == "paid" && inv.Balance <= 0 {
		return nil, ErrInvoiceAlreadyPaid
	}

	// Calculate change if cash
	var tenderedParam pgtype.Int8
	changeGiven := int64(0)
	if req.PaymentMethod == "cash" && req.Tendered != nil {
		if *req.Tendered < req.Amount {
			return nil, ErrInsufficientTendered
		}
		tenderedParam = pgtype.Int8{Int64: *req.Tendered, Valid: true}
		changeGiven = *req.Tendered - req.Amount
	}

	// Active cash session
	var sessionIDParam pgtype.Int8
	activeSession, err := qtx.GetActiveCashSession(ctx)
	if err == nil && activeSession.ID > 0 {
		sessionIDParam = pgtype.Int8{Int64: activeSession.ID, Valid: true}
		if req.PaymentMethod == "cash" {
			_, _ = qtx.IncrementCashSessionExpectedCash(ctx, store.IncrementCashSessionExpectedCashParams{
				ID:           activeSession.ID,
				ExpectedCash: req.Amount,
			})
		}
	}

	var refParam pgtype.Text
	if req.ReferenceNo != nil && strings.TrimSpace(*req.ReferenceNo) != "" {
		refParam = pgtype.Text{String: strings.TrimSpace(*req.ReferenceNo), Valid: true}
	}
	var notesParam pgtype.Text
	if req.Notes != nil && strings.TrimSpace(*req.Notes) != "" {
		notesParam = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}

	// 1. Create Payment
	_, err = qtx.CreatePayment(ctx, store.CreatePaymentParams{
		InvoiceID:     invoiceID,
		CashSessionID: sessionIDParam,
		PaymentMethod: req.PaymentMethod,
		Amount:        req.Amount,
		Tendered:      tenderedParam,
		ChangeGiven:   changeGiven,
		ReferenceNo:   refParam,
		Notes:         notesParam,
		CreatedBy:     userID,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	// 2. Update Invoice Progress
	newPaidTotal := inv.PaidTotal + req.Amount
	newBalance := inv.Total - newPaidTotal
	newStatus := "partially_paid"
	if newBalance <= 0 {
		newBalance = 0
		newStatus = "paid"
	}

	_, err = qtx.UpdateInvoicePaymentProgress(ctx, store.UpdateInvoicePaymentProgressParams{
		ID:        invoiceID,
		PaidTotal: newPaidTotal,
		Balance:   newBalance,
		Status:    newStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("update invoice progress: %w", err)
	}

	// 3. Mark delivered if requested & linked to job card
	if req.MarkDelivered != nil && *req.MarkDelivered && inv.JobCardID.Valid {
		_, _ = qtx.UpdateJobStatus(ctx, store.UpdateJobStatusParams{
			ID:     inv.JobCardID.Int64,
			Status: "delivered",
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetInvoice(ctx, invoiceID)
}

func (s *Service) VoidInvoice(ctx context.Context, userID int64, userRole string, invoiceID int64, req VoidInvoiceRequest) (*InvoiceDetailResponse, error) {
	if userRole != "owner" {
		return nil, ErrUnauthorizedRole
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, errors.New("void reason is required")
	}

	inv, err := s.queries.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	if inv.Status == "void" {
		return nil, errors.New("invoice is already void")
	}

	_, err = s.queries.VoidInvoice(ctx, store.VoidInvoiceParams{
		ID:         invoiceID,
		VoidReason: pgtype.Text{String: strings.TrimSpace(req.Reason), Valid: true},
		VoidedBy:   pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("void invoice: %w", err)
	}

	return s.GetInvoice(ctx, invoiceID)
}

func (s *Service) GetInvoice(ctx context.Context, id int64) (*InvoiceDetailResponse, error) {
	inv, err := s.queries.GetInvoiceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	items, err := s.queries.ListInvoiceItemsByInvoiceID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list invoice items: %w", err)
	}

	payments, err := s.queries.ListPaymentsByInvoiceID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list invoice payments: %w", err)
	}

	resp := &InvoiceDetailResponse{
		ID:            inv.ID,
		InvoiceNo:     inv.InvoiceNo,
		CustomerID:    inv.CustomerID,
		CustomerName:  inv.CustomerName,
		CustomerPhone: inv.CustomerPhone,
		Status:        inv.Status,
		Subtotal:      inv.Subtotal,
		DiscountTotal: inv.DiscountTotal,
		TaxBps:        inv.TaxBps,
		TaxTotal:      inv.TaxTotal,
		Total:         inv.Total,
		PaidTotal:     inv.PaidTotal,
		Balance:       inv.Balance,
		IssuedAt:      inv.IssuedAt.Time.Format(time.RFC3339),
		CreatedByName: inv.CreatedByName,
		CreatedAt:     inv.CreatedAt.Time.Format(time.RFC3339),
		Items:         make([]InvoiceItemResponse, 0, len(items)),
		Payments:      make([]PaymentResponse, 0, len(payments)),
	}

	if inv.CustomerAddress.Valid {
		resp.CustomerAddress = &inv.CustomerAddress.String
	}
	if inv.JobCardID.Valid {
		resp.JobCardID = &inv.JobCardID.Int64
	}
	if inv.JobNo.Valid {
		resp.JobNo = &inv.JobNo.String
	}
	if inv.VehicleID.Valid {
		resp.VehicleID = &inv.VehicleID.Int64
	}
	if inv.PlateNo.Valid {
		resp.PlateNo = &inv.PlateNo.String
	}
	if inv.VehicleMake.Valid {
		resp.VehicleMake = &inv.VehicleMake.String
	}
	if inv.VehicleModel.Valid {
		resp.VehicleModel = &inv.VehicleModel.String
	}
	if inv.VehicleColor.Valid {
		resp.VehicleColor = &inv.VehicleColor.String
	}
	if inv.DueAt.Valid {
		dueStr := inv.DueAt.Time.Format(time.RFC3339)
		resp.DueAt = &dueStr
	}
	if inv.PaidAt.Valid {
		paidStr := inv.PaidAt.Time.Format(time.RFC3339)
		resp.PaidAt = &paidStr
	}
	if inv.VoidedAt.Valid {
		voidStr := inv.VoidedAt.Time.Format(time.RFC3339)
		resp.VoidedAt = &voidStr
	}
	if inv.VoidReason.Valid {
		resp.VoidReason = &inv.VoidReason.String
	}
	if inv.Notes.Valid {
		resp.Notes = &inv.Notes.String
	}

	for _, it := range items {
		itemResp := InvoiceItemResponse{
			ID:          it.ID,
			InvoiceID:   it.InvoiceID,
			ItemType:    it.ItemType,
			Description: it.Description,
			Qty:         numericToFloat(it.Qty),
			UnitPrice:   it.UnitPrice,
			Discount:    it.Discount,
			LineTotal:   it.LineTotal,
			CreatedAt:   it.CreatedAt.Time.Format(time.RFC3339),
		}
		if it.ServiceID.Valid {
			itemResp.ServiceID = &it.ServiceID.Int64
		}
		if it.PartID.Valid {
			itemResp.PartID = &it.PartID.Int64
		}
		if it.MechanicName.Valid {
			itemResp.MechanicName = &it.MechanicName.String
		}
		resp.Items = append(resp.Items, itemResp)
	}

	for _, p := range payments {
		pResp := PaymentResponse{
			ID:            p.ID,
			InvoiceID:     p.InvoiceID,
			PaymentMethod: p.PaymentMethod,
			Amount:        p.Amount,
			ChangeGiven:   p.ChangeGiven,
			CreatedByName: p.CreatedByName,
			CreatedAt:     p.CreatedAt.Time.Format(time.RFC3339),
		}
		if p.CashSessionID.Valid {
			pResp.CashSessionID = &p.CashSessionID.Int64
		}
		if p.Tendered.Valid {
			pResp.Tendered = &p.Tendered.Int64
		}
		if p.ReferenceNo.Valid {
			pResp.ReferenceNo = &p.ReferenceNo.String
		}
		if p.Notes.Valid {
			pResp.Notes = &p.Notes.String
		}
		resp.Payments = append(resp.Payments, pResp)
	}

	return resp, nil
}

func (s *Service) ListInvoices(ctx context.Context, status *string, search *string, limit int32, offset int32) (*ListInvoicesResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	statusParam := ""
	if status != nil {
		statusParam = *status
	}
	searchParam := ""
	if search != nil {
		searchParam = strings.TrimSpace(*search)
	}

	rows, err := s.queries.ListInvoices(ctx, store.ListInvoicesParams{
		Column1: statusParam,
		Column2: searchParam,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	total, err := s.queries.CountInvoices(ctx, store.CountInvoicesParams{
		Column1: statusParam,
		Column2: searchParam,
	})
	if err != nil {
		return nil, fmt.Errorf("count invoices: %w", err)
	}

	summaries := make([]InvoiceSummaryResponse, 0, len(rows))
	for _, r := range rows {
		sum := InvoiceSummaryResponse{
			ID:            r.ID,
			InvoiceNo:     r.InvoiceNo,
			CustomerID:    r.CustomerID,
			CustomerName:  r.CustomerName,
			CustomerPhone: r.CustomerPhone,
			Status:        r.Status,
			Subtotal:      r.Subtotal,
			DiscountTotal: r.DiscountTotal,
			TaxBps:        r.TaxBps,
			TaxTotal:      r.TaxTotal,
			Total:         r.Total,
			PaidTotal:     r.PaidTotal,
			Balance:       r.Balance,
			IssuedAt:      r.IssuedAt.Time.Format(time.RFC3339),
			CreatedAt:     r.CreatedAt.Time.Format(time.RFC3339),
		}
		if r.JobCardID.Valid {
			sum.JobCardID = &r.JobCardID.Int64
		}
		if r.JobNo.Valid {
			sum.JobNo = &r.JobNo.String
		}
		if r.VehicleID.Valid {
			sum.VehicleID = &r.VehicleID.Int64
		}
		if r.PlateNo.Valid {
			sum.PlateNo = &r.PlateNo.String
		}
		if r.PaidAt.Valid {
			pStr := r.PaidAt.Time.Format(time.RFC3339)
			sum.PaidAt = &pStr
		}
		if r.VoidedAt.Valid {
			vStr := r.VoidedAt.Time.Format(time.RFC3339)
			sum.VoidedAt = &vStr
		}
		summaries = append(summaries, sum)
	}

	return &ListInvoicesResponse{
		Invoices: summaries,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

// Cash Session Methods

func (s *Service) GetActiveCashSession(ctx context.Context) (*CashSessionResponse, error) {
	row, err := s.queries.GetActiveCashSession(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // None open
		}
		return nil, err
	}

	detail, err := s.queries.GetCashSessionByID(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return toCashSessionResponse(detail), nil
}

func (s *Service) OpenCashSession(ctx context.Context, userID int64, req OpenCashSessionRequest) (*CashSessionResponse, error) {
	// Check existing
	if existing, err := s.queries.GetActiveCashSession(ctx); err == nil && existing.ID > 0 {
		return nil, ErrSessionAlreadyOpen
	}

	sessionNo, err := s.queries.NextCashSessionNo(ctx)
	if err != nil {
		return nil, fmt.Errorf("next cash session no: %w", err)
	}

	var notesParam pgtype.Text
	if req.Notes != nil && strings.TrimSpace(*req.Notes) != "" {
		notesParam = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}

	session, err := s.queries.CreateCashSession(ctx, store.CreateCashSessionParams{
		SessionNo:    sessionNo,
		UserID:       userID,
		OpeningFloat: req.OpeningFloat,
		Notes:        notesParam,
	})
	if err != nil {
		return nil, fmt.Errorf("create cash session: %w", err)
	}

	detail, err := s.queries.GetCashSessionByID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return toCashSessionResponse(detail), nil
}

func (s *Service) CloseCashSession(ctx context.Context, sessionID int64, req CloseCashSessionRequest) (*CashSessionResponse, error) {
	var notesParam pgtype.Text
	if req.Notes != nil && strings.TrimSpace(*req.Notes) != "" {
		notesParam = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}

	session, err := s.queries.CloseCashSession(ctx, store.CloseCashSessionParams{
		ID:             sessionID,
		ClosingCounted: pgtype.Int8{Int64: req.ClosingCounted, Valid: true},
		Notes:          notesParam,
	})
	if err != nil {
		return nil, fmt.Errorf("close cash session: %w", err)
	}

	detail, err := s.queries.GetCashSessionByID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	return toCashSessionResponse(detail), nil
}

func (s *Service) ListCashSessions(ctx context.Context, limit int32, offset int32) ([]CashSessionResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.queries.ListCashSessions(ctx, store.ListCashSessionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list cash sessions: %w", err)
	}

	result := make([]CashSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp := CashSessionResponse{
			ID:           r.ID,
			SessionNo:    r.SessionNo,
			UserID:       r.UserID,
			UserName:     r.UserName,
			OpenedAt:     r.OpenedAt.Time.Format(time.RFC3339),
			OpeningFloat: r.OpeningFloat,
			ExpectedCash: r.ExpectedCash,
			Status:       r.Status,
		}
		if r.ClosedAt.Valid {
			cStr := r.ClosedAt.Time.Format(time.RFC3339)
			resp.ClosedAt = &cStr
		}
		if r.ClosingCounted.Valid {
			resp.ClosingCounted = &r.ClosingCounted.Int64
		}
		if r.Difference.Valid {
			resp.Difference = &r.Difference.Int64
		}
		if r.Notes.Valid {
			resp.Notes = &r.Notes.String
		}
		result = append(result, resp)
	}

	return result, nil
}

func toCashSessionResponse(r store.GetCashSessionByIDRow) *CashSessionResponse {
	resp := &CashSessionResponse{
		ID:           r.ID,
		SessionNo:    r.SessionNo,
		UserID:       r.UserID,
		UserName:     r.UserName,
		OpenedAt:     r.OpenedAt.Time.Format(time.RFC3339),
		OpeningFloat: r.OpeningFloat,
		ExpectedCash: r.ExpectedCash,
		Status:       r.Status,
	}
	if r.ClosedAt.Valid {
		cStr := r.ClosedAt.Time.Format(time.RFC3339)
		resp.ClosedAt = &cStr
	}
	if r.ClosingCounted.Valid {
		resp.ClosingCounted = &r.ClosingCounted.Int64
	}
	if r.Difference.Valid {
		resp.Difference = &r.Difference.Int64
	}
	if r.Notes.Valid {
		resp.Notes = &r.Notes.String
	}
	return resp
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

func floatToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	err := n.Scan(fmt.Sprintf("%.3f", f))
	return n, err
}
