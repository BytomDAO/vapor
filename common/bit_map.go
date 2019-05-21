package common

import (
	"errors"
)

const bitLen = 32

var (
	errIndexOutOfBounds = errors.New("index out of bounds error")
)

type BitMap struct {
	size uint64
	arr []int32
}

func NewBitMap(size uint64) *BitMap {
	obj := &BitMap{size: size}
	num := (size + bitLen - 1) / bitLen
	arr := make([]int32, num)
	obj.arr = arr
	return obj
}

func (b *BitMap) Set(index uint64) error {
	if index >= b.size {
		return errIndexOutOfBounds
	}

	arrIndex, bitIndex := index / bitLen, index % bitLen
	b.arr[arrIndex] |= (1 << bitIndex)
	return nil
}

func (b *BitMap) Clean(index uint64) error {
	if index >= b.size {
		return errIndexOutOfBounds
	}

	arrIndex, bitIndex := index / bitLen, index % bitLen
	b.arr[arrIndex] &= (^(1 << bitIndex))
	return nil
}

func (b *BitMap) Test(index uint64) (bool, error) {
	if index >= b.size {
		return false, errIndexOutOfBounds
	}

	arrIndex, bitIndex := index / bitLen, index % bitLen
	return b.arr[arrIndex] & (1 << bitIndex) != 0, nil
}
