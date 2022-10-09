package downloader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type FilesCache struct {
	AtimeStore map[string]int64
	FilesByAge []string
	FilesSize  map[string]int64
}

type GC struct {
	Interval     time.Duration `validate:"required"`
	MaxAtimeAge  time.Duration `validate:"required"`
	MaxDiskUsage int           `validate:"required"`
	DataDir      string        `validate:"required"`
	Cache        *FilesCache   `validate:"required"`
	Logger       *logrus.Entry `validate:"required"`
	DryRun       bool
}

func (gc *GC) UpdateAtime(item string) {
	gc.Cache.FilesByAge = append(gc.Cache.FilesByAge, item)
	if gc.Cache.FilesByAge[0] == item {
		gc.Cache.FilesByAge = gc.Cache.FilesByAge[1:]
	}
	ts := time.Now().Unix()
	gc.Cache.AtimeStore[item] = ts
}

func (gc *GC) dataDirSize() float64 {
	var dirSize int64 = 0

	readSize := func(path string, file os.FileInfo, err error) error {
		// cache file sizes into a map so that
		// we avoid to read all the time from disk
		if size, ok := gc.Cache.FilesSize[file.Name()]; ok && !file.IsDir() {
			dirSize += size
			return nil
		}

		if !file.IsDir() {
			gc.Cache.FilesSize[file.Name()] = file.Size()
			dirSize += file.Size()
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
			if _, ok := gc.Cache.AtimeStore[fi.Name()]; !ok {
				gc.Logger.Debugln("no entry in atimestore for item", fi.Name())
				continue
			}

			fileAtimeAge := time.Now().Unix() - gc.Cache.AtimeStore[fi.Name()]
			if fileAtimeAge > int64(gc.MaxAtimeAge.Seconds()) {
				gc.Logger.Debugln("deleting file:", fi.Name())
				filepath := fmt.Sprintf("%s/%s", gc.DataDir, fi.Name())
				err := os.Remove(filepath)
				if err != nil {
					gc.Logger.Errorf("failed to remove file %s, error: %v", filepath, err)
					continue
				}
				delete(gc.Cache.AtimeStore, fi.Name())
				continue
			}
		}

		killswitch.mu.Lock()

		if gc.dataDirSize() > float64(gc.MaxDiskUsage) {
			gc.Logger.Debugln("enabling downloader killswitch as we reached the maximum disk space")
			killswitch.Trigger = true
			gc.cleanDataDir()
		} else if killswitch.Trigger {
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

	for _, file := range gc.Cache.FilesByAge {
		err := os.Remove(fmt.Sprintf("%s/%s", gc.DataDir, file))
		if err != nil {
			gc.Logger.Errorf("failed to remove file %s", file)
			continue
		}

		gc.Cache.FilesByAge = gc.Cache.FilesByAge[1:]
		if gc.dataDirSize() < float64(gc.MaxDiskUsage) {
			break
		}
	}
	return nil
}
