package db

import (
	//"backend/src/models"
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB() {
	dbURL := os.Getenv("DBURL")

	// Create the connection pool
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("Unable to connect to database", "err", err)
		os.Exit(1)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		slog.Error("Database unreachable", "err", err)
		os.Exit(1)
	}

	DB = pool
	slog.Info("Successfully connected to Postgres")

	query := `
	CREATE TABLE IF NOT EXISTS users (
		id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email       VARCHAR(255) NOT NULL UNIQUE,
		username    VARCHAR(100) NOT NULL,
		pass_hash	TEXT,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP 
	);

	CREATE TABLE IF NOT EXISTS streams (
		id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		topic       VARCHAR(255),
		feedback    TEXT,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		stream_id   UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
		topic TEXT NOT NULL,
		path        TEXT NOT NULL,          -- e.g. '0001.0005.0014.0051'
		is_leaf     BOOLEAN NOT NULL DEFAULT TRUE,
		generated   BOOLEAN NOT NULL DEFAULT FALSE,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_nodes_stream_id ON nodes (stream_id);

	CREATE INDEX IF NOT EXISTS idx_nodes_path ON nodes (path text_pattern_ops);

	CREATE TABLE IF NOT EXISTS articles (
		node_id     UUID PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
		content     TEXT,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS article_metadata (
		node_id           UUID PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
		status            VARCHAR(32),
		comment           TEXT,
		score             FLOAT,
		seconds_watched   INTEGER,
		created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = DB.Exec(context.Background(), query)
	if err != nil {
		slog.Error("Failed to create tables", "err", err)
		return
	}

	// Migration for DBs created before the generated column existed.
	migration := `
	ALTER TABLE nodes ADD COLUMN IF NOT EXISTS generated BOOLEAN NOT NULL DEFAULT FALSE;
	-- Backfill: nodes that already have an article are marked generated.
	UPDATE nodes SET generated = true WHERE id IN (SELECT node_id FROM articles);
	`
	if _, err := DB.Exec(context.Background(), migration); err != nil {
		slog.Error("Failed to run migration", "err", err)
		return
	}
	slog.Info("Database schema initialized successfully")
}
