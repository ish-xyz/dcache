package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	killswitch KillSwitch
)

type KillSwitch struct {
	Trigger bool
	mu      sync.Mutex
}

type Downloader struct {
	Queue  chan *Item    `validate:"required"`
	Client *http.Client  `validate:"required"`
	Logger *logrus.Entry `validate:"required"`
	GC     *GC           `validate:"required"`
	DryRun bool
}

type Item struct {
	Req      *http.Request
	FilePath string
}

func NewDownloader(log *logrus.Entry, dataDir string, maxAtime, interval time.Duration, maxDiskUsage int) *Downloader {

	cache := &FilesCache{
		AtimeStore: make(map[string]int64),
		FilesByAge: make([]string, 1),
		FilesSize:  make(map[string]int64),
	}

	gc := &GC{
		MaxAtimeAge:  maxAtime,
		MaxDiskUsage: maxDiskUsage,
		Interval:     interval,
		DataDir:      dataDir,
		Logger:       log.WithField("component", "node.downloader.gc"),
		Cache:        cache,
		DryRun:       false,
	}

	return &Downloader{
		Queue:  make(chan *Item, 100),
		Logger: log,
		Client: &http.Client{},
		GC:     gc,
		DryRun: false,
	}
}

func (d *Downloader) Push(req *http.Request, filepath string) error {
	it := &Item{
		Req:      req,
		FilePath: filepath,
	}
	select {
	case d.Queue <- it:
		return nil
	default:
		return fmt.Errorf("buffer is full")
	}
}

func (d *Downloader) Pop(wait bool) (*Item, error) {
	if wait {
		return <-d.Queue, nil
	}

	select {
	case it := <-d.Queue:
		return it, nil
	default:
		return nil, fmt.Errorf("empty queue")
	}
}

func (d *Downloader) download(item *Item) error {

	tmpFilePath := fmt.Sprintf("/tmp/%s", filepath.Base(item.FilePath))
	resp, err := d.Client.Do(item.Req)
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non 200 status code while trying to download %s", item.Req.URL.String())
	}

	file, err := os.Create(tmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to create new empty temporary file: %v", err)
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to memory copy into file: %v", err)
	}

	err = os.Rename(tmpFilePath, item.FilePath)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file: %v", err)
	}

	if resp.Header.Get("content-length") != fmt.Sprintf("%d", size) {
		return fmt.Errorf("size mismatch, wanted %s actual %s", resp.Header.Get("content-length"), fmt.Sprintf("%d", size))
	}

	return err
}

func (d *Downloader) Run() {
	for {
		if killswitch.Trigger {
			d.Logger.Warningln("kill switch enabled, unable to download new files")
		} else {
			lastItem, _ := d.Pop(true)
			err := d.download(lastItem)
			if err != nil {
				d.Logger.Errorf("failed to download item %s with error: %v", lastItem.FilePath, err)
				if _, statErr := os.Stat(lastItem.FilePath); statErr == nil {
					d.Logger.Infof("removing file %s", lastItem.FilePath)
					err = os.Remove(lastItem.FilePath)
					if err != nil {
						d.Logger.Errorf("failed to delete corrupt file %s with error %v", lastItem.FilePath, err)
					}
				}
				//TODO: should notify scheduler that the peer didn't serve the file properly
			} else {
				d.Logger.Infof("cached %s in %s", lastItem.Req.URL.String(), lastItem.FilePath)
			}
		}

		if d.DryRun {
			d.Logger.Infoln("dry run for testing purposes")
			return
		}
	}
}
