package resetexamples

// generate:reset
type StorageConfig struct {
	Host     string
	Port     int
	Database string
	Options  map[string]interface{}
	Timeout  *int
	SSL      *bool
}

// generate:reset
type CacheEntry struct {
	Key       string
	Value     []byte
	ExpiresAt int64
	Metadata  map[string]string
	Tags      []string
	HitCount  *int
}

// generate:reset
type ConnectionPool struct {
	MaxConnections int
	CurrentCount   int
	Available      []string
	InUse          map[string]bool
	Config         *StorageConfig
	Stats          map[string]int64
}

// generate:reset
type QueryResult struct {
	Rows     [][]interface{}
	Columns  []string
	RowCount *int
	Error    *string
	Metadata map[string]interface{}
	Duration *float64
	CacheHit *bool
}
