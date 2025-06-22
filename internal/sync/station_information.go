package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngc7293/hixi/pkg/gbfs/v1_0"
)

func FetchStationInformationOnce(pool *pgxpool.Pool, url string) (int64, error) {
	stationInformation, err := FetchDocument[v1_0.StationInformationData](url)

	if err != nil {
		return 0, err
	}

	tx, err := pool.Begin(context.Background())

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
				"location",
                "capacity"
			) VALUES (
				$1,
				$2,
			 	$3,
				$4
			) ON CONFLICT ("external_id") DO UPDATE SET
				"external_id" = excluded."external_id",
				"name" =  excluded."name",
				"location" =  excluded."location",
				"capacity" =  excluded."capacity"
			`,
			station.StationID,
			station.Name,
			fmt.Sprintf("POINT(%f %f)", station.Lon, station.Lat),
			station.Capacity,
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

func FetchStationInformationLoop(pool *pgxpool.Pool, feedUrl string) error {
	for {
		ttl, err := FetchStationInformationOnce(pool, feedUrl)

		if err != nil {
			return fmt.Errorf("failed to sync station information: %w", err)
		}

		time.Sleep(time.Duration(ttl) * time.Second)
	}
}
