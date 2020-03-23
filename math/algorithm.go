package math

// MinUint64 return the min of x and y
func MinUint64(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}
