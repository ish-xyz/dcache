package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

type Downloader struct {
	mu         sync.Mutex
	Queue      []*Item
	KillSwitch bool
	Client     *http.Client
	Logger     *logrus.Entry
}

type Item struct {
	Req      *http.Request
	FilePath string
}

func NewDownloader(log *logrus.Entry) *Downloader {

	return &Downloader{
		Queue:      []*Item{},
		KillSwitch: false,
		Logger:     log,
		Client:     &http.Client{},
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

func (d *Downloader) Watch() error {
	for {
		if d.KillSwitch {
			break
		}

		if len(d.Queue) > 0 {
			lastItem, err := d.Pop()
			if err != nil {
				d.Logger.Errorln("failed to take last item of the queue")
				continue
			}
			d.download(lastItem)
		}
	}
	return nil
}

func (d *Downloader) download(item *Item) error {

	file, err := os.Create(item.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := d.Client.Do(item.Req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	if resp.Header.Get("content-length") != fmt.Sprintf("%d", size) {
		d.Logger.Errorln("content-length and file size don't match, deleting file")
		err = os.Remove(item.FilePath)
		if err != nil {
			d.Logger.Errorln("failed to delete file:", err)
		}
		return err
	}

	return err
}
