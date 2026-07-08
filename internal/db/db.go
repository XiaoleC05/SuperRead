package db

import (
	"context"
	"log"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Init() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Pool, err = pgxpool.New(ctx, config.Cfg.DatabaseURL)
	if err != nil {
		return err
	}

	if err := Pool.Ping(ctx); err != nil {
		return err
	}

	log.Println("Database connected successfully")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
