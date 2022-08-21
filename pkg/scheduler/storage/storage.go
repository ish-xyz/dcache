package storage

//
type NodeSchema struct {
	Name        string `json:"name"`
	IPv4        string `json:"ipv4"`
	Connections int    `json:"connections"`
}

// Write() -> location,
type Storage interface {
	WriteNode(node *NodeSchema) error
	ReadNode(nodeName string) (*NodeSchema, error)
	WriteLayer(layer string, nodeName string, ops string) error
	ReadLayer(layer string) (map[string]int, error)
}

// Initialise storage for scheduler
func NewStorage(storageType string, opts map[string]string) Storage {
	return &MemoryStorage{
		LayersStorage: map[string]map[string]int{},
		Nodes:         map[string]*NodeSchema{},
	}
}
