package log

import (
	"path/filepath"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"

	"github.com/vapor/config"
)

const (

	rotationTime int64 = 86400
	maxAge       int64 = 604800
)

var defaultFormatter = &logrus.TextFormatter{DisableColors: true}


func InitLogFile(config *config.Config) {
	hook := newBtmHook(config.LogFile)
	logrus.AddHook(hook)
}

type BtmHook struct {
	logPath string
	lock    *sync.Mutex
}

func newBtmHook(logPath string) *BtmHook {
	hook := &BtmHook{lock: new(sync.Mutex)}
	hook.logPath = logPath
	return hook
}

// Write a log line to an io.Writer.
func (hook *BtmHook) ioWrite(entry *logrus.Entry) error {
	module := "general"
	if data, ok := entry.Data["module"]; ok {
		module = data.(string)
	}

	logPath := filepath.Join(hook.logPath, module)

	writer, err := rotatelogs.New(
		logPath+".%Y%m%d",
		rotatelogs.WithMaxAge(time.Duration(maxAge)*time.Second),
		rotatelogs.WithRotationTime(time.Duration(rotationTime)*time.Second),
	)
	if err != nil {
		return err
	}

	msg, err := defaultFormatter.Format(entry)
	if err != nil {
		return err
	}

	_, err = writer.Write(msg)

	return err
}

func (hook *BtmHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	return hook.ioWrite(entry)
}

// Levels returns configured log levels.
func (hook *BtmHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
