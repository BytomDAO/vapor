package log

import (
	"io"
	"path/filepath"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"

	"github.com/vapor/config"
)

const (
	ROTATION_TIME int64 = 86400
	MAX_AGE       int64 = 604800
	)

var defaultFormatter = &logrus.TextFormatter{DisableColors: true}

type logModulefunc func() (io.Writer, error)

func InitLogFile(config *config.Config) {
	hook := newBtmHook(config)
	logrus.AddHook(hook)
}

type BtmHook struct {
	logPath   string
	lock      *sync.Mutex
}

func newBtmHook(config *config.Config) *BtmHook {
	hook := &BtmHook{lock: new(sync.Mutex)}
	hook.logPath = config.LogDir()

	return hook
}


func newModuleWriter(logPath string) logModulefunc {
	return func() (io.Writer, error) {
		return rotatelogs.New(
			logPath+".%Y%m%d",
			rotatelogs.WithMaxAge(time.Duration(MAX_AGE)*time.Second),
			rotatelogs.WithRotationTime(time.Duration(ROTATION_TIME)*time.Second),
		)
	}
}

// Write a log line to an io.Writer.
func (hook *BtmHook) ioWrite(entry *logrus.Entry) error {
	module := ""
	moduleInterface, ok := entry.Data["module"]
	if !ok {
		module = "general"
	} else {
		module = moduleInterface.(string)
	}

	logPath := filepath.Join(hook.logPath, module)

	writer, err := newModuleWriter(logPath)()
	if err != nil {
		return err
	}
	msg, err:=defaultFormatter.Format(entry)
	if err != nil {
		logrus.Println("failed to generate string for entry:", err)
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
