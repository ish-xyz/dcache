package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type Downloader struct {
	Queue      chan *Item
	KillSwitch bool
	Client     *http.Client
	Logger     *logrus.Entry
	GC         *GC
}

type Item struct {
	Req      *http.Request
	FilePath string
}

func NewDownloader(log *logrus.Entry, dataDir string, maxAtime, interval time.Duration, maxDiskUsage, minDiskFree int) *Downloader {

	gc := &GC{
		MaxAtimeAge:  maxAtime,
		MaxDiskUsage: maxDiskUsage,
		MinDiskFree:  minDiskFree,
		Interval:     interval,
		DataDir:      dataDir,
		Logger:       log.WithField("component", "node.downloader.gc"),
		AtimeStore:   make(map[string]int64),
	}

	return &Downloader{
		Queue:      make(chan *Item),
		KillSwitch: false,
		Logger:     log,
		Client:     &http.Client{},
		GC:         gc,
	}
}

func (d *Downloader) Push(req *http.Request, filepath string) {

	item := &Item{
		Req:      req,
		FilePath: filepath,
	}
	d.Queue <- item
}

func (d *Downloader) Pop() *Item {
	return <-d.Queue
}

func (d *Downloader) download(item *Item) error {

	resp, err := d.Client.Do(item.Req)
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	file, err := os.Create(item.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)

	if resp.Header.Get("content-length") != fmt.Sprintf("%d", size) {
		return fmt.Errorf("size mismatch, wanted %s actual %s", resp.Header.Get("content-length"), fmt.Sprintf("%d", size))
	}

	return err
}

func (d *Downloader) Run() error {
	for {
		if d.KillSwitch {
			break
		}

		lastItem := d.Pop()
		err := d.download(lastItem)
		if err != nil {
			d.Logger.Errorf("failed to download item %s with error: %v", lastItem.FilePath, err)
			d.Logger.Infof("removing file %s", lastItem.FilePath)
			err = os.Remove(lastItem.FilePath)
			if err != nil {
				return fmt.Errorf("failed to delete corrupt file %s with error %v", lastItem.FilePath, err)
			}
			continue
		}
		d.Logger.Infof("cached %s in %s", lastItem.Req.URL.String(), lastItem.FilePath)
	}
	return nil
}
