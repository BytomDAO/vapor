package synchron

type Porter struct{}

func NewPorter() *Porter {
	return &Porter{}
}

func (p *Porter) Run() {}
