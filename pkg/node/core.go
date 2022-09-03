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
	RequestIDKey ContextKey
	Name         string
	IPv4         string
	Scheduler    string
	Port         int
	Client       *http.Client
}

func NewNode(key ContextKey, name, ipv4, scheduler string, port int) *Node {
	return &Node{
		RequestIDKey: key,
		Name:         name,
		IPv4:         ipv4,
		Scheduler:    scheduler,
		Port:         port,
		Client:       &http.Client{},
	}
}

func (no *Node) Request(method string, resource string, headers map[string]string, body []byte) (*http.Response, error) {

	req, _ := http.NewRequest(method, resource, bytes.NewBuffer(body))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return no.Client.Do(req)
}

func (no *Node) Register(ctx context.Context) error {

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

	fmt.Println(ctx.Value(no.RequestIDKey).(string))

	headers := map[string]string{
		"Content-Type":          "application/json",
		string(no.RequestIDKey): ctx.Value(no.RequestIDKey).(string),
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

	Registered = true
	logrus.Infoln("node registration completed.")
	return nil
}

// add 1 node connection on the scheduler
func (no *Node) AddConnection(ctx context.Context) error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.Scheduler, apiVersion, "addNodeConnection", no.Name)

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

	resource := fmt.Sprintf("%s/%s/%s/%s", no.Scheduler, apiVersion, "removeNodeConnection", no.Name)

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
func (no *Node) GetStat(ctx context.Context) (map[string]interface{}, error) {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.Scheduler, apiVersion, "getNode", no.Name)

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
	return resp.Data["node"].(map[string]interface{}), nil
}

// Ask the scheduler to find a node to download the layer
func (no *Node) NotifyLayer(ctx context.Context, layer, ops string) error {

	var resp Response
	var resource string
	var method string

	if ops == "add" {
		resource = fmt.Sprintf("%s/%s/%s/%s", no.Scheduler, apiVersion, "addNodeConnection", no.Name)
		method = "PUT"
	} else if ops == "remove" {
		resource = fmt.Sprintf("%s/%s/%s/%s", no.Scheduler, apiVersion, "removeNodeConnection", no.Name)
		method = "DELETE"
	} else {
		return fmt.Errorf("notifyLayer: unknown operation")
	}

	logrus.Infof("notifying removal of layer %s", layer)

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

	logrus.Infof("succcess: %s connection for layer %s", ops, layer)
	return nil
}

// find node to download from
func (no *Node) FindSource(ctx context.Context, layer string) (map[string]string, error) {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.Scheduler, apiVersion, "schedule", layer)

	logrus.Infof("scheduling dowload for layer %s", layer)

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
		return nil, fmt.Errorf("no node found for layer %s", layer)
	}

	nodeMap := resp.Data["node"].(map[string]string)

	logrus.Infof("succcessfully found node %s", nodeMap["name"])
	return nodeMap, nil
}

/*
Proxy:

- proxy pass to the upstream, should filter our every request that meets a certain regex
- node client/core should have:
	methods to
		* deregister()
		* downloadFromNode() // check that node is up and if not fallback to the upstream
		* garbageCollector() // spin up in separate go-routine
- fileserver
	trigger on incoming connections -> addConnection() & removeConnection()
	synchronizer -> routine that every X seconds synchronises the amount of connections \
		on the fileserver and the one advertised to the scheduler

*/