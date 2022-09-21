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
func headRequestCheck(client *http.Client, req *http.Request, upstream, proxyPath string) (*http.Response, error) {

	// Prepare HEAD request
	ctx := req.Context()
	headReq := req.Clone(ctx)
	httpResource := fmt.Sprintf("%s%s", upstream, strings.TrimPrefix(req.RequestURI, proxyPath))
	headReq.Host = strings.Split(upstream, "://")[1]
	headReq.RequestURI = "" // it's illegal to have RequestURI predefined
	headReq.Method = "HEAD"
	u, err := url.Parse(httpResource)
	if err != nil {
		return nil, fmt.Errorf("url parsing failed for %s", httpResource)
	} else {
		headReq.URL = u
	}

	headResp, err := client.Do(headReq)
	if err != nil {
		return nil, fmt.Errorf("error with HEAD request %v", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != 200 {
		return nil, fmt.Errorf("HEAD request status code is not 200, is: %d", headResp.StatusCode)
	}

	return headResp, nil

}

// Generate item hash
// TODO: Hashing is not really the best solution here, encoding or smth else might be better
// not going with b64 as indexes would become damn long
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

			// HEAD request is necessary to see if the upstream allows us to download/serve certain content
			headResp, err := headRequestCheck(srv.Node.Client, r, srv.Upstream.Address, proxyPath)
			if err != nil {
				logrus.Warnln("falling back to upstream, because of error:", err)
				goto runProxy
			}

			// Scenario 1:
			// - item is already present in the local cache
			// - check connections limit
			// - serve if allowed
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
			// - ask scheduler for peer
			// - peer found
			// - redirect request to peer
			// - download item locally for next requests

			// Scenario 3:
			// - ask scheduler for peer
			// - peer not found
			// - download item locally for next requests
			// - redirect to upstream

			peer, err := srv.Node.FindSource(r.Context(), item)
			if err != nil {
				logrus.Warnf("unable to find peer able to serve the requested item %s.", item)
				logrus.Debugln(err)
				goto runProxy
				// go downloadItem(r.URL, etag, item) from upstream
			}

			// TODO: downloadItem(peer.IPv4, etag, item) from peer

			// A second HEAD request is necessary to see if peer can correctly serve the content
			_, err = headRequestCheck(srv.Node.Client, r, srv.Upstream.Address, proxyPath)
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

func notify() {
	fmt.Println("TODO")
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
