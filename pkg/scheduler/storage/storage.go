package storage

import "github.com/ish-xyz/dreg/pkg/node"

// Write() -> location,
type Storage interface {
	WriteNode(node *node.NodeStat, force bool) error
	ReadNode(nodeName string) (*node.NodeStat, error)
	WriteLayer(layer string, nodeName string, ops string) error
	ReadLayer(layer string) (map[string]int, error)
}

// Initialise storage for scheduler
func NewStorage(storageType string, opts map[string]string) Storage {
	return &MemoryStorage{
		LayersStorage: map[string]map[string]int{},
		Nodes:         map[string]*node.NodeStat{},
	}
}
