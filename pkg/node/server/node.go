package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ish-xyz/dcache/pkg/node"
	"github.com/ish-xyz/dcache/pkg/node/downloader"
	"github.com/sirupsen/logrus"
)

type UpstreamConfig struct {
	Address  string `validate:"required,url"`
	Insecure bool   `validate:"required"` // add boolean validator
}

type Node struct {
	Client         *node.Client           `validate:"required"`
	Upstream       *UpstreamConfig        `validate:"required,dive"`
	DataDir        string                 `validate:"required"` // Add dir validator
	Scheme         string                 `validate:"required"`
	IPv4           string                 `validate:"required,ipv4"`
	Port           int                    `validate:"required,number"`
	MaxConnections int                    `validate:"required,number"`
	Downloader     *downloader.Downloader `validate:"required"`
	Regex          *regexp.Regexp         `validate:"required"`
	Logger         *logrus.Entry          `validate:"required"`
}

//TODO this can probably be improved, struct is too big and the args on this function are too much
func NewNode(
	nodeObj *node.Client,
	uconf *UpstreamConfig,
	dataDir,
	scheme,
	ipv4 string,
	port,
	maxconn int,
	dw *downloader.Downloader,
	re *regexp.Regexp,
	lg *logrus.Entry,
) *Node {

	return &Node{
		Client:         nodeObj,
		Upstream:       uconf,
		DataDir:        strings.TrimSuffix(dataDir, "/"),
		Scheme:         strings.TrimSuffix(scheme, "://"),
		IPv4:           ipv4,
		Port:           port,
		MaxConnections: maxconn,
		Downloader:     dw,
		Regex:          re,
		Logger:         lg,
	}
}

// ProxyRequestHandler handles the http request using proxy
func (no *Node) ProxyRequestHandler(proxy, fakeProxy *httputil.ReverseProxy, proxyPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// TODO: what happens if we allow multiple HTTP methods
		if no.Regex.MatchString(r.RequestURI) && r.Method == "GET" {

			no.Logger.Debugln("regex matched for ", r.RequestURI)

			upstreamUrl := fmt.Sprintf("%s%s", no.Upstream.Address, strings.TrimPrefix(r.RequestURI, proxyPath))
			upstreamHost := strings.Split(no.Upstream.Address, "://")[1]

			// prepare HEAD request
			headReq, err := copyRequest(r.Context(), r, upstreamUrl, upstreamHost, "HEAD")
			if err != nil {
				no.Logger.Errorln("Error parsing http resource for head request:", err)
				goto upstream
			}

			// HEAD request is necessary to see if the upstream allows us to download/serve certain content
			headResp, err := runRequestCheck(no.Client.HTTPClient, headReq)
			if err != nil {
				no.Logger.Warnln("falling back to upstream, because of error:", err)
				goto upstream
			}

			// scenario 1: item is already present in the local cache of the node
			item := generateHash(r.URL, headResp.Header["Etag"][0])
			no.Logger.Debugf("item name: %s", item)

			filepath := fmt.Sprintf("%s/%s", no.DataDir, item)
			if _, err := os.Stat(filepath); err == nil {
				selfInfo, err := no.Client.Info()
				if err != nil {
					no.Logger.Errorln("failed to contact scheduler to get node info, fallingback to upstream")
					goto upstream
				}

				no.Logger.Debugln("checking connections, retrieved node info", selfInfo)
				if selfInfo.Connections+1 < selfInfo.MaxConnections {
					no.ServeSingleFile(w, r, filepath)
					return
				}
				no.Logger.Warnln("max connections for peer already reached, redirecting to upstream")
				goto upstream // TODO: remove when scenario 2 it's implemented
			}

			no.Logger.Debugf("file %s not found in local cache, redirecting to upstream", item)
			no.Logger.Debugf("heating cache for next requests")

			// Note for myself: can't use r.Context() because the download
			// 	will get most likely processed after this request has finished and the contex canceled
			upstreamReq, _ := copyRequest(context.TODO(), r, upstreamUrl, upstreamHost, "GET")
			no.Downloader.Push(upstreamReq, filepath)

			goto upstream
		}

	upstream:
		no.Logger.Infof("request for %s is going to upstream", r.URL.String())
		proxy.ServeHTTP(w, r)
		//return

		// runFakeProxy:
		// 	no.Logger.Infof("request for %s is being cached", r.URL.String())
		// 	fakeProxy.ServeHTTP(w, r)
	}
}

func (no *Node) ServeSingleFile(w http.ResponseWriter, r *http.Request, itemPath string) {

	err := no.Client.AddConnection()
	if err != nil {
		no.Logger.Errorln("failed to add connection to scheduler")
	}

	no.Downloader.GC.UpdateAtime(filepath.Base(itemPath))

	http.ServeFile(w, r, itemPath)

	err = no.Client.RemoveConnection()
	if err != nil {
		no.Logger.Errorln("failed to remove connection from scheduler")
	}

}

func (no *Node) Run() error {

	proxyPath := "/proxy"
	address := fmt.Sprintf("%s:%d", no.IPv4, no.Port)
	fakeProxy := newFakeProxy()
	url, err := url.Parse(no.Upstream.Address)
	if err != nil {
		return err
	}
	proxy := newCustomProxy(url, proxyPath)

	no.Logger.Infof("starting up server on %s", address)
	http.HandleFunc(fmt.Sprintf("%s/", proxyPath), no.ProxyRequestHandler(proxy, fakeProxy, proxyPath))

	log.Fatal(http.ListenAndServe(address, nil))
	return nil
}
