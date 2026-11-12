package customer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"garage-pos-be/internal/store"
	"garage-pos-be/internal/vehicle"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrDuplicatePhone   = errors.New("customer phone number already registered")
)

type Service struct {
	queries        *store.Queries
	pool           *pgxpool.Pool
	vehicleService *vehicle.Service
}

func NewService(queries *store.Queries, pool *pgxpool.Pool, vehicleService *vehicle.Service) *Service {
	return &Service{
		queries:        queries,
		pool:           pool,
		vehicleService: vehicleService,
	}
}

func (s *Service) Create(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error) {
	cleanPhone := strings.TrimSpace(req.Phone)

	// Check phone uniqueness
	existing, err := s.queries.GetCustomerByPhone(ctx, cleanPhone)
	if err == nil && existing.ID > 0 {
		return nil, ErrDuplicatePhone
	}

	params := store.CreateCustomerParams{
		Name:     strings.TrimSpace(req.Name),
		Phone:    cleanPhone,
		IsActive: true,
	}

	if req.Email != nil {
		params.Email = pgtype.Text{String: strings.TrimSpace(*req.Email), Valid: true}
	}
	if req.Address != nil {
		params.Address = pgtype.Text{String: strings.TrimSpace(*req.Address), Valid: true}
	}
	if req.Notes != nil {
		params.Notes = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}

	row, err := s.queries.CreateCustomer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return toCustomerResponse(row), nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*CustomerDetailResponse, error) {
	row, err := s.queries.GetCustomerByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	vehicles, err := s.vehicleService.ListByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list customer vehicles: %w", err)
	}

	base := toCustomerResponse(row)
	return &CustomerDetailResponse{
		CustomerResponse: *base,
		Vehicles:         vehicles,
	}, nil
}

func (s *Service) GetByPhone(ctx context.Context, phone string) (*CustomerDetailResponse, error) {
	cleanPhone := strings.TrimSpace(phone)
	row, err := s.queries.GetCustomerByPhone(ctx, cleanPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	vehicles, err := s.vehicleService.ListByCustomerID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("list customer vehicles: %w", err)
	}

	base := toCustomerResponse(row)
	return &CustomerDetailResponse{
		CustomerResponse: *base,
		Vehicles:         vehicles,
	}, nil
}

func (s *Service) Search(ctx context.Context, query string, page, pageSize int) ([]CustomerResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := s.queries.SearchCustomers(ctx, store.SearchCustomersParams{
		Column1: query,
		Limit:   int32(pageSize),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := s.queries.CountCustomers(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	res := make([]CustomerResponse, len(rows))
	for i, r := range rows {
		res[i] = *toCustomerResponse(r)
	}

	return res, total, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateCustomerRequest) (*CustomerResponse, error) {
	params := store.UpdateCustomerParams{
		ID: id,
	}

	if req.Name != nil {
		params.Name = pgtype.Text{String: strings.TrimSpace(*req.Name), Valid: true}
	}
	if req.Phone != nil {
		params.Phone = pgtype.Text{String: strings.TrimSpace(*req.Phone), Valid: true}
	}
	if req.Email != nil {
		params.Email = pgtype.Text{String: strings.TrimSpace(*req.Email), Valid: true}
	}
	if req.Address != nil {
		params.Address = pgtype.Text{String: strings.TrimSpace(*req.Address), Valid: true}
	}
	if req.Notes != nil {
		params.Notes = pgtype.Text{String: strings.TrimSpace(*req.Notes), Valid: true}
	}
	if req.IsActive != nil {
		params.IsActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	row, err := s.queries.UpdateCustomer(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	return toCustomerResponse(row), nil
}

func toCustomerResponse(c store.Customer) *CustomerResponse {
	resp := &CustomerResponse{
		ID:        c.ID,
		Name:      c.Name,
		Phone:     c.Phone,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}

	if c.Email.Valid {
		resp.Email = &c.Email.String
	}
	if c.Address.Valid {
		resp.Address = &c.Address.String
	}
	if c.Notes.Valid {
		resp.Notes = &c.Notes.String
	}

	return resp
}
