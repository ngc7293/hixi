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

type APIHandler struct {
	conn *pgx.Conn
}

func (api *APIHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func (api *APIHandler) handleStations(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("failed to query stations", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	w.Header().Set("Content-Type", "application/geo+json")
	w.WriteHeader(http.StatusOK)

	features := make([]string, 0)

	for rows.Next() {
		var id int64
		var name string
		var lon, lat float64
		rows.Scan(&id, &name, &lon, &lat)
		features = append(features, fmt.Sprintf(`{"type":"Feature","properties":{"id":%d,"name":"%s"},"geometry":{"type":"Point","coordinates":[%f,%f]}}`, id, name, lon, lat))
	}

	err = rows.Err()

	if err != nil {
		slog.Error("failed to query stations", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`{"type":"FeatureCollection","features":[`))
	w.Write([]byte(strings.Join(features, ",")))
	w.Write([]byte(`]}`))
}

func (api *APIHandler) handleStationAvailability(w http.ResponseWriter, r *http.Request) {
	rows, err := api.conn.Query(r.Context(), `
		SELECT
			"time_bucket",
			"bikes_available",
			"ebikes_available"
		FROM "public"."historical_station_availability"
		WHERE
			"station_id" = $1
			AND "time_bucket" BETWEEN NOW() - '1 DAY'::INTERVAL AND NOW()
		ORDER BY "time_bucket" ASC
	`, r.PathValue("stationId"))

	if err != nil {
		slog.Error("failed to query station availability", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	data := make([]string, 0)

	for rows.Next() {
		var time time.Time
		var bikesAvailable, ebikesAvailable float64
		rows.Scan(&time, &bikesAvailable, &ebikesAvailable)
		data = append(data, fmt.Sprintf(`[%d,%.2f,%.2f]`, time.Unix(), bikesAvailable, ebikesAvailable))
	}

	err = rows.Err()

	if err != nil {
		slog.Error("failed to query stations", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`[`))
	w.Write([]byte(strings.Join(data, ",")))
	w.Write([]byte(`]`))
}

func Serve(dbUrl string) error {
	conn, err := pgx.Connect(context.Background(), dbUrl)

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer conn.Close(context.Background())

	mux := http.NewServeMux()
	api := &APIHandler{conn: conn}

	mux.HandleFunc("/", api.handleIndex)
	mux.HandleFunc("/stations", api.handleStations)
	mux.HandleFunc("/stations/{stationId}", api.handleStationAvailability)

	return http.ListenAndServe(":8080", mux)
}
