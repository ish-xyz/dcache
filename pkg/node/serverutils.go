package node

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func downloadItem(client *http.Client, filepath string, url string) error {

	// head request

	// calculate hash
	// check if item hash is equal to retrieved hash

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
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

// Perform a head request to the upstream to check if the resource should be accessed
func headRequestCheck(client *http.Client, req *http.Request) (*http.Response, error) {

	headResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error with HEAD request %v", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != 200 {
		return nil, fmt.Errorf("HEAD request status code is not 200, is: %d", headResp.StatusCode)
	}

	return headResp, nil

}
