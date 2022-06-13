package api

type Transport string

const (
	TransportHTTP Transport = "http"
	TransportGRPC Transport = "grpc"
	TransportSQL  Transport = "sql"
)

type Metrics struct {
	Source       string `json:"source"`
	SourceRegion string `json:"source_region"`

	Transport Transport `json:"transport"`

	Target       string `json:"target"`
	TargetRegion string `json:"target_region"`

	Timestamp    int64 `json:"timestamp"`
	CountGood    int64 `json:"count_good"`
	CountWarning int64 `json:"count_warning"`
	CountBad     int64 `json:"count_bad"`
}

type SubmitMetrics struct {
	Metrics []Metrics `json:"metrics"`
}

type Traffic struct {
	From         string `json:"from"`
	To           string `json:"to"`
	CountGood    int64  `json:"count_good"`
	CountWarning int64  `json:"count_warning"`
	CountBad     int64  `json:"count_bad"`
}

type GetTraffic struct {
	Traffic []Traffic `json:"traffic"`
}
