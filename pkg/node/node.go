package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

var (
	registered bool
	apiVersion = "v1"
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
}

func NewNode(name, ipv4, scheduler string, port int) *Node {
	return &Node{
		Name:      name,
		IPv4:      ipv4,
		Scheduler: scheduler,
		Port:      port,
	}
}

func (no *Node) Register() error {

	var _data Response

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

	logrus.Infoln("requesting resource: ", resource)
	logrus.Infof("sending data %s", string(payload))

	resp, err := http.Post(resource, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &_data)
	if err != nil {
		return err
	}

	if _data.Status != "success" {
		logrus.Warnf("error received from scheduler while registering: %s", _data.Message)
		return fmt.Errorf(_data.Message)
	}

	registered = true
	logrus.Debugln("node registration completed.")
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
