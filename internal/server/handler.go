package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	v1 "github.com/ngc7293/hixi/pkg/api/v1"
)

type Handler struct {
	pool   *pgxpool.Pool
	mapUrl string
}

func (api *Handler) ListStation(w http.ResponseWriter, r *http.Request) {
	response := v1.ListStationResponse{
		Type: "FeatureCollection",
	}

	{
		rows, err := api.pool.Query(r.Context(), `
			WITH "station_status" AS (
				SELECT
					"station_id" AS "id",
					MAX("time") > (NOW() - '2 hours'::INTERVAL) AS "active"
				FROM "public"."live_station_availability"
				GROUP BY "station_id"
			)
			SELECT
				"id",
				"name",
				ST_X("location"),
				ST_Y("location"),
				COALESCE("active", false)
			FROM "public"."station"
			LEFT JOIN "station_status" USING ("id")
			WHERE "location" IS NOT NULL`,
		)

		if err != nil {
			slog.Error("failed to query stations", "path", r.URL.Path, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		defer rows.Close()

		for rows.Next() {
			var id int64
			var name string
			var lon, lat float64
			var active bool

			err := rows.Scan(&id, &name, &lon, &lat, &active)

			if err != nil {
				slog.Error("failed to query stations", "path", r.URL.Path, "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			response.Features = append(response.Features, v1.StationFeature{
				Type:       "Feature",
				Properties: v1.StationFeatureProperties{ID: id, Name: name, Active: active},
				Geometry:   v1.GeoJSONPoint{Type: "Point", Coordinates: [2]float64{lon, lat}},
			})
		}

		err = rows.Err()

		if err != nil {
			slog.Error("failed to query stations", "path", r.URL.Path, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	content, err := json.Marshal(response)

	if err != nil {
		slog.Error("failed to marshal response", "path", r.URL.Path, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "max-age=300, public")
	w.Header().Set("Content-Type", "application/geo+json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(content)

	if err != nil {
		slog.Error("failed to write response", "path", r.URL.Path, "error", err)
		return
	}
}

func (api *Handler) GetStation(w http.ResponseWriter, r *http.Request) {
	stationID := r.PathValue("stationId")
	response := v1.GetStationResponse{}

	{
		rows, err := api.pool.Query(r.Context(), `
			SELECT
				TIME_BUCKET('15 minutes'::INTERVAL, "time_bucket") AS "time_bucket",
				"bikes_available",
				"ebikes_available"
			FROM "public"."historical_station_availability"
			WHERE
				"station_id" = $1
				AND "time_bucket" BETWEEN NOW() - '1 DAY'::INTERVAL AND NOW()
			ORDER BY "time_bucket"
			`,
			stationID,
		)

		if err != nil {
			slog.Error("failed to write response", "path", r.URL.Path, "error", err)
			return
		}

		defer rows.Close()

		for rows.Next() {
			var timeBucket time.Time
			var bikesAvailable, ebikesAvailable float64

			err := rows.Scan(&timeBucket, &bikesAvailable, &ebikesAvailable)

			if err != nil {
				slog.Error("failed to query stations", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			response.HistoricalAvailability = append(response.HistoricalAvailability, v1.Availability{
				Time:            timeBucket.Unix(),
				BikesAvailable:  bikesAvailable,
				EbikesAvailable: ebikesAvailable,
			})
		}

		err = rows.Err()

		if err != nil {
			slog.Error("failed to query stations", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	{
		var capacity *int64

		err := api.pool.QueryRow(r.Context(), `
			SELECT
				"capacity"
			FROM "public"."station"
			WHERE "id" = $1
			LIMIT 1`,
			stationID,
		).Scan(&capacity)

		if err != nil {
			slog.Error("failed to query station capacity", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		response.Capacity = capacity
	}

	{
		var time_ time.Time
		var bikesAvailable, ebikesAvailable float64

		err := api.pool.QueryRow(r.Context(), `
			SELECT
				"time",
				"bikes_available",
				"ebikes_available"
			FROM "public"."live_station_availability"
			WHERE "station_id" = $1
			ORDER BY "time" DESC
			LIMIT 1`,
			stationID,
		).Scan(&time_, &bikesAvailable, &ebikesAvailable)

		if err != nil {
			slog.Error("failed to query current station availability", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		response.CurrentAvailability = v1.Availability{
			Time:            time_.Unix(),
			BikesAvailable:  bikesAvailable,
			EbikesAvailable: ebikesAvailable,
		}
	}

	content, err := json.Marshal(response)

	if err != nil {
		slog.Error("failed to marshal response", "path", r.URL.Path, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "max-age=60, public")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(content)

	if err != nil {
		slog.Error("failed to write response", "path", r.URL.Path, "error", err)
		return
	}
}

func (api *Handler) MapProxy(w http.ResponseWriter, r *http.Request) {
	finalUrl := strings.Replace(api.mapUrl, "{z}", r.PathValue("z"), 1)
	finalUrl = strings.Replace(finalUrl, "{x}", r.PathValue("x"), 1)
	finalUrl = strings.Replace(finalUrl, "{y}", r.PathValue("y"), 1)

	response, err := http.Get(finalUrl)

	if err != nil {
		slog.Error("failed to fetch map tile", "path", r.URL.Path, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		slog.Error("failed to fetch map tile", "path", r.URL.Path, "status", response.Status)
		http.Error(w, "map tile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "max-age=43200, public")
	w.WriteHeader(response.StatusCode)

	_, err = io.Copy(w, response.Body)

	if err != nil {
		slog.Error("failed to write map tile response", "path", r.URL.Path, "error", err)
		return
	}
}

func (api *Handler) Health(w http.ResponseWriter, r *http.Request) {
	err := api.pool.Ping(r.Context())

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("OK"))

	if err != nil {
		slog.Error("failed to write health response", "path", r.URL.Path, "error", err)
		return
	}
}

func Serve(pool *pgxpool.Pool) error {
	mapUrl, ok := os.LookupEnv("MAP_URL")

	if !ok {
		return fmt.Errorf("MAP_URL environment variable is not set")
	}

	mux := http.NewServeMux()
	api := &Handler{pool: pool, mapUrl: mapUrl}

	mux.HandleFunc("/stations", api.ListStation)
	mux.HandleFunc("/stations/{stationId}", api.GetStation)
	mux.HandleFunc("/map/{z}/{x}/{y}", api.MapProxy)
	mux.HandleFunc("/health", api.Health)

	fs := http.FileServerFS(os.DirFS("dist/"))
	mux.Handle("/", fs)

	return http.ListenAndServe(":8080", mux)
}
