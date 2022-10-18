package storage

import (
	"fmt"
	"sync"

	"github.com/ish-xyz/dcache/pkg/node"
)

const (
	Add int = iota
	Remove
	Destroy
)

type MemoryStorage struct {
	mu    sync.Mutex
	Index map[string]map[string]int
	Nodes map[string]*node.NodeSchema
}

func (store *MemoryStorage) WriteNode(node *node.NodeSchema, force bool) error {

	store.mu.Lock()
	defer store.mu.Unlock()

	_, ok := store.Nodes[node.Name]
	if ok && !force {
		return fmt.Errorf("node already exists")
	}
	store.Nodes[node.Name] = node
	return nil
}

func (store *MemoryStorage) ReadNode(nodeName string) (*node.NodeSchema, error) {

	store.mu.Lock()
	defer store.mu.Unlock()

	node, ok := store.Nodes[nodeName]
	if ok {
		return node, nil
	}
	return nil, fmt.Errorf("node does not exists")
}

// Write nodes statuses for items
func (store *MemoryStorage) WriteIndex(hash string, nodeName string, ops int) error {

	store.mu.Lock()
	defer store.mu.Unlock()

	switch ops {
	case Add:
		if _, ok := store.Index[hash]; ok {
			store.Index[hash][nodeName] += 1
		} else {
			store.Index[hash] = map[string]int{
				nodeName: 1,
			}
		}
		return nil
	case Remove:
		if _, ok := store.Index[hash]; ok {
			store.Index[hash][nodeName] -= 1
		}
		return nil
	case Destroy:
		delete(store.Index[hash], nodeName)
		return nil
	default:
		return fmt.Errorf("store: invalid operation")
	}

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
