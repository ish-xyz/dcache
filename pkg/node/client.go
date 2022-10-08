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

type Client struct {
	Name             string        `validate:"required,alphanum"`
	SchedulerAddress string        `validate:"required,url"`
	HTTPClient       *http.Client  `validate:"required"`
	Logger           *logrus.Entry `validate:"required"`
}

type NodeInfo struct {
	Name           string `json:"name" validate:"required,alphanum"`
	IPv4           string `json:"ipv4" validate:"required,ip"`
	Connections    int    `json:"connections"`
	MaxConnections int    `json:"maxConnections" validate:"required,number"`
	Port           int    `json:"port" validate:"required"`
	Scheme         string `json:"scheme" validate:"required"`
}

func NewClient(
	name string,
	scheduler string,
	lg *logrus.Entry,
) *Client {

	return &Client{
		Name:             name,
		SchedulerAddress: scheduler,
		HTTPClient:       &http.Client{},
		Logger:           lg,
	}
}

func (no *Client) Request(method string, resource string, headers map[string]string, body []byte) (*http.Response, error) {

	req, _ := http.NewRequest(method, resource, bytes.NewBuffer(body))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return no.HTTPClient.Do(req)
}

func (no *Client) Register(ipv4, scheme string, port, maxconn int) error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s", no.SchedulerAddress, apiVersion, "registerNode")
	nodeInfo := &NodeInfo{
		Name:           no.Name,
		IPv4:           ipv4,
		Port:           port,
		Scheme:         scheme,
		Connections:    0,
		MaxConnections: maxconn,
	}
	payload, err := json.Marshal(nodeInfo)
	if err != nil {
		return err
	}

	no.Logger.Debugln("registering node to: ", resource)
	no.Logger.Debugln("sending data %s", string(payload))

	headers := map[string]string{"Content-Type": "application/json"}

	rawResp, err := no.Request("POST", resource, headers, payload)
	if err != nil {
		no.Logger.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "success" {
		no.Logger.Debugf("error received from scheduler while registering: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	Registered = true
	no.Logger.Infoln("node registration completed.")
	return nil
}

// add 1 node connection on the scheduler
func (no *Client) AddConnection() error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "addNodeConnection", no.Name)

	no.Logger.Debugln("adding 1 connection")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("PUT", resource, headers, nil)
	if err != nil {
		no.Logger.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		no.Logger.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		no.Logger.Debugln(resp.Message)
		return fmt.Errorf(resp.Message)
	}

	no.Logger.Debugln("connection added successfully")
	return nil
}

// remove 1 node connection on the scheduler
func (no *Client) RemoveConnection() error {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "removeNodeConnection", no.Name)

	no.Logger.Debugln("removing 1 connection")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("DELETE", resource, headers, nil)
	if err != nil {
		no.Logger.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		no.Logger.Debugln("error decoding payload:", err, string(body))
		return err
	}

	if resp.Status != "success" {
		no.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	no.Logger.Debugln("connection removed successfully")
	return nil
}

// returns a map[string]interface{} with the node stat from the scheduler storage
func (no *Client) Info() (*NodeInfo, error) {

	var resp Response

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "getNode", no.Name)

	no.Logger.Debugln("getting node information")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("GET", resource, headers, nil)
	if err != nil {
		no.Logger.Debugln("error requesting resource: %s", resource)
		return nil, err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		no.Logger.Warnln("error decoding payload:", err)
		return nil, err
	}

	if resp.Status != "success" {
		no.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return nil, err
	}

	no.Logger.Debugln("retrieved data:", resp.Data["node"].(map[string]interface{}))

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

// Notify scheduler that the current node has an item
func (no *Client) NotifyItem(item string, ops int) error {

	var resp Response
	var resource string
	var method string

	// 1 -> create
	// 4 -> remove
	no.Logger.Debugf("notifying to scheduler: ops -> %d, item -> %s", ops, item)
	if ops == 1 {
		resource = fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "addNodeConnection", no.Name)
		method = "PUT"
	} else if ops == 4 {
		resource = fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "removeNodeConnection", no.Name)
		method = "DELETE"
	} else {
		return fmt.Errorf("NotifyItem: unknown operation")
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request(method, resource, headers, nil)
	if err != nil {
		no.Logger.Debugln("error requesting resource: %s", resource)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		no.Logger.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		no.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return err
	}

	return nil
}

// Ask the scheduler to find a node to download the item
func (no *Client) Schedule(item string) (*NodeInfo, error) {

	var resp Response
	no.Logger.Debugln("scheduling dowload for item %s", item)

	resource := fmt.Sprintf("%s/%s/%s/%s", no.SchedulerAddress, apiVersion, "schedule", item)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := no.Request("GET", resource, headers, nil)
	if err != nil {
		no.Logger.Debugln("error requesting resource: %s", resource)
		return nil, err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		no.Logger.Debugln("error decoding payload:", err)
		return nil, err
	}

	if rawResp.StatusCode != 200 {
		no.Logger.Debugln("scheduler response is not 200: %s", resp.Message)
		return nil, fmt.Errorf("scheduler response is not 200")
	}

	// TODO: Need to ensure that this values are always here
	nodeInfo := &NodeInfo{
		Name:        resp.Data["node"].(map[string]interface{})["name"].(string),
		IPv4:        resp.Data["node"].(map[string]interface{})["ipv4"].(string),
		Port:        resp.Data["node"].(map[string]interface{})["port"].(int),
		Connections: resp.Data["node"].(map[string]interface{})["connections"].(int),
		Scheme:      resp.Data["node"].(map[string]interface{})["scheme"].(string),
	}

	no.Logger.Debugln("succcessfully found node %s", nodeInfo.Name)

	return nodeInfo, nil
}
