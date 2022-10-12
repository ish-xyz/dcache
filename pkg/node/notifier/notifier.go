package notifier

import (
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/ish-xyz/dcache/pkg/node"
	"github.com/sirupsen/logrus"
)

type Notifier struct {
	NodeClient node.NodeClient
	DataDir    string
	Logger     *logrus.Entry
	DryRun     bool
}

func NewNotifier(nc node.NodeClient, dataDir string, log *logrus.Entry) *Notifier {
	return &Notifier{
		NodeClient: nc,
		DataDir:    dataDir,
		Logger:     log,
		DryRun:     false,
	}
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

				if event.Op == 1 {
					nt.Logger.Infof("CREATE event received for %s", item)
					err = nt.NodeClient.NotifyItem(item, int(event.Op))

				} else if event.Op == 4 {
					nt.Logger.Infof("REMOVE event received for %s", item)
					err = nt.NodeClient.NotifyItem(item, int(event.Op))
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
