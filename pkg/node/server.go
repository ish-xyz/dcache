package node

import (
	"crypto/sha256"
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

func generateHash(url *url.URL, etag string) string {
	id := fmt.Sprintf("%s.%s", url.String(), etag)
	sumBytes := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", sumBytes)
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

			item := generateHash(r.URL, headers["Etag"][0])

			goto runProxy

			nodestat, err := srv.Node.FindSource(r.Context(), item)
			if err != nil {
				logrus.Infoln("can't find peer able to serve item. Falling back to upstream")
				logrus.Debugln(err)
				goto runProxy
			}

			r.URL.Scheme = nodestat.Scheme
			r.URL.Host = nodestat.IPv4
			r.Host = nodestat.IPv4
			r.URL.Path = strings.TrimPrefix(r.URL.Path, proxyPath)
			r.RequestURI = strings.TrimPrefix(r.RequestURI, proxyPath)

			if _, ok := r.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				r.Header.Set("User-Agent", "")
			}

			goto runFakeProxy
		}
	runProxy:
		logrus.Info("running proxy")
		proxy.ServeHTTP(w, r)
	runFakeProxy:
		logrus.Info("running fakeProxy")
		proxy.ServeHTTP(w, r)

		fmt.Println("post serve http")
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
