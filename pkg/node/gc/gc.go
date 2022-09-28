package gc

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type GarbageCollector struct {
	DataDir  string
	Interval time.Duration
	MaxAge   time.Duration
	Logger   *logrus.Entry
}

func NewGC(dataDir string, interval, maxAge int) *GarbageCollector {
	return &GarbageCollector{
		DataDir:  dataDir,
		Interval: time.Duration(interval),
		MaxAge:   time.Duration(maxAge),
	}
}

func (gc *GarbageCollector) Run() {
	for {
		fmt.Println("TODO: implement GC")
	}
}
