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

	barcodeMap := make(map[int64]string)
	bRows, bErr := s.pool.Query(ctx, `SELECT id, barcode FROM parts WHERE barcode IS NOT NULL AND barcode <> ''`)
	if bErr == nil {
		defer bRows.Close()
		for bRows.Next() {
			var pid int64
			var bc string
			if err := bRows.Scan(&pid, &bc); err == nil {
				barcodeMap[pid] = bc
			}
		}
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
		if bc, ok := barcodeMap[r.ID]; ok {
			bcCopy := bc
			res[i].Barcode = &bcCopy
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
	var bc pgtype.Text
	if err := s.pool.QueryRow(ctx, `SELECT barcode FROM parts WHERE id = $1`, id).Scan(&bc); err == nil && bc.Valid && bc.String != "" {
		resp.Barcode = &bc.String
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

	if req.Barcode != nil {
		cleanBC := strings.TrimSpace(*req.Barcode)
		if cleanBC != "" {
			if _, err := s.pool.Exec(ctx, `UPDATE parts SET barcode = $1 WHERE id = $2`, cleanBC, row.ID); err != nil {
				return nil, errors.New("part with this barcode already exists")
			}
		}
	}

	return s.GetByID(ctx, row.ID)
}

func (s *Service) Update(ctx context.Context, id int64, req UpdatePartRequest) (*PartResponse, error) {
	_, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var nameParam, catParam, unitParam pgtype.Text
	var costParam, sellParam pgtype.Int8
	var reorderParam pgtype.Numeric
	var activeParam pgtype.Bool

	if req.Name != nil {
		nameParam = pgtype.Text{String: strings.TrimSpace(*req.Name), Valid: true}
	}
	if req.Category != nil {
		catParam = pgtype.Text{String: strings.TrimSpace(*req.Category), Valid: true}
	}
	if req.Unit != nil {
		unitParam = pgtype.Text{String: strings.TrimSpace(*req.Unit), Valid: true}
	}
	if req.CostPrice != nil {
		costParam = pgtype.Int8{Int64: *req.CostPrice, Valid: true}
	}
	if req.SellPrice != nil {
		sellParam = pgtype.Int8{Int64: *req.SellPrice, Valid: true}
	}
	if req.ReorderLevel != nil {
		_ = reorderParam.Scan(fmt.Sprintf("%.3f", *req.ReorderLevel))
	}
	if req.IsActive != nil {
		activeParam = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	_, err = s.queries.UpdatePart(ctx, store.UpdatePartParams{
		ID:           id,
		Name:         nameParam,
		Category:     catParam,
		Unit:         unitParam,
		CostPrice:    costParam,
		SellPrice:    sellParam,
		ReorderLevel: reorderParam,
		IsActive:     activeParam,
	})
	if err != nil {
		return nil, err
	}

	if req.Barcode != nil {
		cleanBC := strings.TrimSpace(*req.Barcode)
		if cleanBC == "" {
			if _, err := s.pool.Exec(ctx, `UPDATE parts SET barcode = NULL WHERE id = $1`, id); err != nil {
				return nil, fmt.Errorf("clear barcode: %w", err)
			}
		} else {
			if _, err := s.pool.Exec(ctx, `UPDATE parts SET barcode = $1 WHERE id = $2`, cleanBC, id); err != nil {
				return nil, errors.New("part with this barcode already exists")
			}
		}
	}

	return s.GetByID(ctx, id)
}

func (s *Service) GetLowStockReport(ctx context.Context) (*LowStockSummaryResponse, error) {
	active := true
	parts, err := s.List(ctx, "", nil, &active)
	if err != nil {
		return nil, err
	}

	var items []LowStockItem
	var totalEstimatedCost int64
	lowCount := 0
	outCount := 0

	for _, p := range parts {
		if p.StockOnHand <= p.ReorderLevel {
			deficit := p.ReorderLevel - p.StockOnHand
			if deficit < 0 {
				deficit = 0
			}

			// Suggested replenishment
			suggested := (p.ReorderLevel * 2) - p.StockOnHand
			if suggested < p.ReorderLevel {
				suggested = p.ReorderLevel
			}
			if suggested <= 0 {
				suggested = 1
			}

			estCost := int64(suggested * float64(p.CostPrice))
			totalEstimatedCost += estCost

			if p.StockOnHand <= 0 {
				outCount++
			} else {
				lowCount++
			}

			items = append(items, LowStockItem{
				Part:                p,
				Deficit:             deficit,
				SuggestedReorderQty: suggested,
				EstimatedCost:       estCost,
			})
		}
	}

	return &LowStockSummaryResponse{
		LowStockCount:      lowCount,
		OutOfStockCount:    outCount,
		TotalEstimatedCost: totalEstimatedCost,
		Items:              items,
	}, nil
}

func (s *Service) BatchRestock(ctx context.Context, userID int64, req BatchReorderRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	for _, item := range req.Items {
		qtyNumeric := pgtype.Numeric{}
		_ = qtyNumeric.Scan(fmt.Sprintf("%.3f", item.Qty))

		note := "Batch Restock"
		if item.Note != nil && *item.Note != "" {
			note = *item.Note
		}
		if req.SupplierName != nil && *req.SupplierName != "" {
			note += fmt.Sprintf(" (Supplier: %s)", *req.SupplierName)
		}
		if req.InvoiceRef != nil && *req.InvoiceRef != "" {
			note += fmt.Sprintf(" [Inv: %s]", *req.InvoiceRef)
		}

		_, err := qtx.AddStockMovement(ctx, store.AddStockMovementParams{
			PartID:    item.PartID,
			Qty:       qtyNumeric,
			Type:      "stock_in",
			UnitCost:  pgtype.Int8{Int64: item.UnitCost, Valid: true},
			CreatedBy: pgtype.Int8{Int64: userID, Valid: true},
			Note:      pgtype.Text{String: note, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("record stock_in for part %d: %w", item.PartID, err)
		}

		if item.UnitCost > 0 {
			_, _ = qtx.UpdatePart(ctx, store.UpdatePartParams{
				ID:        item.PartID,
				CostPrice: pgtype.Int8{Int64: item.UnitCost, Valid: true},
			})
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) AdjustStock(ctx context.Context, partID int64, userID int64, req AdjustStockRequest) (*PartResponse, error) {
	part, err := s.GetByID(ctx, partID)
	if err != nil {
		return nil, err
	}

	diff := req.NewStockOnHand - part.StockOnHand
	if diff == 0 {
		return part, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	qtyNumeric := pgtype.Numeric{}
	_ = qtyNumeric.Scan(fmt.Sprintf("%.3f", diff))

	note := fmt.Sprintf("Audit adjustment: %.2f -> %.2f (%s)", part.StockOnHand, req.NewStockOnHand, req.Reason)

	_, err = qtx.AddStockMovement(ctx, store.AddStockMovementParams{
		PartID:    part.ID,
		Qty:       qtyNumeric,
		Type:      "adjustment",
		UnitCost:  pgtype.Int8{Int64: part.CostPrice, Valid: true},
		CreatedBy: pgtype.Int8{Int64: userID, Valid: true},
		Note:      pgtype.Text{String: note, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("record adjustment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, partID)
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
