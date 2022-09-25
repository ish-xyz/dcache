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
	Registered bool
	apiVersion = "v1"
)

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type Node struct {
	Name             string       `validate:"required,alphanum"`
	IPv4             string       `validate:"required,ipv4"`
	SchedulerAddress string       `validate:"required,url"`
	Port             int          `validate:"required,number"`
	Client           *http.Client `validate:"required"`
	Scheme           string       `validate:"required"`
	MaxConnections   int          `validate:"required,number"`
}

type NodeInfo struct {
	Name           string `json:"name" validate:"required,alphanum"`
	IPv4           string `json:"ipv4" validate:"required,ip"`
	Connections    int    `json:"connections"`
	MaxConnections int    `json:"maxConnections" validate:"required,number"`
	Port           int    `json:"port" validate:"required"`
	Scheme         string `json:"scheme" validate:"required"`
}

func NewNode(name, ipv4, scheme, scheduler string, port, maxConnections int) *Node {
	return &Node{
		Name:             name,
		IPv4:             ipv4,
		SchedulerAddress: scheduler,
		Port:             port,
		Client:           &http.Client{},
		Scheme:           scheme,
		MaxConnections:   maxConnections,
	}
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

	resource := fmt.Sprintf("%s/%s/%s", no.SchedulerAddress, apiVersion, "registerNode")
	nodeInfo := &NodeInfo{
		Name:           no.Name,
		IPv4:           no.IPv4,
		Port:           no.Port,
		Scheme:         no.Scheme,
		Connections:    0,
		MaxConnections: no.MaxConnections,
	}
	payload, err := json.Marshal(nodeInfo)
	if err != nil {
		return err
	}

	logrus.Debugln("registering node to: ", resource)
	logrus.Debugln("sending data %s", string(payload))

	headers := map[string]string{"Content-Type": "application/json"}

	rawResp, err := no.Request("POST", resource, headers, payload)
	if err != nil {
		logrus.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "success" {
		logrus.Debugln("error received from scheduler while registering: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	Registered = true
	logrus.Infoln("node registration completed.")
	return nil
}

// add 1 node connection on the scheduler
func (no *Node) AddConnection() error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "addNodeConnection", no.Name)

	logrus.Debugln("adding 1 connection")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("PUT", resource, headers, nil)
	if err != nil {
		logrus.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		logrus.Debugln(resp.Message)
		return fmt.Errorf(resp.Message)
	}

	logrus.Debugln("connection added successfully")
	return nil
}

// remove 1 node connection on the scheduler
func (no *Node) RemoveConnection() error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "removeNodeConnection", no.Name)

	logrus.Debugln("removing 1 connection")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("DELETE", resource, headers, nil)
	if err != nil {
		logrus.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Debugln("error decoding payload:", err, string(body))
		return err
	}

	if resp.Status != "success" {
		logrus.Debugln("error received from scheduler: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	logrus.Debugln("connection removed successfully")
	return nil
}

// returns a map[string]interface{} with the node stat from the scheduler storage
func (no *Node) Info() (*NodeInfo, error) {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "getNode", no.Name)

	logrus.Debugln("getting node information")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("GET", resource, headers, nil)
	if err != nil {
		logrus.Debugln("error requesting resource: %s", resource)
		return nil, err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Warnln("error decoding payload:", err)
		return nil, err
	}

	if resp.Status != "success" {
		logrus.Debugln("error received from scheduler: %s", resp.Message)
		return nil, err
	}

	logrus.Debugln("retrieved data:", resp.Data["node"].(map[string]interface{}))

	//TODO: find a cleaner way
	nodeInfo := &NodeInfo{
		Name:           resp.Data["node"].(map[string]interface{})["name"].(string),
		IPv4:           resp.Data["node"].(map[string]interface{})["ipv4"].(string),
		Port:           int(resp.Data["node"].(map[string]interface{})["port"].(float64)),
		Connections:    int(resp.Data["node"].(map[string]interface{})["connections"].(float64)),
		MaxConnections: int(resp.Data["node"].(map[string]interface{})["maxConnections"].(float64)),
		Scheme:         resp.Data["node"].(map[string]interface{})["scheme"].(string),
	}

	return nodeInfo, nil
}

//TODO: make the following method a routine "Notifier" that runs in background
// 		and notifies as soon as items are created or downloaded
// Notify scheduler that the current node has an item
func (no *Node) NotifyItem(item, ops string) error {

	var resp Response
	var resource string
	var method string

	if ops == "add" {
		resource = fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "addNodeConnection", no.Name)
		method = "PUT"
	} else if ops == "remove" {
		resource = fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "removeNodeConnection", no.Name)
		method = "DELETE"
	} else {
		return fmt.Errorf("NotifyItem: unknown operation")
	}

	logrus.Infof("notifying removal of item %s", item)

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request(method, resource, headers, nil)
	if err != nil {
		logrus.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		logrus.Debugln("error received from scheduler: %s", resp.Message)
		return err
	}

	logrus.Infof("succcess: %s connection for item %s", ops, item)
	return nil
}

// Ask the scheduler to find a node to download the item
func (no *Node) FindSource(item string) (*NodeInfo, error) {

	var resp Response
	logrus.Debugln("scheduling dowload for item %s", item)

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "schedule", item)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("GET", resource, headers, nil)
	if err != nil {
		logrus.Debugln("error requesting resource: %s", resource)
		return nil, err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Debugln("error decoding payload:", err)
		return nil, err
	}

	if resp.Status != "success" || rawResp.StatusCode != 200 {
		logrus.Debugln("error received from scheduler: %s", resp.Message)
		return nil, err
	}

	nodeInfo := &NodeInfo{
		Name:        resp.Data["node"].(map[string]interface{})["name"].(string),
		IPv4:        resp.Data["node"].(map[string]interface{})["ipv4"].(string),
		Port:        resp.Data["node"].(map[string]interface{})["port"].(int),
		Connections: resp.Data["node"].(map[string]interface{})["connections"].(int),
		Scheme:      resp.Data["node"].(map[string]interface{})["scheme"].(string),
	}

	logrus.Debugln("succcessfully found node %s", nodeInfo.Name)

	return nodeInfo, nil
}
