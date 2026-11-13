package catalog

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
	ErrCategoryNotFound = errors.New("service category not found")
	ErrServiceNotFound  = errors.New("service not found")
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

func (s *Service) ListCategories(ctx context.Context) ([]CategoryResponse, error) {
	rows, err := s.queries.ListServiceCategories(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]CategoryResponse, len(rows))
	for i, r := range rows {
		res[i] = CategoryResponse{
			ID:        r.ID,
			Name:      r.Name,
			SortOrder: r.SortOrder,
			CreatedAt: r.CreatedAt.Time,
			UpdatedAt: r.UpdatedAt.Time,
		}
	}
	return res, nil
}

func (s *Service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryResponse, error) {
	row, err := s.queries.CreateServiceCategory(ctx, store.CreateServiceCategoryParams{
		Name:      strings.TrimSpace(req.Name),
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return &CategoryResponse{
		ID:        row.ID,
		Name:      row.Name,
		SortOrder: row.SortOrder,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (s *Service) ListServices(ctx context.Context, categoryID *int64, activeOnly *bool) ([]ServiceResponse, error) {
	params := store.ListServicesParams{}
	if categoryID != nil {
		params.CategoryID = pgtype.Int8{Int64: *categoryID, Valid: true}
	}
	if activeOnly != nil {
		params.IsActive = pgtype.Bool{Bool: *activeOnly, Valid: true}
	}

	rows, err := s.queries.ListServices(ctx, params)
	if err != nil {
		return nil, err
	}

	res := make([]ServiceResponse, len(rows))
	for i, r := range rows {
		res[i] = *toServiceResponse(r)
	}

	if err := s.attachVehiclePrices(ctx, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) GetServiceByID(ctx context.Context, id int64) (*ServiceResponse, error) {
	row, err := s.queries.GetServiceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	svc := toServiceResponse(store.ListServicesRow{
		ID:           row.ID,
		CategoryID:   row.CategoryID,
		CategoryName: row.CategoryName,
		Name:         row.Name,
		Description:  row.Description,
		Price:        row.Price,
		EstMinutes:   row.EstMinutes,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	})
	single := []ServiceResponse{*svc}
	if err := s.attachVehiclePrices(ctx, single); err != nil {
		return nil, err
	}
	return &single[0], nil
}

func (s *Service) CreateService(ctx context.Context, req CreateServiceRequest) (*ServiceResponse, error) {
	params := store.CreateServiceParams{
		CategoryID: req.CategoryID,
		Name:       strings.TrimSpace(req.Name),
		Price:      req.Price,
		IsActive:   true,
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: strings.TrimSpace(*req.Description), Valid: true}
	}
	if req.EstMinutes != nil {
		params.EstMinutes = pgtype.Int4{Int32: *req.EstMinutes, Valid: true}
	}

	row, err := s.queries.CreateService(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}

	// Re-fetch to get category name
	return s.GetServiceByID(ctx, row.ID)
}

func (s *Service) UpdateService(ctx context.Context, id int64, req UpdateServiceRequest) (*ServiceResponse, error) {
	params := store.UpdateServiceParams{
		ID: id,
	}

	if req.CategoryID != nil {
		params.CategoryID = pgtype.Int8{Int64: *req.CategoryID, Valid: true}
	}
	if req.Name != nil {
		params.Name = pgtype.Text{String: strings.TrimSpace(*req.Name), Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: strings.TrimSpace(*req.Description), Valid: true}
	}
	if req.Price != nil {
		params.Price = pgtype.Int8{Int64: *req.Price, Valid: true}
	}
	if req.EstMinutes != nil {
		params.EstMinutes = pgtype.Int4{Int32: *req.EstMinutes, Valid: true}
	}
	if req.IsActive != nil {
		params.IsActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	_, err := s.queries.UpdateService(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}

	return s.GetServiceByID(ctx, id)
}

func (s *Service) UpsertVehiclePrice(ctx context.Context, serviceID int64, req UpsertVehiclePriceRequest) (*ServiceResponse, error) {
	// Ensure service exists
	if _, err := s.GetServiceByID(ctx, serviceID); err != nil {
		return nil, err
	}

	makeStr := strings.TrimSpace(req.VehicleMake)
	modelStr := strings.TrimSpace(req.VehicleModel)
	if modelStr == "" {
		return nil, errors.New("vehicle_model is required")
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO service_vehicle_prices (service_id, vehicle_make, vehicle_model, price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (service_id, LOWER(TRIM(vehicle_make)), LOWER(TRIM(vehicle_model)))
		DO UPDATE SET
			vehicle_make = EXCLUDED.vehicle_make,
			vehicle_model = EXCLUDED.vehicle_model,
			price = EXCLUDED.price,
			updated_at = NOW()
	`, serviceID, makeStr, modelStr, req.Price)
	if err != nil {
		return nil, fmt.Errorf("upsert vehicle price: %w", err)
	}

	return s.GetServiceByID(ctx, serviceID)
}

func (s *Service) DeleteVehiclePrice(ctx context.Context, serviceID int64, priceID int64) (*ServiceResponse, error) {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM service_vehicle_prices
		WHERE id = $1 AND service_id = $2
	`, priceID, serviceID)
	if err != nil {
		return nil, fmt.Errorf("delete vehicle price: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errors.New("vehicle price override not found")
	}

	return s.GetServiceByID(ctx, serviceID)
}

func (s *Service) attachVehiclePrices(ctx context.Context, services []ServiceResponse) error {
	if len(services) == 0 || s.pool == nil {
		return nil
	}

	ids := make([]int64, len(services))
	idxByID := make(map[int64]int, len(services))
	for i, svc := range services {
		ids[i] = svc.ID
		idxByID[svc.ID] = i
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, service_id, vehicle_make, vehicle_model, price, created_at, updated_at
		FROM service_vehicle_prices
		WHERE service_id = ANY($1)
		ORDER BY LOWER(vehicle_make), LOWER(vehicle_model)
	`, ids)
	if err != nil {
		return fmt.Errorf("query service vehicle prices: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var vp VehiclePriceResponse
		var createdAtPg, updatedAtPg pgtype.Timestamptz
		if err := rows.Scan(&vp.ID, &vp.ServiceID, &vp.VehicleMake, &vp.VehicleModel, &vp.Price, &createdAtPg, &updatedAtPg); err != nil {
			return fmt.Errorf("scan service vehicle price: %w", err)
		}
		vp.CreatedAt = createdAtPg.Time
		vp.UpdatedAt = updatedAtPg.Time
		if idx, ok := idxByID[vp.ServiceID]; ok {
			services[idx].VehiclePrices = append(services[idx].VehiclePrices, vp)
		}
	}
	return rows.Err()
}

func toServiceResponse(r store.ListServicesRow) *ServiceResponse {
	resp := &ServiceResponse{
		ID:            r.ID,
		CategoryID:    r.CategoryID,
		CategoryName:  r.CategoryName,
		Name:          r.Name,
		Price:         r.Price,
		IsActive:      r.IsActive,
		VehiclePrices: make([]VehiclePriceResponse, 0),
		CreatedAt:     r.CreatedAt.Time,
		UpdatedAt:     r.UpdatedAt.Time,
	}

	if r.Description.Valid {
		resp.Description = &r.Description.String
	}
	if r.EstMinutes.Valid {
		m := r.EstMinutes.Int32
		resp.EstMinutes = &m
	}

	return resp
}
