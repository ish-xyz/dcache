package node

import (
	"context"
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

		requestID := generateRequestID()

		if srv.Regex.MatchString(r.RequestURI) && r.Method == "GET" {

			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			logrus.Debugln("regex matched for ", r.RequestURI)

			upstreamUrl := fmt.Sprintf("%s%s", srv.Upstream.Address, strings.TrimPrefix(r.RequestURI, proxyPath))
			upstreamHost := strings.Split(srv.Upstream.Address, "://")[1]

			// prepare HEAD request
			headReq, err := copyRequest(r, upstreamUrl, upstreamHost, "HEAD")
			if err != nil {
				logrus.Errorln("Error parsing http resource for head request:", err)
				goto runProxy
			}

			// HEAD request is necessary to see if the upstream allows us to download/serve certain content
			headResp, err := runRequestCheck(srv.Node.Client, headReq)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto runProxy
			}

			// scenario 1: item is already present in the local cache of the node
			item := generateHash(r.URL, headResp.Header["Etag"][0])
			filePathOS := fmt.Sprintf("%s/%s", srv.DataDir, item)
			fileServerPath := fmt.Sprintf("/fs/%s", item)

			if _, err := os.Stat(filePathOS); err == nil {
				selfInfo, err := srv.Node.Info(ctx)
				if err != nil {
					logrus.Errorln("failed to contact scheduler to get node info, fallingback to upstream")
					goto runProxy
				}

				logrus.Debugln("retrieved node info", selfInfo)

				if selfInfo.Connections+1 >= selfInfo.MaxConnections {
					logrus.Warnln("Max connections for peer already reached")
					goto runProxy
				} else {
					proxyToPeer(r, selfInfo, fileServerPath)
					goto runFakeProxy
				}

			} else {
				logrus.Debugf("file %s not found in local cache, redirecting to upstream", item)
				goto runProxy
			}

		}

	runProxy:
		logrus.Debugln("request is going to upstream")
		proxy.ServeHTTP(w, r)
		return

	runFakeProxy:
		logrus.Debugln("request is being cached")
		fakeProxy.ServeHTTP(w, r)
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
