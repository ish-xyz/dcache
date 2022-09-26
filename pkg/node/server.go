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

	"github.com/ish-xyz/dcache/pkg/node/downloader"
	"github.com/sirupsen/logrus"
)

type Server struct {
	Node       *Node           `validate:"required"`
	Upstream   *UpstreamConfig `validate:"required,dive"`
	DataDir    string          `validate:"required"` // Add dir validator
	Downloader *downloader.Downloader
	Regex      *regexp.Regexp `validate:"required"`
}

type UpstreamConfig struct {
	Address  string `validate:"required,url"`
	Insecure bool   `validate:"required"` // add boolean validator
}

func NewServer(nodeObj *Node,
	dataDir string,
	uconf *UpstreamConfig,
	dw *downloader.Downloader,
	re *regexp.Regexp) *Server {

	return &Server{
		Node:       nodeObj,
		DataDir:    strings.TrimSuffix(dataDir, "/"),
		Upstream:   uconf,
		Downloader: dw,
		Regex:      re,
	}
}

// ProxyRequestHandler handles the http request using proxy
func (srv *Server) ProxyRequestHandler(proxy, fakeProxy *httputil.ReverseProxy, proxyPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if srv.Regex.MatchString(r.RequestURI) && r.Method == "GET" {

			logrus.Debugln("regex matched for ", r.RequestURI)

			upstreamUrl := fmt.Sprintf("%s%s", srv.Upstream.Address, strings.TrimPrefix(r.RequestURI, proxyPath))
			upstreamHost := strings.Split(srv.Upstream.Address, "://")[1]

			// prepare HEAD request
			headReq, err := copyRequest(r.Context(), r, upstreamUrl, upstreamHost, "HEAD")
			if err != nil {
				logrus.Errorln("Error parsing http resource for head request:", err)
				goto upstream
			}

			// HEAD request is necessary to see if the upstream allows us to download/serve certain content
			headResp, err := runRequestCheck(srv.Node.Client, headReq)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto upstream
			}

			// scenario 1: item is already present in the local cache of the node
			item := generateHash(r.URL, headResp.Header["Etag"][0])
			logrus.Debugf("item name: %s", item)

			filepath := fmt.Sprintf("%s/%s", srv.DataDir, item)
			if _, err := os.Stat(filepath); err == nil {
				selfInfo, err := srv.Node.Info()
				if err != nil {
					logrus.Errorln("failed to contact scheduler to get node info, fallingback to upstream")
					goto upstream
				}

				logrus.Debugln("retrieved node info", selfInfo)
				if selfInfo.Connections+1 >= selfInfo.MaxConnections {
					logrus.Warnln("Max connections for peer already reached, redirecting to upstream")
					goto upstream // TODO: to be removed when scenario 2 it's implemented
				} else {
					srv.ServeFile(w, r, filepath)
					return
				}
			}

			logrus.Debugf("file %s not found in local cache, redirecting to upstream", item)
			logrus.Debugf("heating cache for next requests")
			// Note for myself: can't use r.Context() because the download
			// 	will get most likely processed after this request has finished and the contex canceled
			upstreamReq, err := copyRequest(context.TODO(), r, upstreamUrl, upstreamHost, "GET")
			if err == nil {
				logrus.Debugln("request err", err)
				srv.Downloader.Push(upstreamReq, filepath)
			}

			goto upstream

		}

	upstream:
		logrus.Infof("request for %s is going to upstream", r.URL.String())
		proxy.ServeHTTP(w, r)
		//return

		// runFakeProxy:
		// 	logrus.Infof("request for %s is being cached", r.URL.String())
		// 	fakeProxy.ServeHTTP(w, r)
	}
}

func (srv *Server) ServeFile(w http.ResponseWriter, r *http.Request, filePath string) {

	err := srv.Node.AddConnection()
	if err != nil {
		logrus.Errorln("failed to add connection to scheduler")
	}

	http.ServeFile(w, r, filePath)

	err = srv.Node.RemoveConnection()
	if err != nil {
		logrus.Errorln("failed to remove connection from scheduler")
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
	address := fmt.Sprintf("%s:%d", srv.Node.IPv4, srv.Node.Port)

	logrus.Infof("starting up server on %s", address)
	http.HandleFunc(fmt.Sprintf("%s/", proxyPath), srv.ProxyRequestHandler(proxy, fakeProxy, proxyPath))

	go srv.Downloader.Watch()

	log.Fatal(http.ListenAndServe(address, nil))
	return nil
}
