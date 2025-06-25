package micro_strategy

import (
	"fmt"
	"freefi/strategymgr/common"
	"strings"
)

const (
	MACD            = "macd"
	RSI             = "rsi"
	KDJ             = "kdj"
	BBANDS          = "bbands"
	BOLL            = "boll"
	VOLATI          = "volatility"
	STOCHRSI        = "stochrsi"
	PEAK_AND_VALLEY = "peak_and_valley"
	CUR2MAX         = "cur2max"
)

func Execute(data []common.KLine, paramsParams MicroStrategyParams) (msRet *MicroStrategyRet, err error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
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
	case BBANDS:
		msRet, err = ExecuteBBands(data, paramsParams)
	case PEAK_AND_VALLEY:
		msRet, err = ExecutePeakAndValley(data, paramsParams)
	case CUR2MAX:
		msRet, err = ExecuteCur2Max(data, paramsParams)
	default:
		err = fmt.Errorf("invalid strategy name: %s", paramsParams.Name)
		return
	}
	msRet.Params = &paramsParams
	return
}
