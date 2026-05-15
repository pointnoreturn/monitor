package libmetric

import "math"

func f64ToU64(f float64) uint64 {
	return math.Float64bits(f)
}

func u64ToF64(u uint64) float64 {
	return math.Float64frombits(u)
}
