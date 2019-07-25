package log

import (
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"

	"github.com/vapor/config"
)

const (
	GENERAL_LOG="general"
)
var defaultFormatter = &logrus.TextFormatter{DisableColors: true}
var BtmLog = &logrus.Logger{}
var ModuleLogger map[string]io.Writer

func init() {
	BtmLog = logrus.New()
}

func InitLogFile(config *config.Config) {
	ModuleLogger = map[string]io.Writer{
		"account":   newModuleWriter(config, "account"),
		GENERAL_LOG:   newModuleWriter(config, GENERAL_LOG),
		"txbuilder": newModuleWriter(config, "txbuilder"),
		"leveldb":   newModuleWriter(config, "leveldb"),
		"node":      newModuleWriter(config, "node"),
	}
	hook := newBtmHook(&logrus.JSONFormatter{})
	BtmLog.Hooks.Add(hook)
}

func newModuleWriter(config *config.Config, module string) io.Writer {
	var DefaultMaxAge int64=360  // 15 天
	var DefaultRotationTime int64=168  // 7天
	logPath := filepath.Join(config.LogDir(), module)
	if module=="leveldb" {
		DefaultRotationTime=24//levelModule产生的log很大，一天分一次，其它的7天分一次
	}
	writer, _ := rotatelogs.New(
		logPath+".%y%m%d",
		//rotatelogs.WithLinkName(module),
		rotatelogs.WithMaxAge(time.Duration(DefaultMaxAge)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(DefaultRotationTime)*time.Hour),
	)
	return writer
}

type BtmHook struct {
	lock      *sync.Mutex
	formatter logrus.Formatter
}

func newBtmHook(formatter logrus.Formatter) *BtmHook {
	hook := &BtmHook{lock: new(sync.Mutex)}
	hook.setFormatter(formatter)
	return hook
}

func (hook *BtmHook) setFormatter(formatter logrus.Formatter) {
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
	module := entry.Data["module"]
	if module==nil{
		writer  = ModuleLogger[GENERAL_LOG]
	}else {
		writer  = ModuleLogger[module.(string)]
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
