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
	Client         node.NodeClient        `validate:"required"`
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

// TODO this can probably be improved, struct is too big and the args on this function are too much
func NewNode(
	nc node.NodeClient,
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
		Client:         nc,
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
func (no *Node) ProxyRequestHandler(upstreamProxy, peerProxy *httputil.ReverseProxy, proxyPath string) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		// TODO: what happens if we allow multiple HTTP methods?
		if no.Regex.MatchString(r.RequestURI) && r.Method == "GET" {

			no.Logger.Debugln("regex matched for ", r.RequestURI)

			url := fmt.Sprintf("%s%s", no.Upstream.Address, strings.TrimPrefix(r.RequestURI, proxyPath))
			host := strings.Split(no.Upstream.Address, "://")[1]

			// prepare HEAD request
			headReq, err := copyRequest(r.Context(), r, url, host, http.MethodHead)
			if err != nil {
				no.Logger.Errorln("Error parsing http resource for head request:", err)
				no.runProxy(upstreamProxy, w, r)
				return
			}

			// HEAD request is necessary to see if the upstream allows us to download/serve certain content
			headResp, err := runRequestCheck(no.Client.GetHttpClient(), headReq)
			if err != nil {
				no.Logger.Warnln("falling back to upstream, because of error:", err)
				no.runProxy(upstreamProxy, w, r)
				return
			}

			// File found in local cache, try to serve it
			item := generateHash(r.URL, headResp.Header["Etag"][0])
			no.Logger.Debugf("item name: %s", item)

			filepath := fmt.Sprintf("%s/%s", no.DataDir, item)
			if _, err := os.Stat(filepath); err == nil {
				selfInfo, err := no.Client.Info()
				if err != nil {
					no.Logger.Errorln("failed to contact scheduler to get node info, fallingback to upstream")
					no.runProxy(upstreamProxy, w, r)
					return
				}

				no.Logger.Debugln("checking connections, retrieved node info", selfInfo)
				if selfInfo.Connections+1 < selfInfo.MaxConnections {
					no.ServeSingleFile(w, r, filepath)
					return
				}
				// TODO: this can be removed but we need to find a way to limit the maximum amount of jumps
				no.Logger.Warnln("max connections for peer reached, redirecting to upstream")
				no.runProxy(upstreamProxy, w, r)
				return
			}

			// File not found in local cache, try to find a suitable peer
			peerinfo, err := no.Client.Schedule(item)
			if err != nil {
				no.Logger.Errorln("error looking for peer:", err)
				no.runProxy(upstreamProxy, w, r)
			} else {
				rewriteToPeer(r, peerinfo)
				url = fmt.Sprintf("%s://%s:%d/%s", peerinfo.Scheme, peerinfo.IPv4, peerinfo.Port, r.URL.Path)
				host = fmt.Sprintf("%s:%d", peerinfo.IPv4, peerinfo.Port)
				no.runProxy(peerProxy, w, r)
			}

			no.Logger.Debugln("heating cache from:", url)
			// NOTE: we can't pass r.Context() to copyRequest because the download
			// will  most likely be processed after the request has been served and the contex wil get canceled
			// Remove this comment when a test has been implemented
			downloaderReq, _ := copyRequest(context.TODO(), r, url, host, http.MethodGet)
			err = no.Downloader.Push(downloaderReq, filepath)
			if err != nil {
				no.Logger.Errorf("failed to push file %s into downloader queue", filepath)
			}
			return
		}
		no.runProxy(upstreamProxy, w, r)
	}
}

func (no *Node) runProxy(proxy *httputil.ReverseProxy, w http.ResponseWriter, r *http.Request) {
	no.Logger.Infoln("proxying request for:", r.URL.String())
	proxy.ServeHTTP(w, r)
}

func (no *Node) ServeSingleFile(w http.ResponseWriter, r *http.Request, itemPath string) {

	err := no.Client.AddConnection()
	if err != nil {
		no.Logger.Errorln("failed to add connection to scheduler")
	}

	no.Logger.Infoln("serving file", r.RequestURI)
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
