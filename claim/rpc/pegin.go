package rpc

type PeginRpc interface {
	GetPeginAddress() (interface{}, error)
	GetPeginContractAddress() (interface{}, error)
}
