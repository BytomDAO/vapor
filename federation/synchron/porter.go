package synchron

import (
	"reflect"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type Porter struct {
	Callbacks [](func(p *Porter) error)
}

func NewPorter() *Porter {
	return &Porter{}
}

func (p *Porter) AttachCallback(f func(p *Porter) error) {
	p.Callbacks = append(p.Callbacks, f)
}

func (p *Porter) Run() {
	for _, f := range p.Callbacks {
		log.Info("Running...", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
		f(p)
		log.Info(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), " done.")
	}
}
