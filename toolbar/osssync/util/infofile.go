package util

import (
	"io"
	"io/ioutil"
)

// Interval determines the number of blocks in a Gzip file in the Interval of blockHeight
// StartBlockHeight is the start of the Interval
// EndBlockHeight: the end of the Interval
// GzSize is the number of blocks store in a Gzip file
type Interval struct {
	StartBlockHeight uint64
	EndBlockHeight   uint64
	GzSize           uint64
}

// NewInterval creates a new Interval from info.json
func NewInterval(start, end, gzSize uint64) *Interval {
	return &Interval{
		StartBlockHeight: start,
		EndBlockHeight:   end,
		GzSize:           gzSize,
	}
}

// Info is a struct for info.json
type Info struct {
	LatestBlockHeight uint64
	Interval          []*Interval
}

// NewInfo creates a new Info for info.json
func NewInfo(end, gzSize uint64) *Info {
	newInvl := NewInterval(1, end, gzSize)
	var arr []*Interval
	arr = append(arr, newInvl)
	return &Info{0, arr}
}

// GetInfoJson from stream
func GetInfoJson(body io.ReadCloser) (*Info, error) {
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	info := new(Info)
	return info, Json2Struct(data, &info)
}
