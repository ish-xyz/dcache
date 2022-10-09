package downloader

import (
	"net/http"
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
		logger.WithField("component", "testing"),
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
	it := d.Pop()

	assert.Equal(t, len(d.Queue), 0)
	assert.Equal(t, it.FilePath, "/tmp/mydatadir/myitem")
	assert.Equal(t, it.Req, myReq)
}
