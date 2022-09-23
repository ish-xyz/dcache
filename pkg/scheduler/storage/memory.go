package storage

import (
	"fmt"
	"sync"

	"github.com/ish-xyz/dpc/pkg/node"
)

type MemoryStorage struct {
	mu    sync.Mutex
	Index map[string]map[string]int
	Nodes map[string]*node.NodeInfo
}

func (store *MemoryStorage) WriteNode(node *node.NodeInfo, force bool) error {

	store.mu.Lock()
	defer store.mu.Unlock()

	_, ok := store.Nodes[node.Name]
	if ok && !force {
		return fmt.Errorf("node already exists")
	}
	store.Nodes[node.Name] = node
	return nil
}

func (store *MemoryStorage) ReadNode(nodeName string) (*node.NodeInfo, error) {

	store.mu.Lock()
	defer store.mu.Unlock()

	node, ok := store.Nodes[nodeName]
	if ok {
		return node, nil
	}
	return nil, fmt.Errorf("node does not exists")
}

// Write nodes statuses for items
func (store *MemoryStorage) WriteIndex(hash string, nodeName string, ops string) error {

	store.mu.Lock()
	defer store.mu.Unlock()

	if ops == "delete" {
		delete(store.Index[hash], nodeName)
		return nil
	} else if ops == "add" {
		store.Index[hash][nodeName] += 1
		return nil
	} else if ops == "remove" {
		store.Index[hash][nodeName] -= 1
		return nil
	}
	return fmt.Errorf("store: invalid operation")
}

// Read node statuses for item
func (store *MemoryStorage) ReadIndex(hash string) (map[string]int, error) {

	store.mu.Lock()
	defer store.mu.Unlock()

	_item, ok := store.Index[hash]
	if ok {
		return _item, nil
	}
	return nil, fmt.Errorf("item does not exist")
}
