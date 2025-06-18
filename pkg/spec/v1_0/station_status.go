package v1_0

type StationStatusData struct {
	Stations []StationStatus `json:"stations"`
}

type StationStatus struct {
	StationID         string `json:"station_id"`
	NumBikesAvailable int64  `json:"num_bikes_available"`
	NumBikesDisabled  *int64 `json:"num_bikes_disabled"`
	NumDocksDisabled  *int64 `json:"num_docks_disabled"`
	NumDocksAvailable int64  `json:"num_docks_available"`
	IsInstalled       int64  `json:"is_installed"`
	IsRenting         int64  `json:"is_renting"`
	IsReturning       int64  `json:"is_returning"`
	LastReported      int64  `json:"last_reported"`

	// Non-standard fields (Bixi)
	NumEbikesAvailable *int64 `json:"num_ebikes_available"`
	NumEbikesDisabled  *int64 `json:"num_ebikes_disabled"`
}
