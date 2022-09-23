package downloader

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Downloader struct {
	mu            sync.Mutex
	Queue         []*Item
	EmergencyStop bool
}

type Item struct {
	Path string
	URL  string
}

func NewDownloader() *Downloader {

	return &Downloader{
		Queue:         []*Item{},
		EmergencyStop: false,
	}
}

func (d *Downloader) Push(url, path string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	item := &Item{
		Path: path,
		URL:  url,
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

func (d *Downloader) Watch(log *logrus.Entry) error {
	for {
		if d.EmergencyStop {
			break
		}

		if len(d.Queue) > 0 {
			lastItem, _ := d.Pop()
			log.Infoln("processing item", lastItem)
			fmt.Println(lastItem)
		}
	}
	return nil
}
