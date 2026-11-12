package settings

import (
	"context"

	"garage-pos-be/internal/store"

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

func (s *Service) GetAll(ctx context.Context) (SettingsResponse, error) {
	rows, err := s.queries.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}

	res := make(SettingsResponse)
	for _, r := range rows {
		res[r.Key] = r.Value
	}
	return res, nil
}

func (s *Service) UpdateAll(ctx context.Context, req UpdateSettingsRequest) (SettingsResponse, error) {
	for k, v := range req {
		err := s.queries.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   k,
			Value: v,
		})
		if err != nil {
			return nil, err
		}
	}

	return s.GetAll(ctx)
}
