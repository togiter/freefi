package datamgr

import (
	"freefi/strategymgr/common"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/utils"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
)

func ToBNPeroid(kPeriod int) string {
	switch kPeriod {
	case 1:
		return "1m"
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 120:
		return "2h"
	case 240:
		return "4h"
	case 360:
		return "6h"
	case 480:
		return "8h"
	case 720:
		return "12h"
	case 1440:
		return "1d"
	case 1440 * 7:
		return "1w"
	default:
		logger.Warnf("ToBNPeroid unknown kPeriod: %d", kPeriod)
		return "1d"
	}
}

func ToKLines(klines interface{}) []common.KLine {
	switch v := klines.(type) {
	case []*futures.Kline:
		ks := make([]common.KLine, len(v))
		for i, k := range v {
			ks[i] = ToKLine(k)
		}		
		return ks
	case []*delivery.Kline:				
		ks := make([]common.KLine, len(v))												
		for i, k := range v {															
			ks[i] = ToKLine(k)															
		}													
		return ks											
	case []*binance.Kline:									
		ks := make([]common.KLine, len(v))												
		for i, k := range v {															
			ks[i] = ToKLine(k)															
		}													
		return ks											
	}
	return []common.KLine{}
}

func ToKLine(kline interface{}) common.KLine {
	switch kline.(type) {
	case *futures.Kline:
		k := kline.(*futures.Kline)
		return common.KLine{
			Open:                  utils.ToFloat64(k.Open),
			High:                  utils.ToFloat64(k.High),
			Low:                   utils.ToFloat64(k.Low),
			Close:                 utils.ToFloat64(k.Close),
			Volume:                utils.ToFloat64(k.Volume),
			CloseTime:             k.CloseTime,
			OpenTime:              k.OpenTime,
			TradeNum:              utils.ToInt64(k.TradeNum),
			TakerBuyBaseAssetVol:  utils.ToFloat64(k.TakerBuyBaseAssetVolume),
			TakerBuyQuoteAssetVol: utils.ToFloat64(k.TakerBuyQuoteAssetVolume),
			QuoteAssetVol:         utils.ToFloat64(k.QuoteAssetVolume),
		}
	case *delivery.Kline:
		k := kline.(*delivery.Kline)
		return common.KLine{
			Open:                  utils.ToFloat64(k.Open),
			High:                  utils.ToFloat64(k.High),
			Low:                   utils.ToFloat64(k.Low),
			Close:                 utils.ToFloat64(k.Close),
			Volume:                utils.ToFloat64(k.Volume),
			CloseTime:             k.CloseTime,
			OpenTime:              k.OpenTime,
			TradeNum:              utils.ToInt64(k.TradeNum),
			TakerBuyBaseAssetVol:  utils.ToFloat64(k.TakerBuyBaseAssetVolume),
			TakerBuyQuoteAssetVol: utils.ToFloat64(k.TakerBuyQuoteAssetVolume),
			QuoteAssetVol:         utils.ToFloat64(k.QuoteAssetVolume),
		}
	case *binance.Kline:
		k := kline.(*binance.Kline)
		return common.KLine{
			Open:                  utils.ToFloat64(k.Open),
			High:                  utils.ToFloat64(k.High),
			Low:                   utils.ToFloat64(k.Low),
			Close:                 utils.ToFloat64(k.Close),
			Volume:                utils.ToFloat64(k.Volume),
			CloseTime:             k.CloseTime,
			OpenTime:              k.OpenTime,
			TradeNum:              utils.ToInt64(k.TradeNum),
			TakerBuyBaseAssetVol:  utils.ToFloat64(k.TakerBuyBaseAssetVolume),
			TakerBuyQuoteAssetVol: utils.ToFloat64(k.TakerBuyQuoteAssetVolume),
			QuoteAssetVol:         utils.ToFloat64(k.QuoteAssetVolume),
		}
	}
	logger.Warnf("ToKLine unknown kline type: %v", kline)
	return common.KLine{}
}
