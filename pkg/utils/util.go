package utils

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
