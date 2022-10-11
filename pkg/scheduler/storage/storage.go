package storage

import (
	"fmt"

	"github.com/ish-xyz/dcache/pkg/node"
)

// Write() -> location,
type Storage interface {
	WriteNode(node *node.NodeInfo, force bool) error
	ReadNode(nodeName string) (*node.NodeInfo, error)
	WriteIndex(hash string, nodeName string, ops int) error
	ReadIndex(hash string) (map[string]int, error)
}

// Initialise storage for scheduler
func NewStorage(storageType string, opts map[string]string) (Storage, error) {
	if storageType == "memory" {
		indexStore := map[string]map[string]int{
			"init": {
				"init": 1,
			},
		}

		return &MemoryStorage{
			Index: indexStore,
			Nodes: map[string]*node.NodeInfo{},
		}, nil
	}

	return nil, fmt.Errorf("invalid backend type")
}
