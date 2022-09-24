package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Downloader struct {
	mu         sync.Mutex
	Queue      []*Item
	KillSwitch bool
	Client     *http.Client
	Logger     *logrus.Entry
	Interval   time.Duration
}

type Item struct {
	Req      *http.Request
	FilePath string
}

func NewDownloader(log *logrus.Entry, inte time.Duration) *Downloader {

	return &Downloader{
		Queue:      []*Item{},
		KillSwitch: false,
		Logger:     log,
		Client:     &http.Client{},
		Interval:   inte,
	}
}

func (d *Downloader) Push(req *http.Request, filepath string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	item := &Item{
		Req:      req,
		FilePath: filepath,
	}

	d.Queue = append(d.Queue, item)
}

func (d *Downloader) Pop() (*Item, error) {

	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.Queue) > 0 {
		item := d.Queue[0]
		d.Queue = d.Queue[1:]

		return item, nil
	}

	return nil, fmt.Errorf("empty queue")
}

func (d *Downloader) download(item *Item) error {

	file, err := os.Create(item.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := d.Client.Do(item.Req)
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	if resp.Header.Get("content-length") != fmt.Sprintf("%d", size) {
		return fmt.Errorf("size mismatch, wanted %s actual %s", resp.Header.Get("content-length"), fmt.Sprintf("%d", size))
	}

	return err
}

func (d *Downloader) Watch() error {
	for {
		if d.KillSwitch {
			break
		}
		if len(d.Queue) > 0 {

			lastItem, err := d.Pop()
			if err != nil {
				d.Logger.Errorln("failed to take last item from the queue")
				continue
			}

			err = d.download(lastItem)
			if err != nil {
				d.Logger.Errorf("failed to download item %s with error: %v", lastItem.FilePath, err)
				d.Logger.Infof("removing file %s", lastItem.FilePath)
				err = os.Remove(lastItem.FilePath)
				if err != nil {
					return fmt.Errorf("failed to delete corrupt file %s with error %v", lastItem.FilePath, err)
				}
			}
			d.Logger.Infof("cached %s in %s", lastItem.Req.URL.String(), lastItem.FilePath)
		}
		time.Sleep(d.Interval * time.Second)
	}
	return nil
}
