package storage

//
type NodeSchema struct {
	Name        string `json:"name" validate:"required,alphanum"`
	IPv4        string `json:"ip" validate:"required,ip"`
	Connections int    `json:"connections" validate:"required"`
	Port        int    `json:"port" validate:"required"`
}

// Write() -> location,
type Storage interface {
	WriteNode(node *NodeSchema, force bool) error
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
