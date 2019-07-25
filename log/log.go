package log

import (
	"errors"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"github.com/vapor/config"
	"io"
	"path/filepath"
	"sync"
	"time"
)

var defaultFormatter = &logrus.TextFormatter{DisableColors: true}
var BtmLog = &logrus.Logger{}
var ModuleLogger map[string]io.Writer
var (
	accountWriter   io.Writer
	txbuilderWriter io.Writer
	levelWriter     io.Writer
)

func init() {
	BtmLog = logrus.New()

}

func InitLogFile(config *config.Config) {
	logPath := config.LogDir()
	ModuleLogger = map[string]io.Writer{
		"account":   newModuleWriter(logPath, "account"),
		"txbuilder": newModuleWriter(logPath, "txbuilder"),
		"leveldb":   newModuleWriter(logPath, "leveldb"),
	}
	hook := NewBtmHook(&logrus.JSONFormatter{})
	BtmLog.Hooks.Add(hook)
}

func newModuleWriter(path, module string) io.Writer {
	logPath := filepath.Join(path, module)
	writer, _ := rotatelogs.New(
		logPath+".%Y%m%d%H%M",
		//rotatelogs.WithLinkName(module),
		rotatelogs.WithMaxAge(time.Duration(86400)*time.Second),     //配置文件
		rotatelogs.WithRotationTime(time.Duration(86400)*time.Second), //配置文件
	)
	return writer
}

type BtmHook struct {
	lock      *sync.Mutex
	formatter logrus.Formatter
}

func NewBtmHook(formatter logrus.Formatter) *BtmHook {
	hook := &BtmHook{lock: new(sync.Mutex)}
	hook.SetFormatter(formatter)
	return hook
}

func (hook *BtmHook) SetFormatter(formatter logrus.Formatter) {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	if formatter == nil {
		formatter = defaultFormatter
	} else {
		switch formatter.(type) {
		case *logrus.TextFormatter:
			textFormatter := formatter.(*logrus.TextFormatter)
			textFormatter.DisableColors = true
		}
	}
	hook.formatter = formatter
}

func (hook *BtmHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	return hook.ioWrite(entry)
}

// Write a log line to an io.Writer.
func (hook *BtmHook) ioWrite(entry *logrus.Entry) error {
	var (
		writer io.Writer
		msg    []byte
		err    error
	)
	module := entry.Data["module"].(string)
	writer, ok := ModuleLogger[module]
	if !ok {
		return errors.New("incorrect module")
	}
	// use our formatter instead of entry.String()
	msg, err = hook.formatter.Format(entry)
	if err != nil {
		logrus.Println("failed to generate string for entry:", err)
		return err
	}
	_, err = writer.Write(msg)
	return err
}

// Levels returns configured log levels.
func (hook *BtmHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
