package gpostgresql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"message-service/internal/config"

	"github.com/useinsider/go-pkg/inslogger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExecQueryRower interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewDBConnection(ctx context.Context, dbConfig *config.DatabaseConfig, logger inslogger.Interface) (*pgxpool.Pool, error) {
	var db *pgxpool.Pool

	connString := strings.TrimSpace(fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Name,
		dbConfig.Host,
		fmt.Sprintf("%d", dbConfig.Port),
	))

	parseConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Errorf("Error parsing pool parseConfig: %v", err)
		return nil, err
	}

	parseConfig.MaxConns = 10
	parseConfig.MinConns = 2
	parseConfig.MaxConnLifetime = 30 * time.Minute
	parseConfig.MaxConnIdleTime = 10 * time.Minute
	parseConfig.HealthCheckPeriod = 2 * time.Minute

	db, err = pgxpool.NewWithConfig(ctx, parseConfig)
	if err != nil {
		logger.Errorf("error connecting to database: %v", err)
		return nil, err
	}

	logger.Log("connected to PostgreSQL")
	return db, nil
}

func Close(ctx context.Context, pool *pgxpool.Pool, logger inslogger.Interface) {
	if pool != nil {
		logger.Log("Closing PostgreSQL connection pool")
		pool.Close()
	}
}
