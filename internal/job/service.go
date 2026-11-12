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

func (s *Service) AddItem(ctx context.Context, jobID int64, input CreateJobItemInput) (*JobDetailResponse, error) {
	card, err := s.queries.GetJobCardByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if card.Status == "completed" || card.Status == "delivered" || card.Status == "cancelled" {
		return nil, ErrJobAlreadyCompleted
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	qtyNumeric, _ := floatToNumeric(input.Qty)
	rawTotal := int64(math.Round(float64(input.UnitPrice) * input.Qty))
	lineTotal := rawTotal - input.Discount
	if lineTotal < 0 {
		lineTotal = 0
	}

	var serviceID pgtype.Int8
	if input.ServiceID != nil {
		serviceID = pgtype.Int8{Int64: *input.ServiceID, Valid: true}
	}
	var partID pgtype.Int8
	if input.PartID != nil {
		partID = pgtype.Int8{Int64: *input.PartID, Valid: true}
	}
	var mechanicID pgtype.Int8
	if input.MechanicID != nil {
		mechanicID = pgtype.Int8{Int64: *input.MechanicID, Valid: true}
	}

	createdItem, err := qtx.CreateJobItem(ctx, store.CreateJobItemParams{
		JobCardID:   jobID,
		ItemType:    input.ItemType,
		ServiceID:   serviceID,
		PartID:      partID,
		Description: input.Description,
		Qty:         qtyNumeric,
		UnitPrice:   input.UnitPrice,
		Discount:    input.Discount,
		LineTotal:   lineTotal,
		MechanicID:  mechanicID,
	})
	if err != nil {
		return nil, fmt.Errorf("create job item: %w", err)
	}

	if input.ItemType == "part" && input.PartID != nil {
		negQty, _ := floatToNumeric(-input.Qty)
		_, err = qtx.AddStockMovement(ctx, store.AddStockMovementParams{
			PartID:   *input.PartID,
			Qty:      negQty,
			Type:     "job_use",
			RefType:  pgtype.Text{String: "job_item", Valid: true},
			RefID:    pgtype.Int8{Int64: createdItem.ID, Valid: true},
			UnitCost: pgtype.Int8{Int64: input.UnitPrice, Valid: true},
			Note:     pgtype.Text{String: fmt.Sprintf("Added to Job %s", card.JobNo), Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("record stock deduction: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, jobID)
}

func (s *Service) RemoveItem(ctx context.Context, jobID int64, itemID int64) (*JobDetailResponse, error) {
	card, err := s.queries.GetJobCardByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if card.Status == "completed" || card.Status == "delivered" || card.Status == "cancelled" {
		return nil, ErrJobAlreadyCompleted
	}

	item, err := s.queries.GetJobItemByID(ctx, itemID)
	if err != nil {
		return nil, errors.New("item not found")
	}
	if item.JobCardID != jobID {
		return nil, errors.New("item does not belong to this job card")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	if item.ItemType == "part" && item.PartID.Valid {
		_, _ = qtx.AddStockMovement(ctx, store.AddStockMovementParams{
			PartID:  item.PartID.Int64,
			Qty:     item.Qty,
			Type:    "job_return",
			RefType: pgtype.Text{String: "job_item", Valid: true},
			RefID:   pgtype.Int8{Int64: item.ID, Valid: true},
			Note:    pgtype.Text{String: fmt.Sprintf("Removed from Job %s", card.JobNo), Valid: true},
		})
	}

	if err := qtx.DeleteJobItem(ctx, itemID); err != nil {
		return nil, fmt.Errorf("delete job item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, jobID)
}

func (s *Service) UpdateJob(ctx context.Context, jobID int64, req UpdateJobCardRequest) (*JobDetailResponse, error) {
	var mileageParam pgtype.Int4
	if req.MileageIn != nil {
		mileageParam = pgtype.Int4{Int32: *req.MileageIn, Valid: true}
	}
	var complaintParam pgtype.Text
	if req.Complaint != nil {
		complaintParam = pgtype.Text{String: *req.Complaint, Valid: true}
	}
	var notesParam pgtype.Text
	if req.Notes != nil {
		notesParam = pgtype.Text{String: *req.Notes, Valid: true}
	}
	var promisedParam pgtype.Timestamptz
	if req.PromisedBy != nil && *req.PromisedBy != "" {
		if t, err := time.Parse(time.RFC3339, *req.PromisedBy); err == nil {
			promisedParam = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	_, err := s.queries.UpdateJobCard(ctx, store.UpdateJobCardParams{
		ID:         jobID,
		MileageIn:  mileageParam,
		Complaint:  complaintParam,
		Notes:      notesParam,
		PromisedBy: promisedParam,
	})
	if err != nil {
		return nil, fmt.Errorf("update job card: %w", err)
	}

	return s.GetByID(ctx, jobID)
}

func floatToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	err := n.Scan(fmt.Sprintf("%.3f", f))
	return n, err
}
