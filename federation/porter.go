package federation

import (
	"reflect"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type Porter struct {
	Callbacks [](func() error)
}

func NewPorter() *Porter {
	return &Porter{}
}

func (p *Porter) AttachCallback(f func() error) {
	p.Callbacks = append(p.Callbacks, f)
}

func (p *Porter) Run() {
	for _, f := range p.Callbacks {
		funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		log.Info("Running func: ", funcName)
		f()
		log.Info("Func done: ", funcName)
	}
}
