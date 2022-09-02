package node

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"

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

			upsReq := fmt.Sprintf("%s%s", srv.Upstream.Address, r.RequestURI)
			resp, err := http.Head(upsReq)
			if err != nil {
				goto upstream
			}
			// head request should have no body, but it's best practice to close the body
			defer resp.Body.Close()

			is2xx := resp.StatusCode >= 200 && resp.StatusCode < 300
			if !is2xx {
				goto upstream
			}

			// TODO: contact scheduler for download
			// Call scheduler and ask to schedule()

			// Download from node if found
			// calculate SHA256, verify SHA256
			// if SHA256 is valid, communicate to scheduler that this node has that layer too

			fmt.Println("caching")
			proxy.ServeHTTP(w, r)
			return
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

	proxy := httputil.NewSingleHostReverseProxy(url)
	fmt.Println(srv.DataDir)
	fs := http.FileServer(http.Dir(srv.DataDir))

	// handle all requests to your server using the proxy
	logrus.Infof("starting up server on %s", srv.Address)

	http.Handle("/data/", http.StripPrefix("/data/", fs))
	http.HandleFunc("/proxy", srv.ProxyRequestHandler(proxy))

	log.Fatal(http.ListenAndServe(srv.Address, nil))
	return nil
}
