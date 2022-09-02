package scheduler

import (
	"github.com/go-playground/validator"
	"github.com/ish-xyz/dreg/pkg/scheduler/storage"
	"github.com/sirupsen/logrus"
)

// Scheduler main functions
// consider using sync mutex or redis directly

var validate *validator.Validate

type Scheduler struct {
	Algo     string
	MaxProcs int
	Store    storage.Storage
}

func NewScheduler(val *validator.Validate, store storage.Storage, maxProcs int, algo string) *Scheduler {
	validate = val
	return &Scheduler{
		Algo:     algo, //@ish-xyz 21/08/2022 TODO: not fully implemented yet
		MaxProcs: maxProcs,
		Store:    store,
	}
}

// Add connection for specified node
func (sch *Scheduler) addNodeConnection(nodeName string) error {

	node, err := sch.Store.ReadNode(nodeName)
	if err != nil {
		return err
	}

	node.Connections += 1
	return sch.Store.WriteNode(node, false)
}

// Remove connection for specified node
func (sch *Scheduler) removeNodeConnection(nodeName string) error {

	node, err := sch.Store.ReadNode(nodeName)
	if err != nil {
		return err
	}
	node.Connections -= 1
	return sch.Store.WriteNode(node, false)
}

// Called by nodes when they periodically advertise the number of connections
func (sch *Scheduler) setNodeConnections(nodeName string, conns int) error {

	node, err := sch.Store.ReadNode(nodeName)
	if err != nil {
		return err
	}
	node.Connections = conns
	return sch.Store.WriteNode(node, true)
}

// Add node to list of nodes
func (sch *Scheduler) registerNode(node *storage.NodeStat) error {

	err := validate.Struct(node)
	if err != nil {
		return err
	}
	return sch.Store.WriteNode(node, false)
}

// Called by the client when the download of a given layer is completed
func (sch *Scheduler) addNodeForLayer(layer, nodeName string) error {

	return sch.Store.WriteLayer(layer, nodeName, "add")
}

// Used by garbage collector when removing layers
func (sch *Scheduler) removeNodeForLayer(layer, nodeName string, force bool) error {

	_layer, err := sch.Store.ReadLayer(layer)
	if err != nil {
		return nil
	}
	if force {
		return sch.Store.WriteLayer(layer, nodeName, "delete")
	}
	sch.Store.WriteLayer(layer, nodeName, "remove")
	if _layer[nodeName] <= 0 {
		sch.Store.WriteLayer(layer, nodeName, "delete")
	}
	return nil
}

// Get nodeStat from storage
func (sch *Scheduler) getNode(nodeName string) (*storage.NodeStat, error) {

	node, err := sch.Store.ReadNode(nodeName)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// Look for all the nodes that have a specific layer,
// then look for the node that has the least connection
// if node not found, return nil
// TODO: add more advanced scheduling
func (sch *Scheduler) schedule(layer string) (*storage.NodeStat, error) {

	candidate := &storage.NodeStat{
		Connections: sch.MaxProcs + 1,
		Name:        "DUMMY_CANDIDATE",
		IPv4:        "127.0.0.1",
	}
	nodes, err := sch.Store.ReadLayer(layer)
	if err != nil {
		return candidate, nil
	}

	for nodeName := range nodes {
		node, err := sch.Store.ReadNode(nodeName)
		if err != nil {
			logrus.Warn("scheduling: node is not registered, skipping")
			continue
		}
		if node.Connections < sch.MaxProcs && node.Connections < candidate.Connections {
			candidate = node
		}
		if candidate.Connections == 0 {
			return candidate, nil
		}
	}

	// Cleanup candidate, this it's a bit ugly.. needs adjustments
	if candidate.Name == "DUMMY_CANDIDATE" {
		candidate = &storage.NodeStat{}
	}

	return candidate, nil
}
