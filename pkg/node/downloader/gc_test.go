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

func TestDirSizeTooBig() {}

func TestDirSizeOK() {}
