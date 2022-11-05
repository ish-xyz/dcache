package notifier

import (
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type Event struct {
	Item string
	Op   int
}

type Notifier struct {
	mu            sync.Mutex
	DataDir       string
	Logger        *logrus.Entry
	Subscriptions []chan *Event
}

type INotifier interface {
	Subscribe(ev chan *Event)
	Run(bool) error
}

func NewNotifier(dataDir string, log *logrus.Entry) *Notifier {

	return &Notifier{
		DataDir:       dataDir,
		Logger:        log,
		Subscriptions: make([]chan *Event, 0),
	}
}

func (nt *Notifier) Subscribe(ev chan *Event) {
	nt.mu.Lock()
	nt.Subscriptions = append(nt.Subscriptions, ev)
	nt.mu.Unlock()
}

func (nt *Notifier) Broadcast(subs []chan *Event, event *Event) {
	for _, ch := range subs {
		select {
		case ch <- event:
			nt.Logger.Debugf("successfully sent event %+v to %+v", event, ch)
		default:
			nt.Logger.Errorf("failed to send event %+v to %+v", event, ch)
		}
	}
}

func (nt *Notifier) Run(once bool) error {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)
		for {
			select {

			case event, ok := <-watcher.Events:

				if !ok {
					return
				}

				item := filepath.Base(event.Name)
				ntEvent := &Event{
					Item: item,
					Op:   int(event.Op),
				}
				nt.Broadcast(nt.Subscriptions, ntEvent)

				if once {
					return
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				nt.Logger.Errorln("watcher error:", err)
			}
		}
	}()

	err = watcher.Add(nt.DataDir)
	if err != nil {
		nt.Logger.Errorln("notifier error:", err)
		return err
	}
	<-done

	return nil
}
