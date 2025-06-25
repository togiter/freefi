package micro_strategy

import "freefi/strategymgr/common"

const (
	InClose = 0
	InHigh  = 1
	InLow   = 2
	InOpen  = 3
	InVol   = 4
)

func getPrices(klines []common.KLine, priceType int) []float64 {
	prices := make([]float64, 0, len(klines))
	for i := range klines {
		price := klines[i].Close
		switch priceType {
		case InClose:
			price = klines[i].Close
		case InHigh:
			price = klines[i].High
		case InLow:
			price = klines[i].Low
		case InOpen:
			price = klines[i].Open
		case InVol:
			price = klines[i].Volume
		}
		prices = append(prices, price)
	}
	return prices
}
