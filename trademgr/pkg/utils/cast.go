package utils

import (
	"math"

	"github.com/spf13/cast"
)

func ToInt64(value interface{}) int64 {
	return cast.ToInt64(value)
}

func ToFloat64(value interface{}) float64 {
	return cast.ToFloat64(value)
}

func ToPrecision(numAny interface{}, precision int) float64 {
	num := cast.ToFloat64(numAny)
	factor := math.Pow(10, float64(precision))
	return math.Round(num*factor) / factor
}

func ToFixed(numAny interface{}, precision int) string {
	num := cast.ToFloat64(numAny)
	factor := math.Pow(10, float64(precision))
	return cast.ToString(math.Round(num*factor) / factor)
}

func ToFixedFloor(numAny interface{}, precision int) string {
	num := cast.ToFloat64(numAny)
	factor := math.Pow(10, float64(precision))

	return cast.ToString(math.Floor(num*factor) / factor)
}

func ToFixedCeil(numAny interface{}, precision int) string {
	num := cast.ToFloat64(numAny)
	factor := math.Pow(10, float64(precision))
	return cast.ToString(math.Ceil(num*factor) / factor)

}
