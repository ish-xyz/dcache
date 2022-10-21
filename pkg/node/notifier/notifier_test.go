package notifier

import (
	"os"
	"testing"

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

// Not working
// func TestBroadcast(t *testing.T) {
// 	nt := setup()
// 	ch := make(chan *Event)

// 	event := &Event{"somepath", 1}
// 	receivedEvent := &Event{}
// 	chans := make([]chan *Event, 0)
// 	chans = append(chans, ch)
// 	nt.Broadcast(chans, event)

// 	select {
// 		case receivedEvent <- ch:
// 			_ = ""
// 		default:
// 			_ = ""
// 		}
// 	}

// 	assert.Equal(t, event, receivedEvent)

// 	//assert.NotNil(t, <-ch)
// 	assert.Equal(t, len(nt.Subscriptions), 1)
// }
