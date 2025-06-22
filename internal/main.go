package internal

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"

	"github.com/ngc7293/hixi/internal/server"
	"github.com/ngc7293/hixi/internal/sync"
	"github.com/ngc7293/hixi/pkg/gbfs"
)

func runDatabaseMigrations(pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(context.Background())

	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}

	defer conn.Release()

	migrator, err := migrate.NewMigrator(context.Background(), conn.Conn(), "schema_migrations")

	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	err = migrator.LoadMigrations(os.DirFS("schema"))

	if err != nil {
		return fmt.Errorf("failed to load migrations %w", err)
	}

	err = migrator.Migrate(context.Background())

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func Run() {
	options := slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &options))
	slog.SetDefault(logger)

	dbUrl := os.Getenv("DATABASE_URL")
	preferLanguage := os.Getenv("PREFER_LANGUAGE")

	if preferLanguage == "" {
		slog.Info("defaulting to preferring english (en) feeds")
		preferLanguage = "en"
	}

	if dbUrl == "" {
		slog.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	if len(os.Args) != 2 {
		slog.Error("usage: hixi <gbfs-discovery-url>")
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dbUrl)

	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	defer pool.Close()

	err = runDatabaseMigrations(pool)

	if err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	c := make(chan error, 1)

	apiOnly, apiErr := strconv.ParseBool(os.Getenv("API_ONLY"))
	syncOnly, syncErr := strconv.ParseBool(os.Getenv("SYNC_ONLY"))

	if apiErr != nil {
		apiOnly = false
	}

	if syncErr != nil {
		syncOnly = false
	}

	if syncOnly && apiOnly {
		slog.Error("cannot run both sync and api only")
		os.Exit(1)
	}

	if apiOnly == false {
		discovery, err := sync.FetchDocument[gbfs.GBFSDiscoveryData](os.Args[1])

		if err != nil {
			slog.Error("failed to fetch gbfs discovery", "error", err)
			os.Exit(1)
		}

		stationStatusUrl, lang, err := sync.FindFeedURLWithLanguage(discovery.Data, "station_status", preferLanguage)

		if err != nil {
			slog.Error("failed to find station_status", "error", err)
			os.Exit(1)
		}

		slog.Info("found station_status", "url", stationStatusUrl, "language", lang)

		stationInformationUrl, lang, err := sync.FindFeedURLWithLanguage(discovery.Data, "station_information", preferLanguage)

		if err != nil {
			slog.Error("failed to find station_information", "error", err)
			os.Exit(1)
		}

		slog.Info("found station_information", "url", stationStatusUrl, "language", lang)

		go func() { c <- sync.FetchStationStatusLoop(pool, stationStatusUrl) }()
		go func() { c <- sync.FetchStationInformationLoop(pool, stationInformationUrl) }()
	}

	if syncOnly == false {
		go func() { c <- server.Serve(pool) }()
	}

	if err := <-c; err != nil {
		panic(err)
	}
}
