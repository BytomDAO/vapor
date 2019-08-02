package api

type server struct {
}

func NewApiServer() {
	return &server{}
}

func (s *server) Run() {
}
