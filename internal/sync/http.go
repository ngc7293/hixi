package sync

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ngc7293/hixi/pkg/gbfs"
)

func FetchDocument[DataType any](url string) (*gbfs.GBFSDocument[DataType], error) {
	slog.Info("fetch document", "method", "get", "url", url)
	resp, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}

	document := gbfs.GBFSDocument[DataType]{}
	err = json.NewDecoder(resp.Body).Decode(&document)

	if err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return &document, nil
}
