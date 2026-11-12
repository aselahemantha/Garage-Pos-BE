package vehicle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrVehicleNotFound     = errors.New("vehicle not found")
	ErrDuplicatePlate      = errors.New("vehicle plate already registered")
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

func (s *Service) Create(ctx context.Context, req CreateVehicleRequest) (*VehicleResponse, error) {
	// Standardize plate number: stored uppercase with single spaces/dashes clean
	cleanPlate := strings.ToUpper(strings.TrimSpace(req.PlateNo))

	existing, err := s.queries.GetVehicleByPlate(ctx, cleanPlate)
	if err == nil && existing.ID > 0 {
		return nil, ErrDuplicatePlate
	}

	params := store.CreateVehicleParams{
		CustomerID: req.CustomerID,
		PlateNo:    cleanPlate,
		Make:       strings.TrimSpace(req.Make),
		Model:      strings.TrimSpace(req.Model),
	}

	if req.Year != nil {
		params.Year = pgtype.Int4{Int32: *req.Year, Valid: true}
	}
	if req.VIN != nil {
		params.Vin = pgtype.Text{String: *req.VIN, Valid: true}
	}
	if req.FuelType != nil {
		params.FuelType = pgtype.Text{String: *req.FuelType, Valid: true}
	}
	if req.Color != nil {
		params.Color = pgtype.Text{String: *req.Color, Valid: true}
	}
	if req.LastMileage != nil {
		params.LastMileage = pgtype.Int4{Int32: *req.LastMileage, Valid: true}
	}

	row, err := s.queries.CreateVehicle(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create vehicle: %w", err)
	}

	return toVehicleResponse(row), nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*VehicleResponse, error) {
	row, err := s.queries.GetVehicleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVehicleNotFound
		}
		return nil, err
	}
	return toVehicleResponse(row), nil
}

func (s *Service) GetByPlate(ctx context.Context, plate string) (*VehicleResponse, error) {
	cleanPlate := strings.ToUpper(strings.TrimSpace(plate))
	row, err := s.queries.GetVehicleByPlate(ctx, cleanPlate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVehicleNotFound
		}
		return nil, err
	}
	return toVehicleResponse(row), nil
}

func (s *Service) ListByCustomerID(ctx context.Context, customerID int64) ([]VehicleResponse, error) {
	rows, err := s.queries.ListVehiclesByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	res := make([]VehicleResponse, len(rows))
	for i, r := range rows {
		res[i] = *toVehicleResponse(r)
	}
	return res, nil
}

func (s *Service) Search(ctx context.Context, query string, page, pageSize int) ([]VehicleResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := s.queries.SearchVehicles(ctx, store.SearchVehiclesParams{
		Column1: query,
		Limit:   int32(pageSize),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := s.queries.CountVehicles(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	res := make([]VehicleResponse, len(rows))
	for i, r := range rows {
		res[i] = *toVehicleResponse(r)
	}

	return res, total, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateVehicleRequest) (*VehicleResponse, error) {
	params := store.UpdateVehicleParams{
		ID: id,
	}

	if req.Make != nil {
		params.Make = pgtype.Text{String: *req.Make, Valid: true}
	}
	if req.Model != nil {
		params.Model = pgtype.Text{String: *req.Model, Valid: true}
	}
	if req.Year != nil {
		params.Year = pgtype.Int4{Int32: *req.Year, Valid: true}
	}
	if req.VIN != nil {
		params.Vin = pgtype.Text{String: *req.VIN, Valid: true}
	}
	if req.FuelType != nil {
		params.FuelType = pgtype.Text{String: *req.FuelType, Valid: true}
	}
	if req.Color != nil {
		params.Color = pgtype.Text{String: *req.Color, Valid: true}
	}
	if req.LastMileage != nil {
		params.LastMileage = pgtype.Int4{Int32: *req.LastMileage, Valid: true}
	}

	row, err := s.queries.UpdateVehicle(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVehicleNotFound
		}
		return nil, err
	}

	return toVehicleResponse(row), nil
}

func (s *Service) GetHistory(ctx context.Context, id int64) (*VehicleHistoryResponse, error) {
	v, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 1. Get customer info
	var cust VehicleCustomerInfo
	var emailPg, addrPg pgtype.Text
	err = s.pool.QueryRow(ctx, `
		SELECT id, name, phone, email, address
		FROM customers
		WHERE id = $1
	`, v.CustomerID).Scan(&cust.ID, &cust.Name, &cust.Phone, &emailPg, &addrPg)
	if err != nil {
		return nil, fmt.Errorf("query customer: %w", err)
	}
	if emailPg.Valid {
		cust.Email = &emailPg.String
	}
	if addrPg.Valid {
		cust.Address = &addrPg.String
	}

	// 2. Get past jobs
	jobRows, err := s.pool.Query(ctx, `
		SELECT j.id, j.job_no, j.status, j.mileage_in, j.complaint, j.notes, u.name,
		       COALESCE(SUM(ji.line_total), 0)::bigint, j.completed_at, j.created_at
		FROM job_cards j
		JOIN users u ON j.advisor_id = u.id
		LEFT JOIN job_items ji ON ji.job_card_id = j.id
		WHERE j.vehicle_id = $1
		GROUP BY j.id, u.name
		ORDER BY j.created_at DESC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query vehicle jobs: %w", err)
	}
	defer jobRows.Close()

	jobs := make([]VehicleHistoryJob, 0)
	var jobIDs []int64

	for jobRows.Next() {
		var j VehicleHistoryJob
		var compPg, notesPg pgtype.Text
		var compAtPg, creatAtPg pgtype.Timestamptz

		if err := jobRows.Scan(
			&j.ID, &j.JobNo, &j.Status, &j.MileageIn, &compPg, &notesPg, &j.AdvisorName,
			&j.TotalAmount, &compAtPg, &creatAtPg,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}

		if compPg.Valid {
			j.Complaint = &compPg.String
		}
		if notesPg.Valid {
			j.Notes = &notesPg.String
		}
		if compAtPg.Valid {
			t := compAtPg.Time
			j.CompletedAt = &t
		}
		j.CreatedAt = creatAtPg.Time
		j.Items = []VehicleHistoryJobItem{}

		jobs = append(jobs, j)
		jobIDs = append(jobIDs, j.ID)
	}

	// 3. Get job items for each job if there are any jobs
	if len(jobIDs) > 0 {
		itemRows, err := s.pool.Query(ctx, `
			SELECT job_card_id, description, item_type, qty, line_total
			FROM job_items
			WHERE job_card_id = ANY($1)
			ORDER BY id ASC
		`, jobIDs)
		if err == nil {
			defer itemRows.Close()
			for itemRows.Next() {
				var jcID int64
				var item VehicleHistoryJobItem
				var qtyNum pgtype.Numeric
				if err := itemRows.Scan(&jcID, &item.Description, &item.ItemType, &qtyNum, &item.LineTotal); err == nil {
					f, _ := qtyNum.Float64Value()
					item.Qty = f.Float64
					for idx := range jobs {
						if jobs[idx].ID == jcID {
							jobs[idx].Items = append(jobs[idx].Items, item)
							break
						}
					}
				}
			}
		}
	}

	// 4. Get invoices for vehicle
	invRows, err := s.pool.Query(ctx, `
		SELECT id, invoice_no, total, paid_total, balance, status, issued_at
		FROM invoices
		WHERE vehicle_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query vehicle invoices: %w", err)
	}
	defer invRows.Close()

	invoices := make([]VehicleHistoryInvoice, 0)
	for invRows.Next() {
		var inv VehicleHistoryInvoice
		var issuedAtPg pgtype.Timestamptz
		if err := invRows.Scan(
			&inv.ID, &inv.InvoiceNo, &inv.Total, &inv.PaidTotal, &inv.Balance, &inv.Status, &issuedAtPg,
		); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		inv.IssuedAt = issuedAtPg.Time
		invoices = append(invoices, inv)
	}

	return &VehicleHistoryResponse{
		Vehicle:  *v,
		Customer: cust,
		Jobs:     jobs,
		Invoices: invoices,
	}, nil
}

func toVehicleResponse(v store.Vehicle) *VehicleResponse {
	resp := &VehicleResponse{
		ID:         v.ID,
		CustomerID: v.CustomerID,
		PlateNo:    v.PlateNo,
		Make:       v.Make,
		Model:      v.Model,
		CreatedAt:  v.CreatedAt.Time,
		UpdatedAt:  v.UpdatedAt.Time,
	}

	if v.Year.Valid {
		y := v.Year.Int32
		resp.Year = &y
	}
	if v.Vin.Valid {
		resp.VIN = &v.Vin.String
	}
	if v.FuelType.Valid {
		resp.FuelType = &v.FuelType.String
	}
	if v.Color.Valid {
		resp.Color = &v.Color.String
	}
	if v.LastMileage.Valid {
		m := v.LastMileage.Int32
		resp.LastMileage = &m
	}

	return resp
}
