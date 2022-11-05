package notifier

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup() *Notifier {
	notifierTestsDir := "/tmp/dcache/notifier-tests"
	os.MkdirAll(notifierTestsDir, os.FileMode(0755))
	logger := logrus.New()
	nt := NewNotifier(
		notifierTestsDir,
		logger.WithField("component", "downloader-testing"),
	)
	return nt
}

func TestSubscribeOK(t *testing.T) {
	nt := setup()
	ch := make(chan *Event)
	nt.Subscribe(ch)

	assert.Equal(t, len(nt.Subscriptions), 1)
}

// Fixed!!!!
func TestBroadcast(t *testing.T) {
	nt := setup()
	ch := make(chan *Event)
	event := &Event{"somepath", 16}
	nt.Subscribe(ch)

	go nt.Run(true)

	time.Sleep(time.Second * 2)

	os.Create(fmt.Sprintf("%s/tmp.tmp", nt.DataDir))

	receivedEvent := <-ch

	assert.Equal(t, event, receivedEvent)
	assert.Equal(t, len(nt.Subscriptions), 1)
}
