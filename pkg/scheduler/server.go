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
	Address   string `json:"address"`
	Scheduler *Scheduler
}

func NewServer(addr string, sch *Scheduler) *Server {
	return &Server{
		Address:   addr,
		Scheduler: sch,
	}
}

func (s *Server) Run() {

	r := mux.NewRouter()
	r.HandleFunc("/v1/addNodeConnection/{nodeName}", s._addNodeConnection).Methods("PUT")
	r.HandleFunc("/v1/removeNodeConnection/{nodeName}", s._removeNodeConnection).Methods("DELETE")
	r.HandleFunc("/v1/setNodeConnections/{nodeName}/{conns}", s._setNodeConnections).Methods("PUT")
	r.HandleFunc("/v1/registerNode", s._registerNode).Methods("POST")

	logrus.Infof("starting up server on %s", s.Address)
	http.Handle("/", r)
	http.ListenAndServe(s.Address, r)
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

	var _node *storage.NodeSchema
	err := json.NewDecoder(r.Body).Decode(&_node)

	if err != nil {
		_apiResponse(w, r, 400, map[string]interface{}{"status": "error", "message": "malformed payload"})
		return
	}

	s.Scheduler.registerNode(_node)
	_apiResponse(w, r, 200, map[string]interface{}{
		"status":  "pending",
		"message": "request received",
	})
}
