package common

const bitLen = 32

type BitMap struct {
	num int
	arr []uint
}

func NewBitMap(maxNum int) *BitMap {
	obj := &BitMap{}
	num := (maxNum + bitLen - 1) / bitLen
	arr := make([]uint, num)
	obj.num, obj.arr = num, arr
	return obj
}

func (b *BitMap) Set(num uint) {
	arrIndex, bitIndex := b.arrIndex(num), b.bitIndex(num)
	elem := b.arr[arrIndex]
	b.arr[arrIndex] = elem | (1 << bitIndex)
}

func (b *BitMap) Clean(num uint) {
	arrIndex, bitIndex := b.arrIndex(num), b.bitIndex(num)
	elem := b.arr[arrIndex]
	b.arr[arrIndex] = elem & (^(1 << bitIndex))
}

func (b *BitMap) Test(num uint) bool {
	arrIndex, bitIndex := b.arrIndex(num), b.bitIndex(num)
	elem := b.arr[arrIndex]
	return (elem & (1 << bitIndex)) != 0
}

func (b *BitMap) arrIndex(num uint) uint {
	return num / bitLen
}

func (b *BitMap) bitIndex(num uint) uint {
	return num % bitLen
}
