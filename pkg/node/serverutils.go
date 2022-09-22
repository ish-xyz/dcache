package node

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
)

// Generate item hash
// TODO: Hashing is not really the best solution here, encoding or smth else might be better.
// I'm not going with b64 as indexes would become damn long
func generateHash(url *url.URL, etag string) string {
	id := fmt.Sprintf("%s.%s", url.String(), etag)
	sumBytes := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", sumBytes)
}

// Helper function to set the peer as server
func redirectRequestToPeer(r *http.Request, target *NodeStat, path string) error {

	r.URL.Scheme = target.Scheme
	r.URL.Host = fmt.Sprintf("%s:%d", target.IPv4, target.Port)
	r.Host = fmt.Sprintf("%s:%d", target.IPv4, target.Port)
	r.URL.Path = path
	r.RequestURI = path

	if _, ok := r.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		r.Header.Set("User-Agent", "")
	}

	return nil
}

// Perform an http request and checks the status code
func runRequestCheck(client *http.Client, req *http.Request) (*http.Response, error) {

	headResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error with request %v", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != 200 {
		return nil, fmt.Errorf("status code of %s request is not 200, is: %d", req.Method, headResp.StatusCode)
	}

	return headResp, nil

}

// Return a deep copy of request
func copyRequest(orig *http.Request, newurl, newhost, method string) (*http.Request, error) {
	headReq := orig.Clone(orig.Context())
	headReq.Host = newhost
	headReq.RequestURI = "" // it's illegal to have RequestURI predefined
	headReq.Method = method
	u, err := url.Parse(newurl)
	if err != nil {
		return nil, err
	}
	headReq.URL = u
	return headReq, nil
}
