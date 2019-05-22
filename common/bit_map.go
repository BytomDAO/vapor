package common

import (
	"errors"
)

const bitLen = 32

var (
	errIndexOutOfBounds = errors.New("index out of bounds error")
)

type BitMap struct {
	size uint32
	arr []int32
}

func NewBitMap(size uint32) *BitMap {
	obj := &BitMap{size: size}
	num := (size + bitLen - 1) / bitLen
	arr := make([]int32, num)
	obj.arr = arr
	return obj
}

func (b *BitMap) Set(index uint32) error {
	if index >= b.size {
		return errIndexOutOfBounds
	}

	arrIndex, bitIndex := index / bitLen, index % bitLen
	b.arr[arrIndex] |= (1 << bitIndex)
	return nil
}

func (b *BitMap) Clean(index uint32) error {
	if index >= b.size {
		return errIndexOutOfBounds
	}

	arrIndex, bitIndex := index / bitLen, index % bitLen
	b.arr[arrIndex] &= (^(1 << bitIndex))
	return nil
}

func (b *BitMap) Test(index uint32) (bool, error) {
	if index >= b.size {
		return false, errIndexOutOfBounds
	}

	arrIndex, bitIndex := index / bitLen, index % bitLen
	return b.arr[arrIndex] & (1 << bitIndex) != 0, nil
}
