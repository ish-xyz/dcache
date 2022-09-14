package node

import (
	"crypto/sha256"
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

// Perform a head request to the upstream to check if the resource should be accessed
func headRequest(client *http.Client, req *http.Request, upstream, proxyPath string) (error, http.Header) {

	// Prepare HEAD request
	ctx := req.Context()
	headReq := req.Clone(ctx)
	httpResource := fmt.Sprintf("%s%s", upstream, strings.TrimPrefix(req.RequestURI, proxyPath))
	headReq.Host = strings.Split(upstream, "://")[1]
	headReq.RequestURI = "" // it's illegal to have RequestURI predefined
	headReq.Method = "HEAD"
	u, err := url.Parse(httpResource)
	if err != nil {
		return fmt.Errorf("url parsing failed for %s", httpResource), nil
	} else {
		headReq.URL = u
	}

	// Perform HEAD request
	headResp, err := client.Do(headReq)
	if err != nil {
		return fmt.Errorf("error with HEAD request %v", err), nil
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != 200 {
		return fmt.Errorf("HEAD request status code is not 200, is: %d", headResp.StatusCode), nil
	}

	return nil, headResp.Header

}

// Generate item hash
// TODO: Hashing is not really the best solution here, encoding or smth else might be better
func generateHash(url *url.URL, etag string) string {
	id := fmt.Sprintf("%s.%s", url.String(), etag)
	sumBytes := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", sumBytes)
}

// Helper function to set the peer recipient
func modifyRequest(r *http.Request, nodestat *NodeStat, dataDir, item string) error {

	r.URL.Scheme = nodestat.Scheme
	r.URL.Host = nodestat.IPv4
	r.Host = nodestat.IPv4
	r.URL.Path = fmt.Sprintf("/%s/%s", dataDir, item)
	r.RequestURI = fmt.Sprintf("/%s/%s", dataDir, item)

	if _, ok := r.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		r.Header.Set("User-Agent", "")
	}

	return nil
}

// ProxyRequestHandler handles the http request using proxy
func (srv *Server) ProxyRequestHandler(proxy, fakeProxy *httputil.ReverseProxy, proxyPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if srv.Regex.MatchString(r.RequestURI) {

			logrus.Debugln("regex matched for ", r.RequestURI)

			err, headers := headRequest(srv.Node.Client, r, srv.Upstream.Address, proxyPath)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto runProxy
			}

			// Scenario 1: Item is already present in the local node, check connections limit and serve if possible
			item := generateHash(r.URL, headers["Etag"][0])
			filepath := fmt.Sprintf("%s/%s", srv.DataDir, item)
			if _, err := os.Stat(filepath); err == nil {
				self, err := srv.Node.Stat(r.Context())
				if err != nil {
					logrus.Errorln("failed to contact scheduler to get nodestat, fallingback to upstream")
					goto runProxy
				}

				// TODO: check node max connections
				modifyRequest(r, self, srv.DataDir, item)
				goto runFakeProxy

			}

			// Scenario 2a: ask scheduler for peer, redirect request to peer and download item locally for next requests
			// Scenario 2b: ask scheduler for peer, peer not found, download item locally for next requests and redirect to upstream
			peer, err := srv.Node.FindSource(r.Context(), item)
			if err != nil {
				// TODO: downloadItem() from source
				logrus.Infoln("can't find peer able to serve item. Caching for next request and falling back to upstream")
				logrus.Debugln(err)
				goto runProxy
			}
			// TODO: downloadItem() from peer
			// modifyRequest(r, peer, srv.DataDir, item)

			fmt.Println(peer)

			goto runProxy // TODO: Change to fake proxy
			goto runFakeProxy
		}

	runProxy:
		logrus.Debugln("request is going to upstream")
		proxy.ServeHTTP(w, r)

	runFakeProxy:
		logrus.Debugln("request is being cached")
		proxy.ServeHTTP(w, r)

		logrus.Debugln("register to scheduler")
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
