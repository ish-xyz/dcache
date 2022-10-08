package downloader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

var fileSizes map[string]int64

type GC struct {
	Interval     time.Duration
	MaxAtimeAge  time.Duration
	MaxDiskUsage int
	DataDir      string
	AtimeStore   map[string]int64
	FilesByAge   []string
	Logger       *logrus.Entry
	DryRun       bool
}

func (gc *GC) UpdateAtime(item string) {
	gc.FilesByAge = append(gc.FilesByAge, item)
	if gc.FilesByAge[0] == item {
		gc.FilesByAge = gc.FilesByAge[1:]
	}
	ts := time.Now().Unix()
	gc.AtimeStore[item] = ts
}

func (gc *GC) dataDirSize() float64 {
	var dirSize int64 = 0

	readSize := func(path string, file os.FileInfo, err error) error {
		// cache file sizes into a map so that
		// we avoid to read all the time from disk
		if size, ok := fileSizes[file.Name()]; ok && !file.IsDir() {
			dirSize += size
		} else {
			if !file.IsDir() {
				fileSizes[file.Name()] = file.Size()
				dirSize += file.Size()
			}
		}
		return nil
	}

	filepath.Walk(gc.DataDir, readSize)

	return float64(dirSize)
}

func (gc *GC) Run() {
	for {
		files, err := ioutil.ReadDir(gc.DataDir)
		if err != nil {
			gc.Logger.Errorln("error while reading dataDir:", err)
		}

		for _, fi := range files {
			gc.Logger.Debugf("checking file %s", fi.Name())
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

		killswitch.mu.Lock()

		if gc.dataDirSize() > float64(gc.MaxDiskUsage) {
			gc.Logger.Debugln("enabling downloader killswitch as we reached the maximum disk space")
			killswitch.Trigger = true
			gc.cleanDataDir()
		} else {
			gc.Logger.Debugln("disabling downloader killswitch")
			killswitch.Trigger = false
		}

		killswitch.mu.Unlock()

		if gc.DryRun {
			return
		}

		time.Sleep(gc.Interval)
	}
}

func (gc *GC) cleanDataDir() error {

	for _, file := range gc.FilesByAge {
		err := os.Remove(fmt.Sprintf("%s/%s", gc.DataDir, file))
		if err != nil {
			gc.Logger.Errorf("failed to remvoe file %s", file)
		}
		if gc.dataDirSize() < float64(gc.MaxDiskUsage) {
			break
		}
	}
	return nil
}
