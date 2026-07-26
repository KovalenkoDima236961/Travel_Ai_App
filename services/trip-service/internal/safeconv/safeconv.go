package safeconv

import "fmt"

const (
	maxInt32 = int64(1<<31 - 1)
	minInt32 = int64(-1 << 31)
)

func IntToInt32(value int, name string) (int32, error) {
	return Int64ToInt32(int64(value), name)
}

func Int64ToInt32(value int64, name string) (int32, error) {
	if value < minInt32 || value > maxInt32 {
		return 0, fmt.Errorf("%s must fit in int32", name)
	}
	return int32(value), nil
}

func NonNegativeIntToUint64(value int, name string) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s must be non-negative", name)
	}
	return uint64(value), nil
}
