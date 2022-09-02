package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	registered   bool
	apiVersion   = "v1"
	requestIDKey = "X-Request-Id"
)

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type Node struct {
	Name      string
	IPv4      string
	Scheduler string
	Port      int
	Client    *http.Client
}

func NewNode(name, ipv4, scheduler string, port int) *Node {
	return &Node{
		Name:      name,
		IPv4:      ipv4,
		Scheduler: scheduler,
		Port:      port,
		Client:    &http.Client{},
	}
}

func generateNewID() string {
	return uuid.New().String()
}

func (no *Node) Request(method string, resource string, headers map[string]string, body []byte) (*http.Response, error) {

	req, _ := http.NewRequest(method, resource, bytes.NewBuffer(body))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return no.Client.Do(req)
}

func (no *Node) Register() error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s", no.Scheduler, apiVersion, "registerNode")
	payload, err := json.Marshal(map[string]interface{}{
		"name":        no.Name,
		"connections": 0,
		"ipv4":        no.IPv4,
		"port":        no.Port,
	})
	if err != nil {
		return err
	}

	logrus.Infoln("registering node: ", resource)
	logrus.Debugln("sending data %s", string(payload))

	headers := map[string]string{
		"Content-Type": "application/json",
		requestIDKey:   generateNewID(),
	}

	rawResp, err := no.Request("POST", resource, headers, payload)
	if err != nil {
		logrus.Warnf("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "success" {
		logrus.Warnf("error received from scheduler while registering: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	registered = true
	logrus.Infoln("node registration completed.")
	return nil
}

func (no *Node) AddNodeConnection() error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s", no.Scheduler, apiVersion, "addNodeConnection")

	logrus.Infoln("adding connection to node")

	headers := map[string]string{
		"Content-Type": "application/json",
		requestIDKey:   generateNewID(),
	}

	rawResp, err := no.Request("PUT", resource, headers, nil)
	if err != nil {
		logrus.Warnf("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "success" {
		logrus.Warnf("error received from scheduler: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	logrus.Infoln("connection added successfully")
	return nil
}

/*
Proxy:

- proxy pass to the upstream, should filter our every request that meets a certain regex
- node client/core should have:
	methods to
		register()
		notifyLayer()
		deregister()
		removeLayer()
		addConnection()
		removeConnection()
		getPeer()
		download()
		garbageCollector() // spin up in separate go-routine
- fileserver
	if fileserver is requested trigger addConnection()

*/
