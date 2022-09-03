package node

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type Server struct {
	Node     *Node
	Upstream *UpstreamConfig
	Address  string
	DataDir  string
	Regex    *regexp.Regexp
}

type UpstreamConfig struct {
	Address  string
	Insecure bool
}

// ProxyRequestHandler handles the http request using proxy
func (srv *Server) ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if srv.Regex.MatchString(r.RequestURI) {

			logrus.Debugln("regex matched for ", r.RequestURI)

			// // Make a copy of the request
			headReq := *r
			headReq.Host = strings.Split(srv.Upstream.Address, "://")[1]
			// illegal to have RequestURI set in request object, should always be computed
			headReq.RequestURI = ""
			headReq.Method = "HEAD"

			resource := fmt.Sprintf("%s%s", srv.Upstream.Address, strings.TrimPrefix(r.RequestURI, "/proxy"))

			logrus.Debugln("head request to ", resource)
			u, err := url.Parse(resource)
			if err != nil {
				logrus.Debugln("url parsing failed for", resource)
				goto upstream
			}

			headReq.URL = u
			headResp, err := srv.Node.Client.Do(&headReq)
			if err != nil {
				logrus.Debugln("error with head request", err)
				goto upstream
			}
			// head request should have no body, but it's best practice to close the body
			defer headResp.Body.Close()

			if headResp.StatusCode != 200 {
				logrus.Debugln("head request status code is", headResp.StatusCode)
				goto upstream
			}

			logrus.Warnln("caching not implemented yet - skipping")

			// flag as layer
			// get layer using *GetLayer()
			// get node using FindNode(layer)
			// if not found, goto upstream
			// if found change the r.URL to the node URL
			// goto upstream
			// if sha256CheckEnabled => calculate & compare sha256 => if flagged as layer => notifyLayer(add, layerId)
		}
	upstream:
		fmt.Println("not caching")
		proxy.ServeHTTP(w, r)

	}
}

func (srv *Server) Run() error {

	// init proxy
	url, err := url.Parse(srv.Upstream.Address)
	if err != nil {
		return err
	}

	proxy := newCustomProxy(url, "/proxy")
	fs := http.FileServer(http.Dir(srv.DataDir))

	// handle all requests to your server using the proxy
	logrus.Infof("starting up server on %s", srv.Address)

	http.Handle("/data/", http.StripPrefix("/data/", fs))
	http.HandleFunc("/proxy/", srv.ProxyRequestHandler(proxy))

	log.Fatal(http.ListenAndServe(srv.Address, nil))
	return nil
}
