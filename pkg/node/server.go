package node

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type Server struct {
	Node     *Node           `validate:"required"`
	Upstream *UpstreamConfig `validate:"required,dive"`
	DataDir  string          `validate:"required"` // Add dir validator
	Regex    *regexp.Regexp  `validate:"required"`
}

type UpstreamConfig struct {
	Address  string `validate:"required,url"`
	Insecure bool   `validate:"required"` // add boolean validator
}

// ProxyRequestHandler handles the http request using proxy
func (srv *Server) ProxyRequestHandler(proxy, fakeProxy *httputil.ReverseProxy, proxyPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if srv.Regex.MatchString(r.RequestURI) {

			logrus.Debugln("regex matched for ", r.RequestURI)

			// Prepare HEAD request
			headReq := r.Clone(r.Context())
			headReqURL := fmt.Sprintf("%s%s", srv.Upstream.Address, strings.TrimPrefix(r.RequestURI, proxyPath))
			headReq.Host = strings.Split(srv.Upstream.Address, "://")[1]
			headReq.RequestURI = "" // it's illegal to have RequestURI predefined
			headReq.Method = "HEAD"
			u, err := url.Parse(headReqURL)
			if err != nil {
				logrus.Errorln("Error parsing http resource for head request:", err)
				goto runProxy
			} else {
				headReq.URL = u
			}

			// HEAD request is necessary to see if the upstream allows us to download/serve certain content
			headResp, err := headRequestCheck(srv.Node.Client, r)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto runProxy
			}

			// Scenario 1:
			// * item is already present in the local cache
			// * check connections limit
			// * serve if allowed
			item := generateHash(r.URL, headResp.Header["Etag"][0])
			filepath := fmt.Sprintf("%s/%s", srv.DataDir, item)
			if _, err := os.Stat(filepath); err == nil {
				selfstat, err := srv.Node.Stat(r.Context())
				if err != nil {
					logrus.Errorln("failed to contact scheduler to get nodestat, fallingback to upstream")
					goto runProxy
				}

				if selfstat.Connections+1 < selfstat.MaxConnections {
					modifyRequest(r, selfstat, srv.DataDir, item)
					goto runFakeProxy
				}
			}

			// Scenario 2:
			// * ask scheduler for peer
			// * peer not found
			// * download item locally for next requests
			// * redirect to upstream
			peer, err := srv.Node.FindSource(r.Context(), item)
			if err != nil {
				logrus.Warnf("unable to find peer able to serve the requested item %s.", item)
				logrus.Debugln(err)
				goto runProxy
				// go downloadItem(r.URL, etag, item) from upstream
			}

			// Scenario 3:
			// * ask scheduler for peer
			// * peer found
			// * redirect request to peer
			// * download item locally for next requests

			// TODO: go downloadItem(peer.IPv4, etag, item) from peer

			// TODO: A second HEAD request is necessary to see if peer can correctly serve the content

			if err != nil {
				logrus.Warnln("peer returned error, falling back to upstream")
				logrus.Debugln(err)
				//TODO: notify that peer doesn't have specific item
				goto runProxy
			}
			modifyRequest(r, peer, srv.DataDir, item)
			goto runFakeProxy
		}

	runProxy:
		logrus.Debugln("request is going to upstream")
		proxy.ServeHTTP(w, r)
		return

	runFakeProxy:
		logrus.Debugln("request is being cached")
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

	http.Handle("/fs/", http.StripPrefix("/fs/", fs))
	http.HandleFunc(fmt.Sprintf("%s/", proxyPath), srv.ProxyRequestHandler(proxy, fakeProxy, proxyPath))

	log.Fatal(http.ListenAndServe(address, nil))
	return nil
}
