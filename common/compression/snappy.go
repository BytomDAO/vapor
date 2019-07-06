package compression

import (
	sny "github.com/golang/snappy"
)

func init() {
	creator := func() Compression {
		return NewSnappy()
	}

	registerCompressionCreator(SnappyBackendStr, creator, false)
}

type Snappy struct {
}

func NewSnappy() *Snappy {
	return &Snappy{}
}

func (s *Snappy) CompressBytes(data []byte) []byte {
	return sny.Encode(nil, data)
}

func (s *Snappy) DecompressBytes(data []byte) ([]byte, error) {
	return sny.Decode(nil, data)
}
