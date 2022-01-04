package utils

import "math"

func GetUint64withDefault(i *uint64, value uint64) uint64 {
	if i != nil {
		return *i
	}

	return value
}

func GetInt64withDefault(i *int64, value int64) int64 {
	if i != nil {
		return *i
	}

	return value
}

func GetUint32withDefault(i *uint32, value uint32) uint32 {
	if i != nil {
		return *i
	}

	return value
}

func GetInt32withDefault(i *int32, value int32) int32 {
	if i != nil {
		return *i
	}

	return value
}

func GetUint64FromMaps(key string, maps map[string]uint64) uint64 {
	if v, ok := maps[key]; ok {
		return v
	}

	return 0
}

const float64EqualityThreshold = 1e-9

func AlmostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}
