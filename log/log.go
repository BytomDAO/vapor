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
	GENERAL_LOG="general"
)
var defaultFormatter = &logrus.TextFormatter{DisableColors: true}
var defaultMaxAge int64=360  // 15 天
var defaultRotationTime int64=168  // 7天
var BtmLog = &logrus.Logger{}


type logModulefunc func() (io.Writer,error)

func init() {
	BtmLog = logrus.New()
}


func InitLogFile(config *config.Config) {
	hook := newBtmHook(config,&logrus.JSONFormatter{})
	BtmLog.Hooks.Add(hook)
}


func newModuleWriter(logPath,module string) logModulefunc {
	if module=="leveldb" {
		defaultRotationTime=24//levelModule产生的log很大，一天分一次，其它的7天分一次
	}
	return func() (io.Writer, error) {
			return rotatelogs.New(
				logPath+".%Y%m%d",
				//rotatelogs.WithLinkName(module),
				rotatelogs.WithMaxAge(time.Duration(defaultMaxAge)*time.Hour),
				rotatelogs.WithRotationTime(time.Duration(defaultRotationTime)*time.Hour),
			)
	}
}


type BtmHook struct {
	logPath string
	lock      *sync.Mutex
	formatter logrus.Formatter
}

func newBtmHook(config *config.Config,formatter logrus.Formatter) *BtmHook {
	hook := &BtmHook{lock: new(sync.Mutex)}
	hook.setFormatter(formatter)
	hook.logPath=config.LogDir()
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


// Write a log line to an io.Writer.
func (hook *BtmHook) ioWrite(entry *logrus.Entry) error {
	var (
		writer io.Writer
		msg    []byte
		err    error
	)
	module:=""
	moduleInterface := entry.Data["module"]
	if moduleInterface==nil{
		module=GENERAL_LOG
	}else {
		module=moduleInterface.(string)
	}

	logPath:=filepath.Join(hook.logPath,module)

	writer,err=newModuleWriter(logPath,module)()
	if err!=nil{
		return err
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



func (hook *BtmHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	return hook.ioWrite(entry)
}

// Levels returns configured log levels.
func (hook *BtmHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
