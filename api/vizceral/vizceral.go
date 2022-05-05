package vizceral

type NodeClass string

const (
	ClassNormal  NodeClass = "normal"
	ClassWarning NodeClass = "warning"
	ClassDanger  NodeClass = "danger"
)

type NodeRenderer string

const (
	RendererGlobal       NodeRenderer = "global"
	RendererRegion       NodeRenderer = "region"
	RendererFocusedChild NodeRenderer = "focusedChild"
)

type Notice struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Severity int    `json:"severity"`
}

type Metrics struct {
	Normal  int64 `json:"normal"`
	Danger  int64 `json:"danger"`
	Warning int64 `json:"warning"`
}

type Connection struct {
	Source string `json:"source"`
	Target string `json:"target"`

	Metrics Metrics `json:"metrics"`

	Notices  []Notice          `json:"notices,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Node struct {
	Class       NodeClass    `json:"class,omitempty"`
	Layout      string       `json:"layout,omitempty"`
	Renderer    NodeRenderer `json:"renderer"`
	Name        string       `json:"name"`
	DisplayName string       `json:"displayName,omitempty"`

	MaxVolume float64 `json:"maxVolume,omitempty"`
	EntryNode string  `json:"entryNode,omitempty"`

	Nodes       []Node       `json:"nodes"`
	Connections []Connection `json:"connections"`

	Notices          []Notice          `json:"notices,omitempty"`
	Updated          int64             `json:"updated"`
	ServerUpdateTime int64             `json:"serverUpdateTime"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}
