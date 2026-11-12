package catalog

import (
	"context"
	"fmt"
	"log/slog"

	"garage-pos-be/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) SeedDefaults(ctx context.Context) error {
	cats, err := s.queries.ListServiceCategories(ctx)
	if err != nil {
		return fmt.Errorf("check categories: %w", err)
	}
	if len(cats) > 0 {
		return nil
	}

	slog.Info("seeding default service catalog from design guideline...")

	// 1. Categories
	categories := []struct {
		Name      string
		SortOrder int32
		Services  []struct {
			Name        string
			Description string
			Price       int64 // LKR minor units (e.g. 1850000 cents for LKR 18,500.00)
			EstMinutes  int32
		}
	}{
		{
			Name:      "Servicing",
			SortOrder: 1,
			Services: []struct {
				Name        string
				Description string
				Price       int64
				EstMinutes  int32
			}{
				{Name: "Full Service", Description: "About 2 h · oil, filters, 40-point check", Price: 1850000, EstMinutes: 120},
				{Name: "Oil and Filter Change", Description: "About 45 min", Price: 1280000, EstMinutes: 45},
			},
		},
		{
			Name:      "Brakes",
			SortOrder: 2,
			Services: []struct {
				Name        string
				Description string
				Price       int64
				EstMinutes  int32
			}{
				{Name: "Brake Pad Replacement", Description: "Labour · about 1 h", Price: 450000, EstMinutes: 60},
			},
		},
		{
			Name:      "Tyres and alignment",
			SortOrder: 3,
			Services: []struct {
				Name        string
				Description string
				Price       int64
				EstMinutes  int32
			}{
				{Name: "Wheel Alignment", Description: "About 40 min", Price: 350000, EstMinutes: 40},
			},
		},
		{
			Name:      "AC and cleaning",
			SortOrder: 4,
			Services: []struct {
				Name        string
				Description string
				Price       int64
				EstMinutes  int32
			}{
				{Name: "AC Service", Description: "About 1 h 30 min", Price: 950000, EstMinutes: 90},
				{Name: "Body Wash and Vacuum", Description: "About 30 min", Price: 250000, EstMinutes: 30},
			},
		},
	}

	for _, c := range categories {
		catRow, err := s.queries.CreateServiceCategory(ctx, store.CreateServiceCategoryParams{
			Name:      c.Name,
			SortOrder: c.SortOrder,
		})
		if err != nil {
			return fmt.Errorf("create seed category %s: %w", c.Name, err)
		}

		for _, svc := range c.Services {
			_, err := s.queries.CreateService(ctx, store.CreateServiceParams{
				CategoryID:  catRow.ID,
				Name:        svc.Name,
				Description: pgtype.Text{String: svc.Description, Valid: true},
				Price:       svc.Price,
				EstMinutes:  pgtype.Int4{Int32: svc.EstMinutes, Valid: true},
				IsActive:    true,
			})
			if err != nil {
				return fmt.Errorf("create seed service %s: %w", svc.Name, err)
			}
		}
	}

	slog.Info("default service catalog seeded successfully")
	return nil
}
