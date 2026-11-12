package job

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrJobNotFound         = errors.New("job card not found")
	ErrInvalidTransition  = errors.New("invalid job status transition")
	ErrJobAlreadyCompleted = errors.New("cannot modify items of completed job")
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

func (s *Service) Create(ctx context.Context, advisorID int64, req CreateJobCardRequest) (*JobDetailResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// 1. Prepare promised_by
	promisedParam := pgtype.Timestamptz{}
	if req.PromisedBy != nil && *req.PromisedBy != "" {
		t, err := time.Parse(time.RFC3339, *req.PromisedBy)
		if err == nil {
			promisedParam = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	complaintParam := pgtype.Text{}
	if req.Complaint != nil {
		complaintParam = pgtype.Text{String: strings.TrimSpace(*req.Complaint), Valid: true}
	}
	notesParam := pgtype.Text{}
	if req.Notes != nil {
		notesParam = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}

	// 2. Insert job card
	cardRow, err := qtx.CreateJobCard(ctx, store.CreateJobCardParams{
		CustomerID: req.CustomerID,
		VehicleID:  req.VehicleID,
		AdvisorID:  advisorID,
		MileageIn:  req.MileageIn,
		Complaint:  complaintParam,
		Notes:      notesParam,
		Status:     "pending",
		PromisedBy: promisedParam,
	})
	if err != nil {
		return nil, fmt.Errorf("create job card: %w", err)
	}

	// 3. Update vehicle last_mileage
	_, _ = qtx.UpdateVehicle(ctx, store.UpdateVehicleParams{
		ID:          req.VehicleID,
		LastMileage: pgtype.Int4{Int32: req.MileageIn, Valid: true},
	})

	// 4. Insert items
	var totalMinor int64 = 0
	itemResponses := make([]JobItemResponse, 0, len(req.Items))

	for _, item := range req.Items {
		qtyNumeric := pgtype.Numeric{}
		_ = qtyNumeric.Scan(fmt.Sprintf("%.3f", item.Qty))

		// line_total = round(qty * unit_price) - discount
		lineTotal := int64(math.Round(item.Qty*float64(item.UnitPrice))) - item.Discount
		if lineTotal < 0 {
			lineTotal = 0
		}
		totalMinor += lineTotal

		params := store.CreateJobItemParams{
			JobCardID:   cardRow.ID,
			ItemType:    item.ItemType,
			Description: strings.TrimSpace(item.Description),
			Qty:         qtyNumeric,
			UnitPrice:   item.UnitPrice,
			Discount:    item.Discount,
			LineTotal:   lineTotal,
		}

		if item.ServiceID != nil {
			params.ServiceID = pgtype.Int8{Int64: *item.ServiceID, Valid: true}
		}
		if item.PartID != nil {
			params.PartID = pgtype.Int8{Int64: *item.PartID, Valid: true}
		}
		if item.MechanicID != nil {
			params.MechanicID = pgtype.Int8{Int64: *item.MechanicID, Valid: true}
		}

		itemRow, err := qtx.CreateJobItem(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("create job item: %w", err)
		}

		// If item is part, deduct stock (job_use movement: negative qty)
		if item.ItemType == "part" && item.PartID != nil {
			negQty := -item.Qty
			negQtyNumeric := pgtype.Numeric{}
			_ = negQtyNumeric.Scan(fmt.Sprintf("%.3f", negQty))

			_, err = qtx.AddStockMovement(ctx, store.AddStockMovementParams{
				PartID:    *item.PartID,
				Qty:       negQtyNumeric,
				Type:      "job_use",
				RefType:   pgtype.Text{String: "job_item", Valid: true},
				RefID:     pgtype.Int8{Int64: itemRow.ID, Valid: true},
				CreatedBy: pgtype.Int8{Int64: advisorID, Valid: true},
				Note:      pgtype.Text{String: fmt.Sprintf("Used on %s", cardRow.JobNo), Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("deduct stock for part: %w", err)
			}
		}

		itemResponses = append(itemResponses, JobItemResponse{
			ID:          itemRow.ID,
			JobCardID:   itemRow.JobCardID,
			ItemType:    itemRow.ItemType,
			Description: itemRow.Description,
			Qty:         item.Qty,
			UnitPrice:   itemRow.UnitPrice,
			Discount:    itemRow.Discount,
			LineTotal:   itemRow.LineTotal,
			CreatedAt:   itemRow.CreatedAt.Time,
			UpdatedAt:   itemRow.UpdatedAt.Time,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Re-fetch full details
	return s.GetByID(ctx, cardRow.ID)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*JobDetailResponse, error) {
	c, err := s.queries.GetJobCardByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	items, err := s.queries.ListJobItems(ctx, id)
	if err != nil {
		return nil, err
	}

	var totalMinor int64 = 0
	itemResps := make([]JobItemResponse, len(items))
	for i, r := range items {
		qtyF, _ := r.Qty.Float64Value()
		totalMinor += r.LineTotal

		itemResps[i] = JobItemResponse{
			ID:          r.ID,
			JobCardID:   r.JobCardID,
			ItemType:    r.ItemType,
			Description: r.Description,
			Qty:         qtyF.Float64,
			UnitPrice:   r.UnitPrice,
			Discount:    r.Discount,
			LineTotal:   r.LineTotal,
			CreatedAt:   r.CreatedAt.Time,
			UpdatedAt:   r.UpdatedAt.Time,
		}
		if r.ServiceID.Valid {
			itemResps[i].ServiceID = &r.ServiceID.Int64
		}
		if r.PartID.Valid {
			itemResps[i].PartID = &r.PartID.Int64
		}
		if r.MechanicID.Valid {
			itemResps[i].MechanicID = &r.MechanicID.Int64
		}
		if r.MechanicName.Valid {
			itemResps[i].MechanicName = &r.MechanicName.String
		}
	}

	cardResp := JobCardResponse{
		ID:            c.ID,
		JobNo:         c.JobNo,
		CustomerID:    c.CustomerID,
		CustomerName:  c.CustomerName,
		CustomerPhone: c.CustomerPhone,
		VehicleID:     c.VehicleID,
		PlateNo:       c.PlateNo,
		VehicleMake:   c.VehicleMake,
		VehicleModel:  c.VehicleModel,
		AdvisorID:     c.AdvisorID,
		AdvisorName:   c.AdvisorName,
		MileageIn:     c.MileageIn,
		Status:        c.Status,
		CreatedAt:     c.CreatedAt.Time,
		UpdatedAt:     c.UpdatedAt.Time,
	}

	if c.VehicleYear.Valid {
		cardResp.VehicleYear = &c.VehicleYear.Int32
	}
	if c.Complaint.Valid {
		cardResp.Complaint = &c.Complaint.String
	}
	if c.Notes.Valid {
		cardResp.Notes = &c.Notes.String
	}
	if c.PromisedBy.Valid {
		cardResp.PromisedBy = &c.PromisedBy.Time
	}
	if c.StartedAt.Valid {
		cardResp.StartedAt = &c.StartedAt.Time
	}
	if c.CompletedAt.Valid {
		cardResp.CompletedAt = &c.CompletedAt.Time
	}
	if c.DeliveredAt.Valid {
		cardResp.DeliveredAt = &c.DeliveredAt.Time
	}

	return &JobDetailResponse{
		JobCardResponse: cardResp,
		Items:           itemResps,
		TotalMinor:      totalMinor,
	}, nil
}

func (s *Service) ListBoard(ctx context.Context) ([]JobCardResponse, error) {
	rows, err := s.queries.ListBoardJobs(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]JobCardResponse, len(rows))
	for i, c := range rows {
		res[i] = JobCardResponse{
			ID:            c.ID,
			JobNo:         c.JobNo,
			CustomerID:    c.CustomerID,
			CustomerName:  c.CustomerName,
			CustomerPhone: c.CustomerPhone,
			VehicleID:     c.VehicleID,
			PlateNo:       c.PlateNo,
			VehicleMake:   c.VehicleMake,
			VehicleModel:  c.VehicleModel,
			AdvisorID:     c.AdvisorID,
			MileageIn:     c.MileageIn,
			Status:        c.Status,
			CreatedAt:     c.CreatedAt.Time,
		}

		if c.VehicleYear.Valid {
			res[i].VehicleYear = &c.VehicleYear.Int32
		}
		if c.Complaint.Valid {
			res[i].Complaint = &c.Complaint.String
		}
		if c.Notes.Valid {
			res[i].Notes = &c.Notes.String
		}
		if c.PromisedBy.Valid {
			res[i].PromisedBy = &c.PromisedBy.Time
		}
		if c.StartedAt.Valid {
			res[i].StartedAt = &c.StartedAt.Time
		}
		if c.CompletedAt.Valid {
			res[i].CompletedAt = &c.CompletedAt.Time
		}
		if c.DeliveredAt.Valid {
			res[i].DeliveredAt = &c.DeliveredAt.Time
		}
	}

	return res, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, newStatus string) (*JobDetailResponse, error) {
	existing, err := s.queries.GetJobCardByID(ctx, id)
	if err != nil {
		return nil, ErrJobNotFound
	}

	// Validate transitions from Spec 6.1:
	// pending -> in_progress, cancelled
	// in_progress -> completed, cancelled
	// completed -> delivered
	valid := false
	curr := existing.Status
	switch curr {
	case "pending":
		valid = newStatus == "in_progress" || newStatus == "cancelled"
	case "in_progress":
		valid = newStatus == "completed" || newStatus == "cancelled"
	case "completed":
		valid = newStatus == "delivered"
	default:
		valid = false
	}

	if !valid {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, curr, newStatus)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// If cancelled: return all parts to stock (positive movement job_return)
	if newStatus == "cancelled" {
		items, err := qtx.ListJobItems(ctx, id)
		if err == nil {
			for _, item := range items {
				if item.ItemType == "part" && item.PartID.Valid {
					_, _ = qtx.AddStockMovement(ctx, store.AddStockMovementParams{
						PartID:  item.PartID.Int64,
						Qty:     item.Qty, // return positive quantity
						Type:    "job_return",
						RefType: pgtype.Text{String: "job_item", Valid: true},
						RefID:   pgtype.Int8{Int64: item.ID, Valid: true},
						Note:    pgtype.Text{String: fmt.Sprintf("Cancelled job %s return", existing.JobNo), Valid: true},
					})
				}
			}
		}
	}

	_, err = qtx.UpdateJobStatus(ctx, store.UpdateJobStatusParams{
		ID:     id,
		Status: newStatus,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}
