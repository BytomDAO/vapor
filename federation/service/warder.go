package service

type Warder struct {
	hostPort string
}

func NewWarder(hostPort string) *Warder {
	return &Warder{hostPort: hostPort}
}
