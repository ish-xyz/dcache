package notifier

import (
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/ish-xyz/dcache/pkg/node/client"
	"github.com/sirupsen/logrus"
)

type Event struct {
	Item string
	Op   int
}

type Notifier struct {
	mu            sync.Mutex
	NodeClient    client.IClient
	DataDir       string
	Logger        *logrus.Entry
	Subscriptions []chan *Event
	DryRun        bool
}

func NewNotifier(nc client.IClient, dataDir string, log *logrus.Entry) *Notifier {
	return &Notifier{
		NodeClient: nc,
		DataDir:    dataDir,
		Logger:     log,
		DryRun:     false,
	}
}

func (nt *Notifier) Subscribe(ev chan *Event) {
	nt.mu.Lock()
	nt.Subscriptions = append(nt.Subscriptions, ev)
	nt.mu.Unlock()
}

func (nt *Notifier) Watch() error {

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
				var err error

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

				// TODO: move to subscription model
				if event.Op == 1 {
					nt.Logger.Infof("CREATE event received for %s", item)
					err = nt.NodeClient.CreateItem(item)

				} else if event.Op == 4 {
					nt.Logger.Infof("REMOVE event received for %s", item)
					err = nt.NodeClient.DeleteItem(item)
				}

				if err != nil {
					nt.Logger.Errorf("failed to notify (%s) item %s to scheduler", item, event.Op.String())
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
