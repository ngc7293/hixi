package v1

// ListStationResponse
// The API response format for the /stations/ endpoint.  The response is a
// GeoJSON FeatureCollection, where each Feature is a station (Point)
type ListStationResponse struct {
	Type     string           `json:"type"` // always "FeatureCollection"
	Features []StationFeature `json:"features"`
}

type StationFeature struct {
	Type       string                   `json:"type"` // always "Feature"
	Properties StationFeatureProperties `json:"properties"`
	Geometry   GeoJSONPoint             `json:"geometry"`
}

type StationFeatureProperties struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type GeoJSONPoint struct {
	Type        string     `json:"type"` // always "Point"
	Coordinates [2]float64 `json:"coordinates"`
}

type GetStationResponse struct {
	HistoricalAvailability []Availability `json:"historical"`
	CurrentAvailability    Availability   `json:"current"`
	Capacity               *int64         `json:"capacity"`
}

type Availability struct {
	Time            int64   `json:"t"`
	BikesAvailable  float64 `json:"b"` // Availability is averaged within the time bucket, so fractional values are possible
	EbikesAvailable float64 `json:"eb"`
}
