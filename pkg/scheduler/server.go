package scheduler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ish-xyz/dreg/pkg/scheduler/storage"
	"github.com/sirupsen/logrus"
)

var (
	requestIDKey = "X-Request-Id"
)

type Server struct {
	Address   string
	Scheduler *Scheduler
	TLSConfig string
}

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
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
	http.Handle("/", logsMiddleware(r))
	http.ListenAndServe(s.Address, logsMiddleware(r))

	// TODO: add default response for other status codes
	// TODO: add redis storage
	// TODO: add authentication
	// TODO: implement request IDs
}

func logsMiddleware(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {

		uri := r.RequestURI
		method := r.Method
		logrus.Infof("request: %v - %s %s %s", r.Header[requestIDKey], r.RemoteAddr, method, uri)
		h.ServeHTTP(w, r) // serve the original request

	}
	return http.HandlerFunc(logFn)
}

func _apiResponse(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) _removeNodeConnection(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	node := vars["nodeName"]

	err := s.Scheduler.removeNodeConnection(node)
	if err != nil {
		logrus.Warnln("_removeNodeConnection:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "1 connection removed from node"
	_apiResponse(w, r, 200, resp)
}

func (s *Server) _addNodeConnection(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]

	err := s.Scheduler.addNodeConnection(nodeName)
	if err != nil {
		logrus.Warnln("_addNodeConnection:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "1 connection added on node"
	_apiResponse(w, r, 200, resp)
}

func (s *Server) _setNodeConnections(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	node := vars["nodeName"]

	connsParam := vars["conns"]
	conns, err := strconv.Atoi(connsParam)
	if err != nil {
		logrus.Warnln("_setNodeConnections:", err.Error())
		resp.Status = "error"
		resp.Message = "can't convert connections to integer"
		_apiResponse(w, r, 400, resp)
		return
	}

	err = s.Scheduler.setNodeConnections(node, conns)
	if err != nil {
		logrus.Warnln("_setNodeConnections:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "succesfully set number of connections for node"
	_apiResponse(w, r, 200, resp)
}

func (s *Server) _registerNode(w http.ResponseWriter, r *http.Request) {

	var resp Response
	var _node storage.NodeStat
	body, _ := ioutil.ReadAll(r.Body)

	err := json.Unmarshal(body, &_node)
	if err != nil {
		logrus.Warnln("_registerNode:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 400, resp)
		return
	}

	err = s.Scheduler.registerNode(&_node)
	if err != nil {
		logrus.Warnln("_registerNode:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		logrus.Warnf("registration failed for node %s", string(body))
		_apiResponse(w, r, 500, resp)
		return
	}

	logrus.Debugf("node registered successfully %+v", _node)

	resp.Status = "success"
	resp.Message = "node registered"

	_apiResponse(w, r, 200, resp)
}

func (s *Server) _removeNodeForLayer(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	node := vars["nodeName"]
	layer := vars["layer"]

	err := s.Scheduler.removeNodeForLayer(layer, node, false)
	if err != nil {
		logrus.Warnln("_removeNodeForLayer:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "layer/node score reduced by 1"

	_apiResponse(w, r, 200, resp)
}

func (s *Server) _addNodeForLayer(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	node := vars["nodeName"]
	layer := vars["layer"]

	err := s.Scheduler.addNodeForLayer(layer, node)
	if err != nil {
		logrus.Warnln("_addNodeForLayer:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "layer/node score increased by 1"

	_apiResponse(w, r, 200, resp)
}

func (s *Server) _schedule(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	layer := vars["layer"]

	node, err := s.Scheduler.schedule(layer)
	if err != nil {
		logrus.Warnln("_schedule:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		_apiResponse(w, r, 500, resp)
		return
	}

	code := 200
	resp.Status = "success"
	resp.Data = map[string]interface{}{
		"node": node,
	}

	if node.Name == "" {
		code = 404
		resp.Status = "success"
		resp.Data = map[string]interface{}{
			"node": "",
		}
	}
	_apiResponse(w, r, code, resp)
}
