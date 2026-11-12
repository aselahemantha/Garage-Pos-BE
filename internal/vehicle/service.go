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
