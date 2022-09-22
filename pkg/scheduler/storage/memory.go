package storage

import (
	"fmt"
	"sync"

	"github.com/ish-xyz/dpc/pkg/node"
)

type MemoryStorage struct {
	Index map[string]map[string]int
	Nodes map[string]*node.NodeInfo
}

var lock = sync.RWMutex{}

func (store *MemoryStorage) WriteNode(node *node.NodeInfo, force bool) error {

	_, ok := store.Nodes[node.Name]
	if ok && !force {
		return fmt.Errorf("node already exists")
	}
	store.Nodes[node.Name] = node
	return nil
}

func (store *MemoryStorage) ReadNode(nodeName string) (*node.NodeInfo, error) {
	lock.Lock()
	defer lock.Unlock()

	node, ok := store.Nodes[nodeName]
	if ok {
		return node, nil
	}
	return nil, fmt.Errorf("node does not exists")
}

// Write nodes statuses for items
func (store *MemoryStorage) WriteIndex(hash string, nodeName string, ops string) error {

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
	_item, ok := store.Index[hash]
	if ok {
		return _item, nil
	}
	return nil, fmt.Errorf("item does not exist")
}
