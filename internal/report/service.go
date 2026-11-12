package report

import (
	"context"
	"fmt"
	"time"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *Service) resolveDateRange(period, startDate, endDate string) (time.Time, time.Time, string) {
	now := time.Now()
	// Use local garage time (e.g. Asia/Colombo +05:30)
	loc := now.Location()

	var start, end time.Time

	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)
	case "yesterday":
		y := now.AddDate(0, 0, -1)
		start = time.Date(y.Year(), y.Month(), y.Day(), 0, 0, 0, 0, loc)
		end = time.Date(y.Year(), y.Month(), y.Day(), 23, 59, 59, 999999999, loc)
	case "this_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday as 7th day
		}
		mon := now.AddDate(0, 0, -(weekday - 1))
		start = time.Date(mon.Year(), mon.Month(), mon.Day(), 0, 0, 0, 0, loc)
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)
	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)
	case "custom":
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		} else {
			start = now.AddDate(0, 0, -30)
		}
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			end = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, loc)
		} else {
			end = now
		}
	default: // "last_30_days"
		period = "last_30_days"
		start = now.AddDate(0, 0, -30)
		end = now
	}

	return start, end, period
}

func (s *Service) GetSummary(ctx context.Context, period, startDate, endDate string) (*SummaryResponse, error) {
	start, end, resolvedPeriod := s.resolveDateRange(period, startDate, endDate)

	startTz := pgtype.Timestamptz{Time: start, Valid: true}
	endTz := pgtype.Timestamptz{Time: end, Valid: true}

	// 1. Revenue Collected
	revRow, err := s.queries.GetRevenueSummary(ctx, store.GetRevenueSummaryParams{
		CreatedAt:   startTz,
		CreatedAt_2: endTz,
	})
	if err != nil {
		return nil, fmt.Errorf("get revenue summary: %w", err)
	}

	// 2. Invoiced Summary
	invRow, err := s.queries.GetInvoicedSummary(ctx, store.GetInvoicedSummaryParams{
		IssuedAt:   startTz,
		IssuedAt_2: endTz,
	})
	if err != nil {
		return nil, fmt.Errorf("get invoiced summary: %w", err)
	}

	// 3. Outstanding Receivable Summary (overall snapshot)
	outRow, err := s.queries.GetOutstandingSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("get outstanding summary: %w", err)
	}

	// 4. Completed Jobs
	jobsCount, err := s.queries.GetCompletedJobsSummary(ctx, store.GetCompletedJobsSummaryParams{
		CompletedAt:   startTz,
		CompletedAt_2: endTz,
	})
	if err != nil {
		return nil, fmt.Errorf("get completed jobs summary: %w", err)
	}

	// 5. Category Split (Labor vs Parts)
	splitRow, err := s.queries.GetRevenueCategorySplit(ctx, store.GetRevenueCategorySplitParams{
		IssuedAt:   startTz,
		IssuedAt_2: endTz,
	})
	if err != nil {
		return nil, fmt.Errorf("get revenue category split: %w", err)
	}

	avgJobValue := int64(0)
	if jobsCount > 0 {
		avgJobValue = invRow.TotalInvoiced / jobsCount
	}

	return &SummaryResponse{
		Period:             resolvedPeriod,
		StartDate:          start.Format("2006-01-02"),
		EndDate:            end.Format("2006-01-02"),
		TotalCollected:     revRow.TotalCollected,
		CashCollected:      revRow.CashCollected,
		CardCollected:      revRow.CardCollected,
		BankCollected:      revRow.BankCollected,
		PaidInvoicesCount:  revRow.PaidInvoicesCount,
		TotalInvoiced:      invRow.TotalInvoiced,
		InvoiceCount:       invRow.InvoiceCount,
		TotalSubtotal:      invRow.TotalSubtotal,
		TotalDiscount:      invRow.TotalDiscount,
		TotalTax:           invRow.TotalTax,
		TotalOutstanding:   outRow.TotalOutstanding,
		OutstandingCount:   outRow.OutstandingCount,
		CompletedJobsCount: jobsCount,
		AverageJobValue:    avgJobValue,
		LaborTotal:         splitRow.LaborTotal,
		PartsTotal:         splitRow.PartsTotal,
		CustomTotal:        splitRow.CustomTotal,
	}, nil
}

func (s *Service) GetRevenueTrend(ctx context.Context, period, startDate, endDate string) ([]RevenueTrendItem, error) {
	start, end, _ := s.resolveDateRange(period, startDate, endDate)

	startTz := pgtype.Timestamptz{Time: start, Valid: true}
	endTz := pgtype.Timestamptz{Time: end, Valid: true}

	rows, err := s.queries.GetDailyRevenueTrend(ctx, store.GetDailyRevenueTrendParams{
		CreatedAt:   startTz,
		CreatedAt_2: endTz,
	})
	if err != nil {
		return nil, fmt.Errorf("get daily revenue trend: %w", err)
	}

	trend := make([]RevenueTrendItem, 0, len(rows))
	for _, r := range rows {
		trend = append(trend, RevenueTrendItem{
			Day:         r.Day,
			TotalAmount: r.TotalAmount,
			CashAmount:  r.CashAmount,
			CardAmount:  r.CardAmount,
			BankAmount:  r.BankAmount,
		})
	}

	return trend, nil
}

func (s *Service) GetPaymentMethods(ctx context.Context, period, startDate, endDate string) ([]PaymentMethodItem, error) {
	start, end, _ := s.resolveDateRange(period, startDate, endDate)

	startTz := pgtype.Timestamptz{Time: start, Valid: true}
	endTz := pgtype.Timestamptz{Time: end, Valid: true}

	rows, err := s.queries.GetPaymentMethodsBreakdown(ctx, store.GetPaymentMethodsBreakdownParams{
		CreatedAt:   startTz,
		CreatedAt_2: endTz,
	})
	if err != nil {
		return nil, fmt.Errorf("get payment methods: %w", err)
	}

	totalAll := int64(0)
	for _, r := range rows {
		totalAll += r.TotalAmount
	}

	methods := make([]PaymentMethodItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if totalAll > 0 {
			pct = (float64(r.TotalAmount) / float64(totalAll)) * 100
		}
		methods = append(methods, PaymentMethodItem{
			PaymentMethod: r.PaymentMethod,
			TotalAmount:   r.TotalAmount,
			PaymentCount:  r.PaymentCount,
			Percentage:    pct,
		})
	}

	return methods, nil
}

func (s *Service) GetTopItems(ctx context.Context, period, startDate, endDate string, limit int32) (map[string]interface{}, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	start, end, _ := s.resolveDateRange(period, startDate, endDate)

	startTz := pgtype.Timestamptz{Time: start, Valid: true}
	endTz := pgtype.Timestamptz{Time: end, Valid: true}

	servicesRows, err := s.queries.GetTopServices(ctx, store.GetTopServicesParams{
		IssuedAt:   startTz,
		IssuedAt_2: endTz,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get top services: %w", err)
	}

	topServices := make([]TopServiceItem, 0, len(servicesRows))
	for _, r := range servicesRows {
		topServices = append(topServices, TopServiceItem{
			ID:             r.ID,
			Name:           r.Name,
			TimesPerformed: r.TimesPerformed,
			TotalRevenue:   r.TotalRevenue,
		})
	}

	partsRows, err := s.queries.GetTopParts(ctx, store.GetTopPartsParams{
		IssuedAt:   startTz,
		IssuedAt_2: endTz,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get top parts: %w", err)
	}

	topParts := make([]TopPartItem, 0, len(partsRows))
	for _, r := range partsRows {
		qtyFloat := 0.0
		if r.QtySold.Valid {
			if f, err := r.QtySold.Float64Value(); err == nil && f.Valid {
				qtyFloat = f.Float64
			}
		}
		topParts = append(topParts, TopPartItem{
			ID:           r.ID,
			Name:         r.Name,
			Sku:          r.Sku,
			QtySold:      qtyFloat,
			TotalRevenue: r.TotalRevenue,
		})
	}

	outRows, err := s.queries.GetTopOutstandingInvoices(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get top outstanding: %w", err)
	}

	topOutstanding := make([]TopOutstandingInvoiceItem, 0, len(outRows))
	for _, r := range outRows {
		item := TopOutstandingInvoiceItem{
			ID:            r.ID,
			InvoiceNo:     r.InvoiceNo,
			CustomerID:    r.CustomerID,
			CustomerName:  r.CustomerName,
			CustomerPhone: r.CustomerPhone,
			Total:         r.Total,
			PaidTotal:     r.PaidTotal,
			Balance:       r.Balance,
			IssuedAt:      r.IssuedAt.Time.Format(time.RFC3339),
		}
		if r.PlateNo.Valid {
			item.PlateNo = &r.PlateNo.String
		}
		topOutstanding = append(topOutstanding, item)
	}

	return map[string]interface{}{
		"top_services":    topServices,
		"top_parts":       topParts,
		"top_outstanding": topOutstanding,
	}, nil
}

func (s *Service) GetEndOfDayReport(ctx context.Context, dateStr string) (*EndOfDayReportResponse, error) {
	loc := time.Now().Location()
	var targetDate time.Time
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = t
		} else {
			targetDate = time.Now()
		}
	} else {
		targetDate = time.Now()
	}

	start := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, loc)
	end := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 999999999, loc)

	startTz := pgtype.Timestamptz{Time: start, Valid: true}
	endTz := pgtype.Timestamptz{Time: end, Valid: true}

	// 1. Revenue
	rev, err := s.queries.GetRevenueSummary(ctx, store.GetRevenueSummaryParams{
		CreatedAt:   startTz,
		CreatedAt_2: endTz,
	})
	if err != nil {
		return nil, err
	}

	// 2. Completed Jobs
	jobsCount, err := s.queries.GetCompletedJobsSummary(ctx, store.GetCompletedJobsSummaryParams{
		CompletedAt:   startTz,
		CompletedAt_2: endTz,
	})
	if err != nil {
		return nil, err
	}

	// 3. Till Sessions for this day
	sessionRows, err := s.queries.GetDayCashSessions(ctx, store.GetDayCashSessionsParams{
		OpenedAt:   startTz,
		OpenedAt_2: endTz,
	})
	if err != nil {
		return nil, err
	}

	var (
		totalOpeningFloat   int64
		totalClosingCounted int64
		totalExpectedCash   int64
		totalDifference     int64
	)

	sessionItems := make([]CashSessionSummaryItem, 0, len(sessionRows))
	for _, r := range sessionRows {
		totalOpeningFloat += r.OpeningFloat
		totalExpectedCash += r.ExpectedCash
		if r.ClosingCounted.Valid {
			totalClosingCounted += r.ClosingCounted.Int64
		}
		if r.Difference.Valid {
			totalDifference += r.Difference.Int64
		}

		item := CashSessionSummaryItem{
			ID:           r.ID,
			SessionNo:    r.SessionNo,
			UserName:     r.UserName,
			OpenedAt:     r.OpenedAt.Time.Format(time.RFC3339),
			OpeningFloat: r.OpeningFloat,
			ExpectedCash: r.ExpectedCash,
			Status:       r.Status,
		}
		if r.ClosedAt.Valid {
			cStr := r.ClosedAt.Time.Format(time.RFC3339)
			item.ClosedAt = &cStr
		}
		if r.ClosingCounted.Valid {
			item.ClosingCounted = &r.ClosingCounted.Int64
		}
		if r.Difference.Valid {
			item.Difference = &r.Difference.Int64
		}
		sessionItems = append(sessionItems, item)
	}

	return &EndOfDayReportResponse{
		Date:                start.Format("2006-01-02"),
		TotalCollected:      rev.TotalCollected,
		CashCollected:       rev.CashCollected,
		CardCollected:       rev.CardCollected,
		BankCollected:       rev.BankCollected,
		InvoicesPaidCount:   rev.PaidInvoicesCount,
		JobsCompletedCount:  jobsCount,
		TotalOpeningFloat:   totalOpeningFloat,
		TotalClosingCounted: totalClosingCounted,
		TotalExpectedCash:   totalExpectedCash,
		TotalDifference:     totalDifference,
		Sessions:            sessionItems,
	}, nil
}
