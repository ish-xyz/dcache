package scheduler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ish-xyz/dreg/pkg/scheduler/storage"
	"github.com/sirupsen/logrus"
)

type Server struct {
	Address   string
	Scheduler *Scheduler
	TLSConfig string
}

func NewServer(addr string, sch *Scheduler) *Server {
	return &Server{
		Address:   addr,
		Scheduler: sch,
		TLSConfig: "",
	}
}

func (s *Server) Run() {

	r := mux.NewRouter()

	r.HandleFunc("/v1/registerNode", s._registerNode).Methods("POST")
	r.HandleFunc("/v1/addNodeConnection/{nodeName}", s._addNodeConnection).Methods("PUT")
	r.HandleFunc("/v1/removeNodeConnection/{nodeName}", s._removeNodeConnection).Methods("DELETE")
	r.HandleFunc("/v1/setNodeConnections/{nodeName}/{conns}", s._setNodeConnections).Methods("PUT")
	r.HandleFunc("/v1/removeNodeForLayer/{layer}/{nodeName}", s._removeNodeForLayer).Methods("DELETE")
	r.HandleFunc("/v1/addNodeForLayer/{layer}/{nodeName}", s._addNodeForLayer).Methods("PUT")
	r.HandleFunc("/v1/schedule/{layer}", s._schedule).Methods("GET")

	logrus.Infof("starting up server on %s", s.Address)
	http.Handle("/", r)
	http.ListenAndServe(s.Address, r)

	// TODO: add default response for other codes
	// TODO: add redis storage
	// TODO: add authentication
}

func _apiResponse(w http.ResponseWriter, r *http.Request, code int, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) _removeNodeConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	node, ok := vars["nodeName"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing node name"})
		return
	}
	err := s.Scheduler.removeNodeConnection(node)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	_apiResponse(w, r, 200, map[string]interface{}{"status": "success", "message": "1 connection removed from node"})
}

func (s *Server) _addNodeConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName, ok := vars["nodeName"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing node name"})
		return
	}
	err := s.Scheduler.addNodeConnection(nodeName)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	_apiResponse(w, r, 200, map[string]interface{}{"status": "success", "message": "1 connection added on node"})
}

func (s *Server) _setNodeConnections(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	node, ok := vars["nodeName"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing node name"})
		return
	}
	conns, ok := vars["conns"]
	_conns, err := strconv.Atoi(conns)
	if !ok || err != nil {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing connections variable"})
		return
	}

	err = s.Scheduler.setNodeConnections(node, _conns)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	_apiResponse(w, r, 200, map[string]interface{}{
		"status":  "success",
		"message": "succesfully set number of connections for node",
	})
}

func (s *Server) _registerNode(w http.ResponseWriter, r *http.Request) {

	var _node *storage.NodeStat
	err := json.NewDecoder(r.Body).Decode(&_node)

	if err != nil {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "malformed payload"})
		return
	}

	err = s.Scheduler.registerNode(_node)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}

	_apiResponse(w, r, 200, map[string]interface{}{
		"status":  "success",
		"message": "node registered",
	})
}

func (s *Server) _removeNodeForLayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	node, ok := vars["nodeName"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing node name"})
		return
	}

	layer, ok := vars["layer"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing layer sha"})
		return
	}

	err := s.Scheduler.removeNodeForLayer(layer, node, false)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}

	_apiResponse(w, r, 200, map[string]interface{}{
		"status":  "success",
		"message": "layer/node score reduced by 1",
	})
}

func (s *Server) _addNodeForLayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	node, ok := vars["nodeName"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing node name"})
		return
	}

	layer, ok := vars["layer"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing layer sha"})
		return
	}

	err := s.Scheduler.addNodeForLayer(layer, node)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}

	_apiResponse(w, r, 200, map[string]interface{}{
		"status":  "success",
		"message": "layer/node score increased by 1",
	})
}

func (s *Server) _schedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	layer, ok := vars["layer"]
	if !ok {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "missing layer sha"})
		return
	}

	node, err := s.Scheduler.schedule(layer)
	if err != nil {
		_apiResponse(w, r, 500, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}

	code := 200
	data := map[string]interface{}{
		"status": "success",
		"node":   node,
	}
	if node.Name == "" {
		code = 404
		data = map[string]interface{}{
			"status": "success",
			"node":   "",
		}
	}
	_apiResponse(w, r, code, data)
}
