package synchron

import (
	"reflect"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type Porter struct {
	MainFuncs [](func() error)
}

func NewPorter() *Porter {
	return &Porter{}
}

func (p *Porter) Run() {
	for _, f := range p.MainFuncs {
		log.Info(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
	}
}
