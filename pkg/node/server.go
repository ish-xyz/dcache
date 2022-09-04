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

func runHeadRequest(client *http.Client, req *http.Request, upstream, proxyPath string) error {

	// Prepare HEAD request
	ctx := req.Context()
	headReq := req.Clone(ctx)
	httpResource := fmt.Sprintf("%s%s", upstream, strings.TrimPrefix(req.RequestURI, proxyPath))
	headReq.Host = strings.Split(upstream, "://")[1]
	headReq.RequestURI = "" // it's illegal to have RequestURI predefined
	headReq.Method = "HEAD"
	u, err := url.Parse(httpResource)
	if err != nil {
		return fmt.Errorf("url parsing failed for %s", httpResource)
	} else {
		headReq.URL = u
	}

	// Perform HEAD request
	headResp, err := client.Do(headReq)
	if err != nil {
		return fmt.Errorf("error with HEAD request %v", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != 200 {
		return fmt.Errorf("HEAD request status code is not 200, is: %d", headResp.StatusCode)
	}

	return nil

}

// ProxyRequestHandler handles the http request using proxy
func (srv *Server) ProxyRequestHandler(proxy, fakeProxy *httputil.ReverseProxy, proxyPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if srv.Regex.MatchString(r.RequestURI) {

			logrus.Debugln("regex matched for ", r.RequestURI)

			err := runHeadRequest(srv.Node.Client, r, srv.Upstream.Address, proxyPath)
			if err != nil {
				logrus.Warn(err)
				goto runProxy
			}

			goto runFakeProxy

			// flag as layer (use context?)
			// get layer using *GetLayer()
			// get node using FindNode(layer)
			// if not found, goto upstream
			// if found change the r.URL to the node URL
			// goto upstream
			// if sha256CheckEnabled => calculate & compare sha256 => if flagged as layer => notifyLayer(add, layerId)
			// fakeProxy.ServerHTTP(w, r)
		}
	runProxy:
		logrus.Info("running proxy")
		proxy.ServeHTTP(w, r)
	runFakeProxy:
		logrus.Info("running fakeProxy")
		proxy.ServeHTTP(w, r)
	}
}

func (srv *Server) Run() error {

	// init proxy
	url, err := url.Parse(srv.Upstream.Address)
	if err != nil {
		return err
	}

	proxyPath := "/proxy"
	fakeProxy := newFakeProxy()
	proxy := newCustomProxy(url, proxyPath)
	fs := http.FileServer(http.Dir(srv.DataDir))

	// handle all requests to your server using the proxy
	logrus.Infof("starting up server on %s", srv.Address)

	http.Handle("/data/", http.StripPrefix("/data/", fs))
	http.HandleFunc(fmt.Sprintf("%s/", proxyPath), srv.ProxyRequestHandler(proxy, fakeProxy, proxyPath))

	log.Fatal(http.ListenAndServe(srv.Address, nil))
	return nil
}
