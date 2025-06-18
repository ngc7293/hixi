package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ngc7293/hixi/pkg/spec"
	"github.com/ngc7293/hixi/pkg/spec/v1_0"
)

func getFeedUrl(data *spec.GBFSDiscoveryLanguage, feedName string) (string, bool) {
	if data == nil {
		return "", false
	}

	for _, feed := range data.Feeds {
		if feed.Name == feedName {
			return feed.URL, true
		}
	}

	return "", false
}

func FindFeedURLWithLanguage(discovery spec.GBFSDiscoveryData, feedName string, preferLanguage string) (string, string, error) {
	if data, ok := discovery[preferLanguage]; ok {
		if url, found := getFeedUrl(&data, feedName); found {
			return url, preferLanguage, nil
		}
	}

	for lang, data := range discovery {
		if url, found := getFeedUrl(&data, feedName); found {
			return url, lang, nil
		}
	}

	return "", "", fmt.Errorf("no station status URL found in GBFS discovery document")
}

func fetchStationLastReportedOrInsert(tx pgx.Tx, stationID string) (*time.Time, error) {
	var lastReported time.Time
	err := tx.QueryRow(
		context.Background(),
		`SELECT "last_status_reported" FROM "public"."station" WHERE "external_id" = $1`,
		stationID,
	).Scan(&lastReported)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if errors.Is(err, pgx.ErrNoRows) {
		lastReported = time.Now()

		_, err = tx.Exec(
			context.Background(),
			`INSERT INTO "public"."station" ("external_id", "last_status_reported") VALUES ($1, $2)`,
			stationID,
			lastReported,
		)

		if err != nil {
			return nil, err
		}
	}

	return &lastReported, nil
}

func SyncStationStatusOnce(conn *pgx.Conn, url string) (int64, error) {
	stationStatus, err := FetchDocument[v1_0.StationStatusData](url)

	if err != nil {
		return 0, err
	}

	tx, err := conn.Begin(context.Background())

	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(context.Background())

	for _, station := range stationStatus.Data.Stations {
		lastReported, err := fetchStationLastReportedOrInsert(tx, station.StationID)

		if err != nil {
			return 0, fmt.Errorf("failed to getsert station: %w", err)
		}

		if lastReported != nil && !time.Unix(station.LastReported, 0).After(*lastReported) {
			continue
		}

		_, err = tx.Exec(
			context.Background(),
			`INSERT INTO "public"."live_station_availability" (
					"time",
					"station_id",
					"bikes_available",
					"bikes_disabled",
					"ebikes_available",
					"ebikes_disabled",
					"docks_available",
					"docks_disabled"
				) VALUES (
					$1,
					(SELECT "id" FROM "public"."station" WHERE "external_id" = $2),
					$3,
					$4,
					$5,
					$6,
					$7,
					$8
			)`,
			time.Now(),
			station.StationID,
			station.NumBikesAvailable,
			station.NumBikesDisabled,
			station.NumEbikesAvailable,
			station.NumEbikesDisabled,
			station.NumDocksAvailable,
			station.NumDocksDisabled,
		)

		if err != nil {
			return 0, fmt.Errorf("failed to insert station availability: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return stationStatus.TTL, nil
}

func SyncStationStatus(dbUrl string, feedUrl string) error {
	conn, err := pgx.Connect(context.Background(), dbUrl)

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer conn.Close(context.Background())

	for {
		ttl, err := SyncStationStatusOnce(conn, feedUrl)

		if err != nil {
			return fmt.Errorf("failed to sync station status: %w", err)
		}

		time.Sleep(time.Duration(ttl) * time.Second)
	}
}
