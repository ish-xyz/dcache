package main

import (
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/ish-xyz/dcache/pkg/node"
	"github.com/sirupsen/logrus"
)

type Notifier struct {
	NodeClient *node.Client
	SrvData    string
	Logger     *logrus.Entry
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

				if event.Op == 1 {
					nt.Logger.Infof("CREATE event received for %s", filepath.Base(event.Name))
					nt.NodeClient.NotifyItem(filepath.Base(event.Name), int(event.Op))
				} else if event.Op == 4 {
					nt.Logger.Infof("REMOVE event received for %s", filepath.Base(event.Name))
					nt.NodeClient.NotifyItem(filepath.Base(event.Name), int(event.Op))
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				nt.Logger.Errorln("watcher error:", err)
			}
		}
	}()

	err = watcher.Add(nt.SrvData)
	if err != nil {
		return err
	}
	<-done

	return nil
}
