package gbfs

type GBFSDocument[DataType any] struct {
	LastUpdated int64    `json:"last_updated"`
	TTL         int64    `json:"ttl"`
	Data        DataType `json:"data"`
}

type GBFSDiscoveryData = map[string]GBFSDiscoveryLanguage

type GBFSDiscoveryLanguage struct {
	Feeds []GBFSFeed `json:"feeds"`
}

type GBFSFeed struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
