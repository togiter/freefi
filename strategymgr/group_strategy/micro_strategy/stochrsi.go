package micro_strategy

import (
	"fmt"
	common "freefi/strategymgr/common"
	"freefi/strategymgr/pkg/logger"
	"math"
	"time"

	"github.com/markcheno/go-talib"
	"github.com/mitchellh/mapstructure"
)

type StochRSIParams struct {
	RSIPeriod      int     `json:"rsiPeriod"`
	StochRSIPeriod int     `json:"stochRsiPeriod"`
	SmoothKPeriod  int     `json:"smoothKPeriod"`
	SmoothDPeriod  int     `json:"smoothDPeriod"`
	OverBuyVal     float64 `json:"overBuyVal"`
	OverSellVal    float64 `json:"overSellVal"`
}

func stochRSICheck(params StochRSIParams) bool {
	return params.RSIPeriod > 0 && params.StochRSIPeriod > 0 && params.SmoothKPeriod > 0 && params.SmoothDPeriod > 0
}

func ExecuteStochRSI(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}
	tradeSide := common.TradeSideNone
	fomo := 0
	var sRsiParams StochRSIParams
	err = mapstructure.Decode(params.Params, &sRsiParams)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("masRsiParamscd params to struct Error:%v", err)
		return ret, err
	}

	if !stochRSICheck(sRsiParams) {
		return ret, fmt.Errorf("sRsiParams check Error")
	}
	prices := getPrices(klines, InClose)
	rsis := talib.Rsi(prices, sRsiParams.RSIPeriod)
	stochRSIs := stochRSI(rsis, sRsiParams.StochRSIPeriod)
	//平滑K
	maStochRsiK := talib.Sma(stochRSIs, sRsiParams.SmoothKPeriod)
	//平滑D
	maStochRsiD := talib.Sma(maStochRsiK, sRsiParams.SmoothDPeriod)

	lenSt := len(maStochRsiK)
	lenMaSt := len(maStochRsiD)
	if lenSt < 3 || lenMaSt < 3 {
		return ret, fmt.Errorf("klines len Error")
	}

	st_1, st_2, st_3 := maStochRsiK[lenSt-1]*100.0, maStochRsiK[lenSt-2]*100.0, maStochRsiK[lenSt-3]*100.0
	maSt_1, maSt_2, maSt_3 := maStochRsiD[lenMaSt-1]*100.0, maStochRsiD[lenMaSt-2]*100.0, maStochRsiD[lenMaSt-3]*100.0
	overBuyVal := math.Max(sRsiParams.OverBuyVal, 65.0)
	overSellVal := math.Min(sRsiParams.OverSellVal, 35.0)
	logger.Infof("stochRSI-K:%f, stochRSI-K:%f,stochRSI-D:%f, stochRSI-D:%f, overBuyVal:%f, overSellVal:%f", st_1, st_2, maSt_1, maSt_2, overBuyVal, overSellVal)
	mark := ""
	if maSt_1 < st_1 && maSt_2 >= st_2 {
		//金叉

		if maSt_2 <= overSellVal || st_2 <= overSellVal || params.Legacy {
			fomo = 1
			mark = "金叉"
			mark += "+超卖"
			tradeSide = common.TradeSideLong
		}

	} else if maSt_1 > st_1 && maSt_2 <= st_2 {
		//死叉
		mark = "死叉"

		if maSt_2 >= overBuyVal || st_2 >= overBuyVal || params.Legacy {
			fomo = 1
			mark += "+超卖"
			tradeSide = common.TradeSideShort
		}
	} else if !params.Legacy {
		if maSt_2 <= overSellVal || st_2 <= overSellVal {
			//金叉后+超卖状态下持续回升
			if maSt_1 > maSt_2 && maSt_2 < st_2 && maSt_3 >= st_3 {
				tradeSide = common.TradeSideLong
				mark = fmt.Sprintf("超卖+持续回升,ma-1-3(%.2f-%.2f-%.2f),st-1-3(%.2f-%.2f-%.2f)", maSt_1, maSt_2, maSt_3, st_1, st_2, st_3)

			}
		} else if maSt_2 >= overBuyVal || st_2 >= overBuyVal {
			//死叉后+超买状态下持续下跌
			if maSt_1 < maSt_2 && maSt_2 > st_2 && maSt_3 <= st_3 {
				tradeSide = common.TradeSideShort
				mark = fmt.Sprintf("超买+持续下跌,ma-1-3(%.2f-%.2f-%.2f),st-1-3(%.2f-%.2f-%.2f)", maSt_1, maSt_2, maSt_3, st_1, st_2, st_3)

			}
		}
	}
	ret.TradeSuggest.TradeSide = tradeSide
	ret.TradeSuggest.FomoLevel = fomo
	ret.TradeSuggest.Mark = mark
	ret.Opts = make(map[string]interface{})
	ret.Opts["dir"] = -1
	if maSt_1 < st_1 {
		ret.Opts["dir"] = 1
	}
	return ret, nil
}

func stochRSI(rsi []float64, stochRsiPeriod int) []float64 {
	// Step 2: 计算 RSI 的最高值和最低值
	highestRsi := talib.Max(rsi, stochRsiPeriod)
	lowestRsi := talib.Min(rsi, stochRsiPeriod)

	// Step 3: 计算 StochRSI
	stochRsi := make([]float64, len(rsi))
	for i := 0; i < len(rsi); i++ {
		if highestRsi[i] != 0 && (highestRsi[i]-lowestRsi[i]) != 0 {
			stochRsi[i] = (rsi[i] - lowestRsi[i]) / (highestRsi[i] - lowestRsi[i])
		} else {
			stochRsi[i] = 0 // 如果分母为 0，则结果为 0
		}
	}
	return stochRsi
}
