package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"context"

	"github.com/jackc/pgx/v5"
)

type Handler struct {
	conn *pgx.Conn
}

func (api *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func (api *Handler) handleStations(w http.ResponseWriter, r *http.Request) {
	rows, err := api.conn.Query(r.Context(), `
		SELECT
			"id",
			"name",
			ST_X("location"),
			ST_Y("location")
		FROM "public"."station"
		WHERE "location" IS NOT NULL
	`)

	if err != nil {
		slog.Error("failed to query stations", "path", r.URL.Path, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	features := make([]string, 0)

	for rows.Next() {
		var id int64
		var name string
		var lon, lat float64

		err := rows.Scan(&id, &name, &lon, &lat)

		if err != nil {
			slog.Error("failed to query stations", "path", r.URL.Path, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		features = append(features, fmt.Sprintf(`{"type":"Feature","properties":{"id":%d,"name":"%s"},"geometry":{"type":"Point","coordinates":[%f,%f]}}`, id, name, lon, lat))
	}

	err = rows.Err()

	if err != nil {
		slog.Error("failed to query stations", "path", r.URL.Path, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "max-age=300, public")
	w.Header().Set("Content-Type", "application/geo+json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write([]byte(`{"type":"FeatureCollection","features":[` + strings.Join(features, ",") + `]}`))

	if err != nil {
		slog.Error("failed to write stations", "path", r.URL.Path, "error", err)
		return
	}
}

func (api *Handler) handleStationAvailability(w http.ResponseWriter, r *http.Request) {
	rows, err := api.conn.Query(r.Context(), `
		SELECT
			"time_bucket",
			"bikes_available",
			"ebikes_available"
		FROM "public"."historical_station_availability"
		WHERE
			"station_id" = $1
			AND "time_bucket" BETWEEN NOW() - '1 DAY'::INTERVAL AND NOW()
		ORDER BY "time_bucket"
	`, r.PathValue("stationId"))

	if err != nil {
		slog.Error("failed to write response", "path", r.URL.Path, "error", err)
		return
	}

	defer rows.Close()

	data := make([]string, 0)

	for rows.Next() {
		var timeBucket time.Time
		var bikesAvailable, ebikesAvailable float64

		err := rows.Scan(&timeBucket, &bikesAvailable, &ebikesAvailable)

		if err != nil {
			slog.Error("failed to query stations", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data = append(data, fmt.Sprintf(`[%d,%.2f,%.2f]`, timeBucket.Unix(), bikesAvailable, ebikesAvailable))
	}

	err = rows.Err()

	if err != nil {
		slog.Error("failed to query stations", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "max-age=60, public")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write([]byte(`[` + strings.Join(data, ",") + `]`))

	if err != nil {
		slog.Error("failed to write response", "path", r.URL.Path, "error", err)
		return
	}
}

func Serve(dbUrl string) error {
	conn, err := pgx.Connect(context.Background(), dbUrl)

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer conn.Close(context.Background())

	mux := http.NewServeMux()
	api := &Handler{conn: conn}

	mux.HandleFunc("/", api.handleIndex)
	mux.HandleFunc("/stations", api.handleStations)
	mux.HandleFunc("/stations/{stationId}", api.handleStationAvailability)

	return http.ListenAndServe(":8080", mux)
}
