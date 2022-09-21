package node

import (
	"bytes"
	"context"
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

// needed cause context doesn't accept primitive types as key
type ContextKey string

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type Node struct {
	RequestIDKey     ContextKey   `validate:"required"`
	Name             string       `validate:"required,alphanum"`
	IPv4             string       `validate:"required,ipv4"`
	SchedulerAddress string       `validate:"required,url"`
	Port             int          `validate:"required,number"`
	Client           *http.Client `validate:"required"`
	Scheme           string       `validate:"required"`
}

type NodeStat struct {
	Name        string `json:"name" validate:"required,alphanum"`
	IPv4        string `json:"ipv4" validate:"required,ip"`
	Connections int    `json:"connections"`
	Port        int    `json:"port" validate:"required"`
	Scheme      string `json:"scheme" validate:"required"`
}

func NewNode(key ContextKey, name, ipv4, scheme, scheduler string, port int) *Node {
	return &Node{
		RequestIDKey:     key,
		Name:             name,
		IPv4:             ipv4,
		SchedulerAddress: scheduler,
		Port:             port,
		Client:           &http.Client{},
		Scheme:           scheme,
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
	nodestat := &NodeStat{
		Name:        no.Name,
		IPv4:        no.IPv4,
		Port:        no.Port,
		Scheme:      no.Scheme,
		Connections: 0,
	}
	payload, err := json.Marshal(nodestat)
	if err != nil {
		return err
	}

	logrus.Infoln("registering node: ", resource)
	logrus.Debugln("sending data %s", string(payload))

	headers := map[string]string{"Content-Type": "application/json"}

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

	Registered = true
	logrus.Infoln("node registration completed.")
	return nil
}

// add 1 node connection on the scheduler
func (no *Node) AddConnection(ctx context.Context) error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "addNodeConnection", no.Name)

	logrus.Infoln("adding 1 connection")

	headers := map[string]string{
		"Content-Type":          "application/json",
		string(no.RequestIDKey): ctx.Value(no.RequestIDKey).(string),
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
		logrus.Warnln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		logrus.Warnf("error received from scheduler: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	logrus.Infoln("connection added successfully")
	return nil
}

// remove 1 node connection on the scheduler
func (no *Node) RemoveConnection(ctx context.Context) error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "removeNodeConnection", no.Name)

	logrus.Infoln("removing 1 connection")

	headers := map[string]string{
		"Content-Type":          "application/json",
		string(no.RequestIDKey): ctx.Value(no.RequestIDKey).(string),
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
		logrus.Warnln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		logrus.Warnf("error received from scheduler: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	logrus.Infoln("connection removed successfully")
	return nil
}

// returns a map[string]interface{} with the node stat from the scheduler storage
func (no *Node) Stat(ctx context.Context) (*NodeStat, error) {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "getNode", no.Name)

	logrus.Infoln("getting nodeStat information")

	headers := map[string]string{
		"Content-Type":          "application/json",
		string(no.RequestIDKey): ctx.Value(no.RequestIDKey).(string),
	}

	rawResp, err := no.Request("GET", resource, headers, nil)
	if err != nil {
		logrus.Warnf("error requesting resource: %s", resource)
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
		logrus.Warnf("error received from scheduler: %s", resp.Message)
		return nil, err
	}

	logrus.Infoln("connection added successfully")

	//TODO: find a cleaner way
	nodestat := &NodeStat{
		Name:        resp.Data["node"].(map[string]interface{})["name"].(string),
		IPv4:        resp.Data["node"].(map[string]interface{})["ipv4"].(string),
		Port:        resp.Data["node"].(map[string]interface{})["port"].(int),
		Connections: resp.Data["node"].(map[string]interface{})["connections"].(int),
		Scheme:      resp.Data["node"].(map[string]interface{})["scheme"].(string),
	}

	return nodestat, nil
}

// Ask the scheduler to find a node to download the item
func (no *Node) NotifyItem(ctx context.Context, item, ops string) error {

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
		"Content-Type":          "application/json",
		string(no.RequestIDKey): ctx.Value(no.RequestIDKey).(string),
	}

	rawResp, err := no.Request(method, resource, headers, nil)
	if err != nil {
		logrus.Warnf("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Warnln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		logrus.Warnf("error received from scheduler: %s", resp.Message)
		return err
	}

	logrus.Infof("succcess: %s connection for item %s", ops, item)
	return nil
}

// find node to download from
func (no *Node) FindSource(ctx context.Context, item string) (*NodeStat, error) {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "schedule", item)

	logrus.Infof("scheduling dowload for item %s", item)

	headers := map[string]string{
		"Content-Type":          "application/json",
		string(no.RequestIDKey): ctx.Value(no.RequestIDKey).(string),
	}

	rawResp, err := no.Request("GET", resource, headers, nil)
	if err != nil {
		logrus.Warnf("error requesting resource: %s", resource)
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
		logrus.Warnf("error received from scheduler: %s", resp.Message)
		return nil, err
	}

	if resp.Data["node"] == "" {
		return nil, fmt.Errorf("no node found for item %s", item)
	}

	//TODO: find a cleaner way
	nodestat := &NodeStat{
		Name:        resp.Data["node"].(map[string]interface{})["name"].(string),
		IPv4:        resp.Data["node"].(map[string]interface{})["ipv4"].(string),
		Port:        resp.Data["node"].(map[string]interface{})["port"].(int),
		Connections: resp.Data["node"].(map[string]interface{})["connections"].(int),
		Scheme:      resp.Data["node"].(map[string]interface{})["scheme"].(string),
	}

	logrus.Infof("succcessfully found node %s", nodestat.Name)
	return nodestat, nil
}

func downloadItem(url, destination, item string) {
	fmt.Println("download item in filepath")
	// TODO: NotifyItem too
}

/*
Proxy:

- proxy pass to the upstream, should filter our every request that meets a certain regex
- node client/core should have:
	methods to
		* deregister()
		* syncNodeInfo()
		* garbageCollector() // spin up in separate go-routine
- fileserver
	trigger on incoming connections -> addConnection() & removeConnection()
	synchronizer -> routine that every X seconds synchronises the amount of connections \
		on the fileserver and the one advertised to the scheduler

*/
