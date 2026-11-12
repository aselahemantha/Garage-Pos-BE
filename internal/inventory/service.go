package inventory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPartNotFound      = errors.New("part not found")
	ErrDuplicateSKU      = errors.New("part with this SKU already exists")
	ErrInsufficientStock = errors.New("insufficient stock on hand")
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

func (s *Service) List(ctx context.Context, search string, category *string, activeOnly *bool) ([]PartResponse, error) {
	params := store.ListPartsParams{
		Column1: strings.TrimSpace(search),
	}
	if category != nil {
		params.Category = pgtype.Text{String: *category, Valid: true}
	}
	if activeOnly != nil {
		params.IsActive = pgtype.Bool{Bool: *activeOnly, Valid: true}
	}

	rows, err := s.queries.ListParts(ctx, params)
	if err != nil {
		return nil, err
	}

	res := make([]PartResponse, len(rows))
	for i, r := range rows {
		reorderF, _ := r.ReorderLevel.Float64Value()
		stockF, _ := r.StockOnHand.Float64Value()

		res[i] = PartResponse{
			ID:           r.ID,
			SKU:          r.Sku,
			Name:         r.Name,
			Unit:         r.Unit,
			CostPrice:    r.CostPrice,
			SellPrice:    r.SellPrice,
			ReorderLevel: reorderF.Float64,
			StockOnHand:  stockF.Float64,
			IsActive:     r.IsActive,
			CreatedAt:    r.CreatedAt.Time,
			UpdatedAt:    r.UpdatedAt.Time,
		}
		if r.Category.Valid {
			res[i].Category = &r.Category.String
		}
	}

	return res, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*PartResponse, error) {
	r, err := s.queries.GetPartByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPartNotFound
		}
		return nil, err
	}

	reorderF, _ := r.ReorderLevel.Float64Value()
	stockF, _ := r.StockOnHand.Float64Value()

	resp := &PartResponse{
		ID:           r.ID,
		SKU:          r.Sku,
		Name:         r.Name,
		Unit:         r.Unit,
		CostPrice:    r.CostPrice,
		SellPrice:    r.SellPrice,
		ReorderLevel: reorderF.Float64,
		StockOnHand:  stockF.Float64,
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
	if r.Category.Valid {
		resp.Category = &r.Category.String
	}

	return resp, nil
}

func (s *Service) Create(ctx context.Context, req CreatePartRequest) (*PartResponse, error) {
	cleanSKU := strings.ToUpper(strings.TrimSpace(req.SKU))

	existing, err := s.queries.GetPartBySKU(ctx, cleanSKU)
	if err == nil && existing.ID > 0 {
		return nil, ErrDuplicateSKU
	}

	reorderNumeric := pgtype.Numeric{}
	_ = reorderNumeric.Scan(fmt.Sprintf("%.3f", req.ReorderLevel))

	params := store.CreatePartParams{
		Sku:          cleanSKU,
		Name:         strings.TrimSpace(req.Name),
		Unit:         strings.TrimSpace(req.Unit),
		CostPrice:    req.CostPrice,
		SellPrice:    req.SellPrice,
		ReorderLevel: reorderNumeric,
		IsActive:     true,
	}
	if req.Category != nil {
		params.Category = pgtype.Text{String: strings.TrimSpace(*req.Category), Valid: true}
	}

	row, err := s.queries.CreatePart(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create part: %w", err)
	}

	return s.GetByID(ctx, row.ID)
}

func (s *Service) StockIn(ctx context.Context, partID int64, userID int64, req StockInRequest) error {
	part, err := s.GetByID(ctx, partID)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	qtyNumeric := pgtype.Numeric{}
	_ = qtyNumeric.Scan(fmt.Sprintf("%.3f", req.Qty))

	noteParam := pgtype.Text{}
	if req.Note != nil {
		noteParam = pgtype.Text{String: *req.Note, Valid: true}
	}

	// 1. Add positive stock movement
	_, err = qtx.AddStockMovement(ctx, store.AddStockMovementParams{
		PartID:    part.ID,
		Qty:       qtyNumeric,
		Type:      "stock_in",
		UnitCost:  pgtype.Int8{Int64: req.UnitCost, Valid: true},
		CreatedBy: pgtype.Int8{Int64: userID, Valid: true},
		Note:      noteParam,
	})
	if err != nil {
		return fmt.Errorf("record stock_in: %w", err)
	}

	// 2. Update part's latest cost price
	_, err = qtx.UpdatePart(ctx, store.UpdatePartParams{
		ID:        part.ID,
		CostPrice: pgtype.Int8{Int64: req.UnitCost, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update part cost price: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *Service) SeedDefaultParts(ctx context.Context) error {
	parts, err := s.List(ctx, "", nil, nil)
	if err != nil {
		return err
	}
	if len(parts) > 0 {
		return nil
	}

	slog.Info("seeding default parts from design guideline...")

	type seedPart struct {
		SKU          string
		Name         string
		Category     string
		Unit         string
		CostPrice    int64
		SellPrice    int64
		ReorderLevel float64
		InitialStock float64
	}

	items := []seedPart{
		{SKU: "BRK-PAD-F", Name: "Brake Pads (Front Set)", Category: "Brakes", Unit: "set", CostPrice: 760000, SellPrice: 980000, ReorderLevel: 3, InitialStock: 6},
		{SKU: "FLT-OIL-SZ", Name: "Oil Filter (Suzuki)", Category: "Filters", Unit: "each", CostPrice: 135000, SellPrice: 185000, ReorderLevel: 5, InitialStock: 9},
		{SKU: "FLT-OIL-TY", Name: "Oil Filter (Toyota)", Category: "Filters", Unit: "each", CostPrice: 120000, SellPrice: 180000, ReorderLevel: 5, InitialStock: 3},
		{SKU: "FLT-AIR-SZ", Name: "Air Filter (Suzuki)", Category: "Filters", Unit: "each", CostPrice: 250000, SellPrice: 340000, ReorderLevel: 4, InitialStock: 2},
		{SKU: "FLT-AIR-TA", Name: "Air Filter (Toyota Aqua)", Category: "Filters", Unit: "each", CostPrice: 270000, SellPrice: 360000, ReorderLevel: 4, InitialStock: 7},
		{SKU: "OIL-5W30-4", Name: "Engine Oil 5W-30 (4L)", Category: "Oils and fluids", Unit: "can", CostPrice: 940000, SellPrice: 1120000, ReorderLevel: 6, InitialStock: 14},
		{SKU: "CLT-1L", Name: "Coolant (1L)", Category: "Oils and fluids", Unit: "bottle", CostPrice: 98000, SellPrice: 135000, ReorderLevel: 6, InitialStock: 0},
		{SKU: "WIP-20", Name: "Wiper Blade 20\"", Category: "Other", Unit: "each", CostPrice: 110000, SellPrice: 165000, ReorderLevel: 4, InitialStock: 11},
	}

	for _, item := range items {
		p, err := s.Create(ctx, CreatePartRequest{
			SKU:          item.SKU,
			Name:         item.Name,
			Category:     &item.Category,
			Unit:         item.Unit,
			CostPrice:    item.CostPrice,
			SellPrice:    item.SellPrice,
			ReorderLevel: item.ReorderLevel,
		})
		if err != nil {
			return err
		}

		if item.InitialStock > 0 {
			qtyNumeric := pgtype.Numeric{}
			_ = qtyNumeric.Scan(fmt.Sprintf("%.3f", item.InitialStock))
			_, _ = s.queries.AddStockMovement(ctx, store.AddStockMovementParams{
				PartID:   p.ID,
				Qty:      qtyNumeric,
				Type:     "stock_in",
				UnitCost: pgtype.Int8{Int64: item.CostPrice, Valid: true},
				Note:     pgtype.Text{String: "Initial stock opening", Valid: true},
			})
		}
	}

	slog.Info("default parts seeded successfully")
	return nil
}
