package storage

import (
	"fmt"
	"sync"

	"github.com/ish-xyz/dreg/pkg/node"
)

type MemoryStorage struct {
	LayersStorage map[string]map[string]int
	Nodes         map[string]*node.NodeStat
}

var lock = sync.RWMutex{}

func (store *MemoryStorage) WriteNode(node *node.NodeStat, force bool) error {

	_, ok := store.Nodes[node.Name]
	if ok && !force {
		return fmt.Errorf("node already exists")
	}
	store.Nodes[node.Name] = node
	return nil
}

func (store *MemoryStorage) ReadNode(nodeName string) (*node.NodeStat, error) {
	lock.Lock()
	defer lock.Unlock()

	node, ok := store.Nodes[nodeName]
	if ok {
		return node, nil
	}
	return nil, fmt.Errorf("node does not exists")
}

// Write nodes statuses for layers
func (store *MemoryStorage) WriteLayer(layer string, nodeName string, ops string) error {

	if ops == "delete" {
		delete(store.LayersStorage[layer], nodeName)
		return nil
	} else if ops == "add" {
		store.LayersStorage[layer][nodeName] += 1
		return nil
	} else if ops == "remove" {
		store.LayersStorage[layer][nodeName] -= 1
		return nil
	}
	return fmt.Errorf("store: invalid operation")
}

// Read layer
func (store *MemoryStorage) ReadLayer(layer string) (map[string]int, error) {
	_layer, ok := store.LayersStorage[layer]
	if ok {
		return _layer, nil
	}
	return nil, fmt.Errorf("layer does not exist")
}
