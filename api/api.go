package api

import "time"

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

	Timestamp int64         `json:"timestamp"`
	Duration  time.Duration `json:"duration"`

	CountGood    int64 `json:"count_good"`
	CountWarning int64 `json:"count_warning"`
	CountBad     int64 `json:"count_bad"`
}

type SubmitMetrics struct {
	Metrics  []Metrics  `json:"metrics"`
	NodeInfo []NodeInfo `json:"node_info"`
}

type NodeInfo struct {
	// Name should be unique in a region
	Name string `json:"name"`
	// DisplayName is what will be displayed on the front end
	DisplayName string `json:"display_name"`
	// Type controls what kind of node this is
	Type NodeType `json:"type"`
}

type NodeType string

const (
	NodeDatabase NodeType = "database"
	NodeInternet NodeType = "internet"
	NodeService  NodeType = "service"
)

type Traffic struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Ts           int64  `json:"ts"`
	Duration     int    `json:"duration"`
	CountGood    int64  `json:"count_good"`
	CountWarning int64  `json:"count_warning"`
	CountBad     int64  `json:"count_bad"`
}

type GetTrafficResponse struct {
	Traffic []Traffic `json:"traffic"`
}

type GetNodesResponse struct {
	NodeInfo []NodeInfo `json:"node_info"`
}
