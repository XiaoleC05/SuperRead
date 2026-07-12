package db

import (
	"context"
	"fmt"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/jackc/pgx/v5"
)

func GetSettings(ctx context.Context, userID int64) (*model.UserSettings, error) {
	query := `
		SELECT user_id, api_key, api_base, model, fetch_interval_min, email, updated_at
		FROM superread.user_settings
		WHERE user_id = $1
	`
	var s model.UserSettings
	err := Pool.QueryRow(ctx, query, userID).Scan(
		&s.UserID, &s.APIKey, &s.APIBase, &s.Model,
		&s.FetchIntervalMin, &s.Email, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &s, nil
}

func UpdateSettings(ctx context.Context, userID int64, req model.UpdateSettingsRequest) (*model.UserSettings, error) {
	current, err := GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	if current == nil {
		current = &model.UserSettings{
			UserID:          userID,
			APIKey:          "",
			APIBase:         "",
			Model:           "gpt-4o-mini",
			FetchIntervalMin: 30,
			Email:           "",
		}
	}

	if req.APIKey != nil {
		current.APIKey = *req.APIKey
	}
	if req.APIBase != nil {
		current.APIBase = *req.APIBase
	}
	if req.Model != nil {
		current.Model = *req.Model
	}
	if req.FetchIntervalMin != nil {
		current.FetchIntervalMin = *req.FetchIntervalMin
	}
	if req.Email != nil {
		current.Email = *req.Email
	}

	query := `
		INSERT INTO superread.user_settings (user_id, api_key, api_base, model, fetch_interval_min, email)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			api_key = EXCLUDED.api_key,
			api_base = EXCLUDED.api_base,
			model = EXCLUDED.model,
			fetch_interval_min = EXCLUDED.fetch_interval_min,
			email = EXCLUDED.email,
			updated_at = NOW()
	`
	_, err = Pool.Exec(ctx, query,
		current.UserID, current.APIKey, current.APIBase,
		current.Model, current.FetchIntervalMin, current.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("update settings: %w", err)
	}

	current.UpdatedAt = time.Now()
	return current, nil
}