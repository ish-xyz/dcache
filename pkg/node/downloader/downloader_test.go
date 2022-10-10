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

func TestPushOK(t *testing.T) {
	logger := logrus.New()
	maxAtime, _ := time.ParseDuration("5m")
	interval, _ := time.ParseDuration("5s")
	d := NewDownloader(
		logger.WithField("component", "downloader-testing"),
		"/tmp/mydatadir",
		maxAtime,
		interval,
		10*1024*1024*1024,
	)
	myReq, _ := http.NewRequest(
		"GET",
		"https:/null.null",
		nil,
	)

	d.Push(myReq, "/tmp/mydatadir/myitem")
	it, err := d.Pop(false)

	assert.Equal(t, len(d.Queue), 0)
	assert.Nil(t, err)
	assert.Equal(t, it.FilePath, "/tmp/mydatadir/myitem")
	assert.Equal(t, it.Req, myReq)
}

func TestQueueEmpty(t *testing.T) {
	logger := logrus.New()
	maxAtime, _ := time.ParseDuration("5m")
	interval, _ := time.ParseDuration("5s")
	d := NewDownloader(
		logger.WithField("component", "downloader-testing"),
		"/tmp/mydatadir",
		maxAtime,
		interval,
		10*1024*1024*1024,
	)

	it, err := d.Pop(false)

	assert.Equal(t, len(d.Queue), 0)
	assert.NotNil(t, err)
	assert.Nil(t, it)
}

func TestRunOK(t *testing.T) {

	logger := logrus.New()
	myfile := "/tmp/test-data-download"
	maxAtime, _ := time.ParseDuration("5m")
	interval, _ := time.ParseDuration("5s")
	d := NewDownloader(
		logger.WithField("component", "downloader-testing"),
		"/tmp/mydatadir",
		maxAtime,
		interval,
		10*1024*1024*1024,
	)
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

	os.Remove("/tmp/test-data-download")
}

func TestRunDownloadFailed(t *testing.T) {

	logger := logrus.New()
	myfile := "/tmp/test-data-download"
	maxAtime, _ := time.ParseDuration("5m")
	interval, _ := time.ParseDuration("5s")
	d := NewDownloader(
		logger.WithField("component", "downloader-testing"),
		"/tmp/mydatadir",
		maxAtime,
		interval,
		10*1024*1024*1024,
	)
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

	os.Remove("/tmp/test-data-download")
}

func TestRunDownloadKillSwitch(t *testing.T) {

	logger := logrus.New()
	myfile := "/tmp/test-data-download"
	maxAtime, _ := time.ParseDuration("5m")
	interval, _ := time.ParseDuration("5s")
	d := NewDownloader(
		logger.WithField("component", "downloader-testing"),
		"/tmp/mydatadir",
		maxAtime,
		interval,
		10*1024*1024*1024,
	)
	d.DryRun = true
	killswitch.Trigger = true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	myreq, myreqErr := http.NewRequest(http.MethodGet, srv.URL, nil)
	d.Push(myreq, myfile)
	d.Run()

	assert.Nil(t, myreqErr)
	assert.Equal(t, 1, len(d.Queue))

	os.Remove("/tmp/test-data-download")
}
