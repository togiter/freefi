package utils

import "github.com/spf13/cast"

func ToInt64(value interface{}) int64 {
	return cast.ToInt64(value)
}

func ToFloat64(value interface{}) float64 {
	return cast.ToFloat64(value)
}
