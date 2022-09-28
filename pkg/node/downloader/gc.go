package downloader

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type GC struct {
	Interval     time.Duration
	MaxAtimeAge  time.Duration
	MaxDiskUsage int
	MinDiskFree  int
	DataDir      string
	AtimeStore   map[string]int64
	Logger       *logrus.Entry
}

func (gc *GC) UpdateAtime(item string) {
	ts := time.Now().Unix()
	gc.AtimeStore[item] = ts
}

func (gc *GC) Run() {
	for {
		files, err := ioutil.ReadDir(gc.DataDir)
		if err != nil {
			gc.Logger.Errorln("error while reading dataDir:", err)
			continue
		}
		for _, fi := range files {
			gc.Logger.Debugln("checking file %s", fi.Name())
			if _, ok := gc.AtimeStore[fi.Name()]; !ok {
				gc.Logger.Warningf("can't find file %s on Atime memory store", fi.Name())
				continue
			}

			fileAtimeAge := time.Now().Unix() - gc.AtimeStore[fi.Name()]
			if fileAtimeAge > int64(gc.MaxAtimeAge.Seconds()) {
				gc.Logger.Debugln("deleting file:", fi.Name())
				filepath := fmt.Sprintf("%s/%s", gc.DataDir, fi.Name())
				err := os.Remove(filepath)
				if err == nil {
					delete(gc.AtimeStore, fi.Name())
				}
				gc.Logger.Errorf("failed to remove file %s, error: %v", filepath, err)
				continue
			}
			gc.Logger.Debugln("file is too young, keeping it ->", fi.Name())
		}
		time.Sleep(gc.Interval)
	}
}
