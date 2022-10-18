package scheduler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ish-xyz/dcache/pkg/node"
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
	Status  string           `json:"status"`
	Message string           `json:"message,omitempty"`
	Node    *node.NodeSchema `json:"node,omitempty"`
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
	r.NotFoundHandler = http.HandlerFunc(notFound)

	// Connections handlers
	r.HandleFunc("/v1/connections/{nodeName}", s.addNodeConnection).Methods("POST")
	r.HandleFunc("/v1/connections/{nodeName}", s.removeNodeConnection).Methods("DELETE")
	r.HandleFunc("/v1/connections/{nodeName}/{conns}", s.setNodeConnections).Methods("PUT")

	// Nodes handlers (TODO: finish missing APIs)
	r.HandleFunc("/v1/nodes", s.createNode).Methods("POST")
	//r.HandleFunc("/v1/nodes/{nodeName}", s.deleteNode).Methods("DELETE")
	//r.HandleFunc("/v1/nodes/{nodeName}", s.updateNode).Methods("PUT")
	r.HandleFunc("/v1/nodes/{nodeName}", s.getNode).Methods("GET")

	r.HandleFunc("/v1/items/{item}/{nodeName}", s.removeNodeForItem).Methods("DELETE")
	r.HandleFunc("/v1/items/{item}/{nodeName}", s.addNodeForItem).Methods("POST")

	r.HandleFunc("/v1/peers/{item}", s.getPeers).Methods("GET")

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
		reqID := []string{"no-request-id"}
		if _, ok := r.Header[requestIDKey]; ok {
			reqID = r.Header[requestIDKey]
		}
		logrus.Infof("request: %v - %s %s %s", reqID, r.RemoteAddr, method, uri)
		h.ServeHTTP(w, r) // serve the original request

	}
	return http.HandlerFunc(logFn)
}

func jsonApiResponse(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	logrus.Debugln("json encoding data:", data)
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	resp := &Response{
		Status:  "error",
		Message: "resource not found",
	}
	jsonApiResponse(w, r, 404, resp)
}

func (s *Server) createNode(w http.ResponseWriter, r *http.Request) {

	var resp Response
	var _node node.NodeSchema
	body, _ := ioutil.ReadAll(r.Body)

	err := json.Unmarshal(body, &_node)
	if err != nil {
		logrus.Warnln("createNode:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 400, resp)
		return
	}

	err = s.Scheduler.createNode(&_node)
	if err != nil {
		logrus.Warnln("createNode:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		logrus.Warnf("registration failed for node %s", string(body))
		jsonApiResponse(w, r, 500, resp)
		return
	}

	logrus.Debugf("node registered successfully %+v", _node)

	resp.Status = "success"
	resp.Message = "node registered"

	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]

	node, err := s.Scheduler.getNode(nodeName)
	if err != nil {
		logrus.Warnln("_getNode:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Node = node

	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) addNodeConnection(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]

	err := s.Scheduler.addNodeConnection(nodeName)
	if err != nil {
		logrus.Warnln("_addNodeConnection:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "1 connection added on node"
	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) removeNodeConnection(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]

	err := s.Scheduler.removeNodeConnection(nodeName)
	if err != nil {
		logrus.Warnln("_removeNodeConnection:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "1 connection removed from node"
	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) setNodeConnections(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]

	connsParam := vars["conns"]
	conns, err := strconv.Atoi(connsParam)
	if err != nil {
		logrus.Warnln("_setNodeConnections:", err.Error())
		resp.Status = "error"
		resp.Message = "can't convert connections to integer"
		jsonApiResponse(w, r, 400, resp)
		return
	}

	err = s.Scheduler.setNodeConnections(nodeName, conns)
	if err != nil {
		logrus.Warnln("_setNodeConnections:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "succesfully set number of connections for node"
	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) addNodeForItem(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]
	item := vars["item"]

	err := s.Scheduler.addNodeForItem(item, nodeName)
	if err != nil {
		logrus.Warnln("_addNodeForItem:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "item/node score increased by 1"

	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) removeNodeForItem(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	nodeName := vars["nodeName"]
	item := vars["item"]

	err := s.Scheduler.removeNodeForItem(item, nodeName, false)
	if err != nil {
		logrus.Warnln("_removeNodeForItem:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	resp.Status = "success"
	resp.Message = "item/node score reduced by 1"

	jsonApiResponse(w, r, 200, resp)
}

func (s *Server) getPeers(w http.ResponseWriter, r *http.Request) {

	var resp Response
	vars := mux.Vars(r)
	item := vars["item"]

	node, err := s.Scheduler.getPeers(item)
	if err != nil {
		logrus.Warnln("_schedule:", err.Error())
		resp.Status = "error"
		resp.Message = err.Error()
		jsonApiResponse(w, r, 500, resp)
		return
	}

	// Prepare response
	code := 200
	resp.Status = "success"
	resp.Node = node
	if resp.Node == nil {
		code = 404
	}

	jsonApiResponse(w, r, code, resp)
}
