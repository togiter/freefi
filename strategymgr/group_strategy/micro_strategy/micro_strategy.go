package micro_strategy

import (
	"fmt"
	"freefi/strategymgr/common"
	"strings"
)

const (
	MACD     = "macd"
	RSI      = "rsi"
	KDJ      = "kdj"
	BOLL     = "boll"
	VOLATI   = "volatility"
	STOCHRSI = "stochrsi"
)

func Execute(data []common.KLine, paramsParams MicroStrategyParams) (msRet *MicroStrategyRet, err error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("No data available")
	}
	defer func() {
		msRet.MakeFinalTrade()
		msRet.TradeSuggest.Price = data[len(data)-1].Close
	}()
	switch strings.ToLower(paramsParams.Name) {
	case MACD:
		msRet, err = ExecuteMACD(data, paramsParams)
	case RSI:
		msRet, err = ExecuteRSI(data, paramsParams)
	case KDJ:
		msRet, err = ExecuteKDJ(data, paramsParams)
	case BOLL:
		msRet, err = ExecuteBOLL(data, paramsParams)
	case VOLATI:
		msRet, err = ExecuteVolatility(data, paramsParams)
	case STOCHRSI:
		msRet, err = ExecuteStochRSI(data, paramsParams)
	default:
		err = fmt.Errorf("Invalid strategy name: %s", paramsParams.Name)
	}
	return
}
