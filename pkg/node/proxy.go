package node

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"

	"github.com/sirupsen/logrus"
)

type Node struct {
	Name string `json:"name"`
}

type Proxy struct {
	Node     *Node          `json:"node"`
	Upstream string         `json:"upstream"`
	Address  string         `json:"address"`
	Regex    *regexp.Regexp `json:"regex"`
	IPv4     string         `json:"ipv4"`
}

// ProxyRequestHandler handles the http request using proxy
func (no *Proxy) ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if no.Regex.MatchString(r.RequestURI) {
			logrus.Infof("TODO: contact scheduler for download\n")
			// TODO: contact scheduler for download
			// Call scheduler and ask to schedule()

			// Download from node if found
			// calculate SHA256, verify SHA256
			// if SHA256 is valid, communicate to scheduler that this node has that layer too
			return
		}
		proxy.ServeHTTP(w, r)
	}
}

func (no *Proxy) Run() error {

	// init proxy
	url, err := url.Parse(no.Upstream)
	if err != nil {
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	// handle all requests to your server using the proxy
	logrus.Infof("starting up server on %s", no.Address)
	http.HandleFunc("/", no.ProxyRequestHandler(proxy))
	log.Fatal(http.ListenAndServe(no.Address, nil))

	return nil
}
