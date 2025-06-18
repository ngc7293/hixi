package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ngc7293/hixi/pkg/spec/v1_0"
)

func SyncStationInformationOnce(conn *pgx.Conn, url string) (int64, error) {
	stationInformation, err := FetchDocument[v1_0.StationInformationData](url)

	if err != nil {
		return 0, err
	}

	tx, err := conn.Begin(context.Background())

	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(context.Background())

	for _, station := range stationInformation.Data.Stations {
		_, err = tx.Exec(
			context.Background(),
			`
			INSERT INTO "public"."station" (
				"external_id",
				"name",
				"location"
			) VALUES (
				$1,
				$2,
			 	$3
			) ON CONFLICT ("external_id") DO UPDATE SET
				"external_id" = excluded."external_id",
				"name" =  excluded."name",
				"location" =  excluded."location"
			`,
			station.StationID,
			station.Name,
			fmt.Sprintf("POINT(%f %f)", station.Lon, station.Lat),
		)

		if err != nil {
			return 0, fmt.Errorf("failed to insert station availability: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return stationInformation.TTL, nil
}

func SyncStationInformation(dbUrl string, feedUrl string) error {
	conn, err := pgx.Connect(context.Background(), dbUrl)

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer conn.Close(context.Background())

	for {
		ttl, err := SyncStationInformationOnce(conn, feedUrl)

		if err != nil {
			return fmt.Errorf("failed to sync station information: %w", err)
		}

		time.Sleep(time.Duration(ttl) * time.Second)
	}
}
