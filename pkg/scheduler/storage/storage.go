package storage

// Node information necessary for the scheduler
type NodeStat struct {
	Name        string `json:"name" validate:"required,alphanum"`
	IPv4        string `json:"ipv4" validate:"required,ip"`
	Connections int    `json:"connections"`
	Port        int    `json:"port" validate:"required"`
	Scheme      string `json:"scheme" validate:"required"`
}

// Write() -> location,
type Storage interface {
	WriteNode(node *NodeStat, force bool) error
	ReadNode(nodeName string) (*NodeStat, error)
	WriteLayer(layer string, nodeName string, ops string) error
	ReadLayer(layer string) (map[string]int, error)
}

// Initialise storage for scheduler
func NewStorage(storageType string, opts map[string]string) Storage {
	return &MemoryStorage{
		LayersStorage: map[string]map[string]int{},
		Nodes:         map[string]*NodeStat{},
	}
}
