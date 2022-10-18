package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ish-xyz/dcache/pkg/node"
	"github.com/ish-xyz/dcache/pkg/node/notifier"
	"github.com/sirupsen/logrus"
)

// files operation by code/int
const (
	Create = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

var (
	Registered bool
	apiVersion = "v1"
)

type Response struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Node    *node.NodeSchema `json:"node,omitempty"`
}

type Client struct {
	Name             string             `validate:"required,alphanum"`
	Notifier         notifier.INotifier `validate:"required"`
	SchedulerAddress string             `validate:"required,url"`
	HTTPClient       *http.Client       `validate:"required"`
	Logger           *logrus.Entry      `validate:"required"`
}

type IClient interface {
	CreateNode(ipv4, scheme string, port, maxconn int) error
	GetNode(name string) (*node.NodeSchema, error)

	AddConnection() error
	RemoveConnection() error

	CreateItem(item string) error
	DeleteItem(item string) error

	GetPeers(item string) (*node.NodeSchema, error)

	// TODO: remove from interface, it's quite useless
	GetHttpClient() *http.Client
}

func NewClient(
	name string,
	nt notifier.INotifier,
	scheduler string,
	lg *logrus.Entry,
) *Client {

	return &Client{
		Name:             name,
		Notifier:         nt,
		SchedulerAddress: scheduler,
		HTTPClient:       &http.Client{},
		Logger:           lg,
	}
}

func (c *Client) Request(method string, resource string, headers map[string]string, body []byte) (*http.Response, error) {

	//TODO: bake in apiversion and scheduler address here

	req, _ := http.NewRequest(method, resource, bytes.NewBuffer(body))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.HTTPClient.Do(req)
}

func (c *Client) CreateNode(ipv4, scheme string, port, maxconn int) error {

	var resp Response

	method := "POST"
	resource := "nodes"
	headers := map[string]string{"Content-Type": "application/json"}

	url := fmt.Sprintf("%s/%s/%s", c.SchedulerAddress, apiVersion, resource)
	node := &node.NodeSchema{
		Name:           c.Name,
		IPv4:           ipv4,
		Port:           port,
		Scheme:         scheme,
		Connections:    0,
		MaxConnections: maxconn,
	}
	payload, err := json.Marshal(node)
	if err != nil {
		return err
	}

	c.Logger.Debugln("calling url: ", url)
	c.Logger.Debugf("sending data %s", string(payload))

	rawResp, err := c.Request(method, url, headers, payload)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "success" {
		c.Logger.Debugf("error received from scheduler while registering: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	Registered = true
	c.Logger.Infoln("node registration completed.")
	return nil
}

// add 1 node connection on the scheduler
func (c *Client) AddConnection() error {

	var resp Response

	method := "POST"
	resource := "connections"
	headers := map[string]string{"Content-Type": "application/json"}

	url := fmt.Sprintf("%s/%s/%s/%s", c.SchedulerAddress, apiVersion, resource, c.Name)

	c.Logger.Debugln("adding 1 connection")

	rawResp, err := c.Request(method, url, headers, nil)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		c.Logger.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		c.Logger.Debugln(resp.Message)
		return fmt.Errorf(resp.Message)
	}

	c.Logger.Debugln("connection added successfully")
	return nil
}

// remove 1 node connection on the scheduler
func (c *Client) RemoveConnection() error {

	var resp Response

	method := "DELETE"
	resource := "connections"

	url := fmt.Sprintf("%s/%s/%s/%s", c.SchedulerAddress, apiVersion, resource, c.Name)

	c.Logger.Debugln("removing 1 connection")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := c.Request(method, url, headers, nil)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		c.Logger.Debugln("error decoding payload:", err, string(body))
		return err
	}

	if resp.Status != "success" {
		c.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return fmt.Errorf(resp.Message)
	}

	c.Logger.Debugln("connection removed successfully")
	return nil
}

// returns a map[string]interface{} with the node stat from the scheduler storage
func (c *Client) GetNode(name string) (*node.NodeSchema, error) {

	var resp Response

	method := "GET"
	resource := "nodes"

	if name == "self" {
		name = c.Name
	}

	url := fmt.Sprintf("%s/%s/%s/%s", c.SchedulerAddress, apiVersion, resource, name)

	c.Logger.Debugln("getting node information")

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := c.Request(method, url, headers, nil)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return nil, err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		c.Logger.Warnln("error decoding payload:", err)
		return nil, err
	}

	if resp.Status != "success" {
		c.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return nil, err
	}

	if resp.Node == nil {
		return nil, fmt.Errorf("node not found")
	}

	c.Logger.Debugf("node data retrieved %+v", resp.Node)

	return resp.Node, nil
}

// Infinite loop that waits for events and notifies the scheduler
func (c *Client) NotifyItems() {
	ch := make(chan *notifier.Event, 10)
	c.Notifier.Subscribe(ch)

	for {
		event := <-ch
		if event.Op == Create {
			c.Logger.Debugln("sending CREATE request for item", event.Item)
			c.CreateItem(event.Item)
		}
		if event.Op == Remove {
			c.Logger.Debugln("sending DELETE request for item", event.Item)
			c.DeleteItem(event.Item)
		}
	}
}

func (c *Client) DeleteItem(item string) error {

	var resp Response
	resource := "items"
	method := "DELETE"
	headers := map[string]string{"Content-Type": "application/json"}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", c.SchedulerAddress, apiVersion, resource, item, c.Name)

	c.Logger.Debugf("item created, notifying to scheduler: %s", item)

	rawResp, err := c.Request(method, url, headers, nil)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		c.Logger.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		c.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return err
	}
	return nil
}

func (c *Client) CreateItem(item string) error {

	var resp Response
	resource := "items"
	method := "POST"
	headers := map[string]string{"Content-Type": "application/json"}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", c.SchedulerAddress, apiVersion, resource, item, c.Name)

	c.Logger.Debugf("item created, notifying to scheduler: %s", item)

	rawResp, err := c.Request(method, url, headers, nil)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		c.Logger.Debugln("error decoding payload:", err)
		return err
	}

	if resp.Status != "success" {
		c.Logger.Debugf("error received from scheduler: %s", resp.Message)
		return err
	}
	return nil
}

// Ask the scheduler to find a node to download the item
func (c *Client) GetPeers(item string) (*node.NodeSchema, error) {

	var resp Response
	c.Logger.Debugf("scheduling dowload for item %s", item)

	method := "GET"
	resource := "peers"

	url := fmt.Sprintf("%s/%s/%s/%s", c.SchedulerAddress, apiVersion, resource, item)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rawResp, err := c.Request(method, url, headers, nil)
	if err != nil {
		c.Logger.Debugf("error requesting url: %s", url)
		return nil, err
	}
	defer rawResp.Body.Close()

	body, _ := ioutil.ReadAll(rawResp.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		c.Logger.Debugln("error decoding payload:", err)
		return nil, err
	}

	if rawResp.StatusCode != 200 {
		c.Logger.Debugf("scheduler response is not 200: %s", resp.Message)
		return nil, fmt.Errorf("scheduler response is not 200")
	}

	if resp.Node == nil {
		return nil, fmt.Errorf("node not found")
	}

	c.Logger.Debugf("node data retrieved %+v", resp.Node)

	return resp.Node, nil
}

func (c *Client) GetHttpClient() *http.Client {
	return c.HTTPClient
}
