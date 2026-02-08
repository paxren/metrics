package resetexamples

// generate:reset
type RequestData struct {
	Method  string
	Path    string
	Headers map[string][]string
	Body    []byte
	Query   map[string]string
	Params  map[string]string
	Size    *int
	Timeout *int64
}

// generate:reset
type ResponseData struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Error      *string
	Duration   *float64
	Size       *int
	Cached     *bool
}

// generate:reset
type MiddlewareData struct {
	RequestID     string
	UserID        *string
	SessionID     *string
	Timestamp     int64
	Metadata      map[string]interface{}
	Permissions   []string
	Roles         []string
	Authenticated *bool
}

// generate:reset
type HandlerMetrics struct {
	RequestCount   int64
	ErrorCount     int64
	ResponseTime   []float64
	StatusCodes    map[int]int64
	EndpointStats  map[string]int64
	ActiveRequests []string
	ErrorMessages  []string
	LastReset      *int64
	ProcessingTime *float64
}
