package main

import (
	"context"
	"fmt"
	"os"

	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"

	"github.com/ngc7293/hixi/internal/api"
	"github.com/ngc7293/hixi/internal/sync"
	"github.com/ngc7293/hixi/pkg/spec"
)

func runDatabaseMigrations(conn *pgx.Conn) error {
	migrator, err := migrate.NewMigrator(context.Background(), conn, "schema_migrations")

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

func main() {
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

	conn, err := pgx.Connect(context.Background(), dbUrl)

	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	defer conn.Close(context.Background())

	err = runDatabaseMigrations(conn)

	if err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	discovery, err := sync.FetchDocument[spec.GBFSDiscoveryData](os.Args[1])

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

	c := make(chan error, 1)

	go func() { c <- sync.SyncStationStatus(dbUrl, stationStatusUrl) }()
	go func() { c <- sync.SyncStationInformation(dbUrl, stationInformationUrl) }()
	go func() { c <- api.Serve(dbUrl) }()

	if err := <-c; err != nil {
		panic(err)
	}
}
