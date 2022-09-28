package gc

import (
	"time"

	"github.com/sirupsen/logrus"
)

type GC struct {
	DataDir    string
	Interval   time.Duration
	MaxAge     time.Duration
	MaxSize    int
	MinStorage int
	Logger     *logrus.Entry
}

func NewGC(dataDir string, interval, maxAge int, log *logrus.Entry) *GC {
	return &GC{
		DataDir:  dataDir,
		Interval: time.Duration(interval),
		MaxAge:   time.Duration(maxAge),
		Logger:   log,
	}
}

func (gc *GC) Run() error {

	for {
		//atim can't be used here
		//
		time.Sleep(time.Duration(3) * time.Second)
	}
}
