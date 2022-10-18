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
	DryRun        bool
}

type INotifier interface {
	Subscribe(ev chan *Event)
	Run() error
}

func NewNotifier(dataDir string, log *logrus.Entry) *Notifier {

	return &Notifier{
		DataDir:       dataDir,
		Logger:        log,
		Subscriptions: make([]chan *Event, 0),
		DryRun:        false,
	}
}

func (nt *Notifier) Subscribe(ev chan *Event) {
	nt.mu.Lock()
	nt.Subscriptions = append(nt.Subscriptions, ev)
	nt.mu.Unlock()
}

func (nt *Notifier) Run() error {

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

				for _, ch := range nt.Subscriptions {
					select {
					case ch <- ntEvent:
						nt.Logger.Debugf("successfully sent event %+v to %+v", ntEvent, ch)
					default:
						nt.Logger.Errorf("failed to send event %+v to %+v", ntEvent, ch)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				nt.Logger.Errorln("watcher error:", err)

			default:
				if nt.DryRun {
					return
				}
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
