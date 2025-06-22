package sync

import (
	"testing"

	"github.com/ngc7293/hixi/pkg/gbfs"
)

func TestGetStationStatusURL(t *testing.T) {
	tests := []struct {
		name    string
		data    *gbfs.GBFSDiscoveryLanguage
		wantURL string
		wantOk  bool
	}{
		{
			name: "station_status feed exists",
			data: &gbfs.GBFSDiscoveryLanguage{
				Feeds: []gbfs.GBFSFeed{
					{Name: "system_information", URL: "https://example.com/system_information.json"},
					{Name: "station_status", URL: "https://example.com/station_status.json"},
					{Name: "station_information", URL: "https://example.com/station_information.json"},
				},
			},
			wantURL: "https://example.com/station_status.json",
			wantOk:  true,
		},
		{
			name: "station_status feed does not exist",
			data: &gbfs.GBFSDiscoveryLanguage{
				Feeds: []gbfs.GBFSFeed{
					{Name: "system_information", URL: "https://example.com/system_information.json"},
					{Name: "station_information", URL: "https://example.com/station_information.json"},
				},
			},
			wantURL: "",
			wantOk:  false,
		},
		{
			name: "empty feeds",
			data: &gbfs.GBFSDiscoveryLanguage{
				Feeds: []gbfs.GBFSFeed{},
			},
			wantURL: "",
			wantOk:  false,
		},
		{
			name:    "nil data",
			data:    nil,
			wantURL: "",
			wantOk:  false,
		},
		{
			name: "station_status is the only feed",
			data: &gbfs.GBFSDiscoveryLanguage{
				Feeds: []gbfs.GBFSFeed{
					{Name: "station_status", URL: "https://example.com/only_station_status.json"},
				},
			},
			wantURL: "https://example.com/only_station_status.json",
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, ok := getFeedUrl(tt.data, "station_status")
			if url != tt.wantURL {
				t.Errorf("getStationStatusURL() gotURL = %v, want %v", url, tt.wantURL)
			}
			if ok != tt.wantOk {
				t.Errorf("getStationStatusURL() gotOk = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}
