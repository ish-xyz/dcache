package downloader

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestFileTooOld(t *testing.T) {

	logger := logrus.New()
	dataDir := "/tmp/dcache-gc-test-01"
	fileName := "test.txt"
	gc := &GC{
		MaxAtimeAge:  time.Duration(10) * time.Second,
		MaxDiskUsage: 1024 * 1024,
		Interval:     time.Duration(10) * time.Second,
		DataDir:      dataDir,
		Logger:       logger.WithField("component", "node.downloader.gc"),
		AtimeStore:   make(map[string]int64),
		DryRun:       true,
	}

	mkdirErr := os.Mkdir(dataDir, 0755)
	_, createFileErr := os.Create(fmt.Sprintf("%s/%s", dataDir, fileName))

	gc.AtimeStore[fileName] = time.Now().Unix() - 11

	gc.Run()

	_, statErr := os.Stat(fmt.Sprintf("%s/%s", dataDir, fileName))

	assert.Equal(t, nil, mkdirErr)
	assert.Equal(t, nil, createFileErr)
	assert.NotEqual(t, nil, statErr)

	os.RemoveAll(dataDir)
}

func TestFileAgeOK(t *testing.T) {

	logger := logrus.New()
	dataDir := "/tmp/dcache-gc-test-02"
	fileName := "test.txt"
	gc := &GC{
		MaxAtimeAge:  time.Duration(10) * time.Second,
		MaxDiskUsage: 1024 * 1024,
		Interval:     time.Duration(10) * time.Second,
		DataDir:      dataDir,
		Logger:       logger.WithField("component", "node.downloader.gc"),
		AtimeStore:   make(map[string]int64),
		DryRun:       true,
	}

	mkdirErr := os.Mkdir(dataDir, 0755)
	_, createFileErr := os.Create(fmt.Sprintf("%s/%s", dataDir, fileName))

	gc.AtimeStore[fileName] = time.Now().Unix()

	gc.Run()

	_, statErr := os.Stat(fmt.Sprintf("%s/%s", dataDir, fileName))

	assert.Equal(t, nil, mkdirErr)
	assert.Equal(t, nil, createFileErr)
	assert.Equal(t, nil, statErr)

	os.RemoveAll(dataDir)
}

func TestAtimeTable(t *testing.T) {

	logger := logrus.New()
	dataDir := "/tmp/dcache-gc-test-02"
	fileName := "test.txt"
	filepath := fmt.Sprintf("%s/%s", dataDir, fileName)
	gc := &GC{
		MaxAtimeAge:  time.Duration(10) * time.Second,
		MaxDiskUsage: 1024 * 1024,
		Interval:     time.Duration(10) * time.Second,
		DataDir:      dataDir,
		Logger:       logger.WithField("component", "node.downloader.gc"),
		AtimeStore:   make(map[string]int64),
		DryRun:       true,
	}

	gc.UpdateAtime(filepath)
	time.Sleep(time.Duration(1) * time.Second)
	now := time.Now().Unix()

	assert.GreaterOrEqual(t, now, gc.AtimeStore[filepath])
}

func TestDirSizeOK(t *testing.T) {

	logger := logrus.New()
	dataDir := "/tmp/dcache-gc-test-02"
	fileName := "test.txt"
	filepath := fmt.Sprintf("%s/%s", dataDir, fileName)
	gc := &GC{
		MaxAtimeAge:  time.Duration(10) * time.Second,
		MaxDiskUsage: 1024 * 1024,
		Interval:     time.Duration(10) * time.Second,
		DataDir:      dataDir,
		Logger:       logger.WithField("component", "node.downloader.gc"),
		AtimeStore:   make(map[string]int64),
		DryRun:       true,
	}

	os.Mkdir(dataDir, 0755)
	createFileWithSize(filepath, 10*1024*1024)
	gc.UpdateAtime(fileName)

	gc.Run()

	assert.Equal(t, 1, 2)

	os.RemoveAll(dataDir)
}

func createFileWithSize(filepath string, size int) error {
	data := make([]byte, size)
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}
