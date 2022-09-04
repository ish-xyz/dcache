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
func (srv *Server) ProxyRequestHandler(proxy, fakeProxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if srv.Regex.MatchString(r.RequestURI) {

			logrus.Debugln("regex matched for ", r.RequestURI)

			// Make a copy of the request, to perform a HEAD request
			ctx := r.Context()
			headReq := r.Clone(ctx)
			headReq.Host = strings.Split(srv.Upstream.Address, "://")[1]
			headReq.RequestURI = "" // it's illegal to have RequestURI predefined
			headReq.Method = "HEAD"
			httpResource := fmt.Sprintf("%s%s", srv.Upstream.Address, strings.TrimPrefix(r.RequestURI, "/proxy"))
			u, err := url.Parse(httpResource)
			if err != nil {
				logrus.Debugln("url parsing failed for", httpResource)
				goto upstream
			}
			headReq.URL = u

			logrus.Debugln("performing HEAD request to ", httpResource)
			headResp, err := srv.Node.Client.Do(headReq)
			if err != nil {
				logrus.Debugln("error with HEAD request", err)
				goto upstream
			}
			defer headResp.Body.Close()

			if headResp.StatusCode != 200 {
				logrus.Warnln("HEAD request status code is not 200")
				logrus.Debugln("HEAD request status code is", headResp.StatusCode)
				goto upstream
			}
			logrus.Warnln("caching not implemented yet - skipping")

			// flag as layer (use context?)
			// get layer using *GetLayer()
			// get node using FindNode(layer)
			// if not found, goto upstream
			// if found change the r.URL to the node URL
			// goto upstream
			// if sha256CheckEnabled => calculate & compare sha256 => if flagged as layer => notifyLayer(add, layerId)
			// fakeProxy.ServerHTTP(w, r)
		}
	upstream:
		logrus.Info("request not cached")
		proxy.ServeHTTP(w, r)
	}
}

func (srv *Server) Run() error {

	// init proxy
	url, err := url.Parse(srv.Upstream.Address)
	if err != nil {
		return err
	}

	fakeProxy := newFakeProxy()
	proxy := newCustomProxy(url, "/proxy")
	fs := http.FileServer(http.Dir(srv.DataDir))

	// handle all requests to your server using the proxy
	logrus.Infof("starting up server on %s", srv.Address)

	http.Handle("/data/", http.StripPrefix("/data/", fs))
	http.HandleFunc("/proxy/", srv.ProxyRequestHandler(proxy, fakeProxy))

	log.Fatal(http.ListenAndServe(srv.Address, nil))
	return nil
}
