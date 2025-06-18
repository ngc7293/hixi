package v1_0

type StationInformationData struct {
	Stations []StationInformationStation `json:"stations"`
}

type StationInformationStation struct {
	StationID     string   `json:"station_id"`
	Name          string   `json:"name"`
	ShortName     *string  `json:"short_name"`
	Lat           float64  `json:"lat"`
	Lon           float64  `json:"lon"`
	Address       *string  `json:"address"`
	CrossStreet   *string  `json:"cross_street"`
	RegionID      *string  `json:"region_id"`
	PostCode      *string  `json:"post_code"`
	RentalMethods []string `json:"rental_methods"`
	Capacity      *int64   `json:"capacity"`

	// Non-standard fields (Bixi)
	ExternalID                  *string `json:"external_id"`
	EightdHasKeyDispenser       *bool   `json:"eightd_has_key_dispenser"`
	HasKiosk                    *bool   `json:"has_kiosk"`
	ElectricBikeSurchargeWaiver bool    `json:"electric_bike_surcharge_waiver"`
	IsCharging                  bool    `json:"is_charging"`
}
