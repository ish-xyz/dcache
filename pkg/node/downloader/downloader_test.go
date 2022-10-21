package downloader

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var downloaderTestsDir string

func setupDummyDownloader() *Downloader {
	downloaderTestsDir = "/tmp/dcache/downloader-tests"
	os.MkdirAll(downloaderTestsDir, os.FileMode(0755))
	logger := logrus.New()
	maxAtime, _ := time.ParseDuration("5m")
	interval, _ := time.ParseDuration("5s")
	d := NewDownloader(
		logger.WithField("component", "downloader-testing"),
		downloaderTestsDir,
		maxAtime,
		interval,
		10*1024*1024*1024,
		10,
	)
	return d
}

func TestPushOK(t *testing.T) {

	d := setupDummyDownloader()
	myReq, _ := http.NewRequest(
		"GET",
		"https:/null.null",
		nil,
	)

	d.Push(myReq, "/tmp/mydatadir/myitem")
	it, err := d.Pop(false)

	assert.Equal(t, len(d.Stack), 0)
	assert.Nil(t, err)
	assert.Equal(t, it.FilePath, "/tmp/mydatadir/myitem")
	assert.Equal(t, it.Req, myReq)
}

func TestQueueEmpty(t *testing.T) {
	d := setupDummyDownloader()

	it, err := d.Pop(false)

	assert.Equal(t, len(d.Stack), 0)
	assert.NotNil(t, err)
	assert.Nil(t, it)
}

func TestRunOK(t *testing.T) {

	d := setupDummyDownloader()
	myfile := fmt.Sprintf("%s/myfile.test", downloaderTestsDir)
	d.DryRun = true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "somestatus"}`)
	}))
	myreq, myreqErr := http.NewRequest(http.MethodGet, srv.URL, nil)
	d.Push(myreq, myfile)
	d.Run()
	statData, statErr := os.Stat(myfile)

	assert.Nil(t, myreqErr)
	assert.Nil(t, statErr)
	assert.Equal(t, statData.Name(), filepath.Base(myfile))

	os.Remove(myfile)
}

func TestRunDownloadFailed(t *testing.T) {

	d := setupDummyDownloader()
	myfile := fmt.Sprintf("%s/myfile.test", downloaderTestsDir)
	d.DryRun = true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	myreq, myreqErr := http.NewRequest(http.MethodGet, srv.URL, nil)
	d.Push(myreq, myfile)
	d.Run()
	statData, statErr := os.Stat(myfile)

	assert.Nil(t, myreqErr)
	assert.NotNil(t, statErr)
	assert.Equal(t, statData, nil)

	os.Remove(myfile)
}

func TestRunDownloadKillSwitch(t *testing.T) {

	d := setupDummyDownloader()
	myfile := fmt.Sprintf("%s/myfile.test", downloaderTestsDir)
	d.DryRun = true
	killswitch.Trigger = true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	myreq, myreqErr := http.NewRequest(http.MethodGet, srv.URL, nil)
	d.Push(myreq, myfile)
	d.Run()

	assert.Nil(t, myreqErr)
	assert.Equal(t, 1, len(d.Stack))

	os.Remove(myfile)
}

func TestRunDownloadFailedRetry(t *testing.T) {

	d := setupDummyDownloader()
	myfile := fmt.Sprintf("%s/myfile.test", downloaderTestsDir)
	d.DryRun = true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	myreq, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	d.Push(myreq, myfile)
	d.Run()

	assert.Equal(t, 1, len(d.Stack))
	os.Remove(myfile)
}
