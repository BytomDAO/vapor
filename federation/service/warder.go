package service

type Warder struct {
	ip string
}

func NewWarder(ip string) *Warder {
	return &Warder{ip: ip}
}
