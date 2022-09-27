package server

import (
	"context"
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

// Perform an http request and checks the status code
func runRequestCheck(client *http.Client, req *http.Request) (*http.Response, error) {

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error with request %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 304 {
		return nil, fmt.Errorf("status code of %s request is not 200 or 304, is: %d", req.Method, resp.StatusCode)
	}

	return resp, nil

}

// Return a deep copy of request
func copyRequest(ctx context.Context, orig *http.Request, newurl, newhost, method string) (*http.Request, error) {

	newReq := orig.Clone(ctx)
	newReq.Host = newhost
	newReq.RequestURI = "" // it's illegal to have RequestURI predefined
	newReq.Method = method
	u, err := url.Parse(newurl)
	if err != nil {
		return nil, err
	}
	newReq.URL = u
	return newReq, nil
}
