package node

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type Server struct {
	Node     *Node           `validate:"required"`
	Upstream *UpstreamConfig `validate:"required,dive"`
	DataDir  string          `validate:"required,dir"`
	Regex    *regexp.Regexp  `validate:"required"`
}

type UpstreamConfig struct {
	Address  string `validate:"required,url"`
	Insecure bool   `validate:"required,boolean"`
}

// Perform a head request to the upstream to check if the resource should be accessed
func checkSource(client *http.Client, req *http.Request, upstream, proxyPath string) error {

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

			err := checkSource(srv.Node.Client, r, srv.Upstream.Address, proxyPath)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto runProxy
			}

			layer := path.Base(r.URL.Path)
			nodeStat, err := srv.Node.FindSource(r.Context(), layer)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto runProxy
			}

			logrus.Warn(layer)
			logrus.Warn(nodeStat)

			goto runFakeProxy

			// get layer using *GetLayer()
			// get node using FindNode(layer)
			// if not found, goto runFakeProxy
			// if found change the r.URL to the node URL & goto runFakeProxy
			// TODO: if sha256CheckEnabled => calculate & compare sha256
			// when download is completed notifyLayer()
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
	address := fmt.Sprintf("%s:%d", srv.Node.IPv4, srv.Node.Port)

	// handle all requests to your server using the proxy

	logrus.Infof("starting up server on %s", address)

	http.Handle("/data/", http.StripPrefix("/data/", fs))
	http.HandleFunc(fmt.Sprintf("%s/", proxyPath), srv.ProxyRequestHandler(proxy, fakeProxy, proxyPath))

	log.Fatal(http.ListenAndServe(address, nil))
	return nil
}
