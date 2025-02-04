package micro_strategy

import (
	"encoding/json"
	"fmt"
	common "freefi/strategymgr/common"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/net"
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
		Params: params,
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
		err = fmt.Errorf("masRsiParamscd params to struct Error:", err)
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

	st_1, st_2 := maStochRsiK[lenSt-1], maStochRsiK[lenSt-2]
	maSt_1, maSt_2 := maStochRsiD[lenMaSt-1], maStochRsiD[lenMaSt-2]
	overBuyVal := math.Min(sRsiParams.OverBuyVal, 25.0)
	overSellVal := math.Max(sRsiParams.OverSellVal, 75.0)
	logger.Infof("stochRSI-K:%f, stochRSI-K:%f,stochRSI-D:%f, stochRSI-D:%f, overBuyVal:%f, overSellVal:%f", st_1, st_2, maSt_1, maSt_2, overBuyVal, overSellVal)
	mark := ""
	if maSt_1 > st_1 && maSt_2 < st_2 {
		//金叉
		tradeSide = common.TradeSideBuy
		mark = "金叉"
		if maSt_2 < overSellVal || st_2 < overSellVal {
			fomo = 1
			mark += "+超卖"
		}

	} else if maSt_1 < st_1 && maSt_2 > st_2 {
		//死叉
		mark = "死叉"
		tradeSide = common.TradeSideSell
		if maSt_2 > overBuyVal || st_2 > overBuyVal {
			fomo = 1
			mark += "+超买"
		}
	}
	ret.TradeSuggest.TradeSide = tradeSide
	ret.TradeSuggest.FomoLevel = fomo
	ret.TradeSuggest.Mark = mark
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

// Calculate RSI (Relative Strength Index)
func calculateRSI(prices []float64, period int) []float64 {
	rsi := make([]float64, len(prices))
	gains := make([]float64, len(prices))
	losses := make([]float64, len(prices))

	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	avgGain := avg(gains[:period])
	avgLoss := avg(losses[:period])
	rsi[period-1] = 100 - 100/(1+avgGain/avgLoss)

	for i := period; i < len(prices); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
		rsi[i] = 100 - 100/(1+avgGain/avgLoss)
	}

	return rsi
}

// Calculate Stochastic RSI
func calculateStochRSI(rsi []float64, period int) []float64 {
	stochRSI := make([]float64, len(rsi))
	for i := period; i < len(rsi); i++ {
		minRSI := min(rsi[i-period : i])
		maxRSI := max(rsi[i-period : i])
		stochRSI[i] = (rsi[i] - minRSI) / (maxRSI - minRSI) * 100 // Scale to 0-100
	}
	return stochRSI
}

func calculateMAStochRSI(stochRSI []float64, period int) []float64 {
	maStochRSI := make([]float64, len(stochRSI))
	for i := period - 1; i < len(stochRSI); i++ {
		maStochRSI[i] = avg(stochRSI[i-period+1 : i+1])
	}
	return maStochRSI
}

// Calculate %K (smoothed Stochastic RSI)
func calculateSmoothK(stochRSI []float64, period int) []float64 {
	smoothK := make([]float64, len(stochRSI))
	for i := period - 1; i < len(stochRSI); i++ {
		smoothK[i] = avg(stochRSI[i-period+1 : i+1])
	}
	return smoothK
}

// Calculate %D (smoothed %K)
func calculateSmoothD(smoothK []float64, period int) []float64 {
	smoothD := make([]float64, len(smoothK))
	for i := period - 1; i < len(smoothK); i++ {
		smoothD[i] = avg(smoothK[i-period+1 : i+1])
	}
	return smoothD
}

func avg(nums []float64) float64 {
	sum := 0.0
	for _, num := range nums {
		sum += num
	}
	return sum / float64(len(nums))
}

func min(nums []float64) float64 {
	min := math.Inf(1)
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

func max(nums []float64) float64 {
	max := math.Inf(-1)
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

func httpStochRSI(klines []common.KLine) []float64 {
	dataJson, err := json.Marshal(klines)
	if err != nil {
		logger.Errorf("json.Marshal Error:%s", err)
		return nil
	}
	data, err := net.HttpPost(dataJson, "http://localhost:5000/stochrsi")
	if err != nil {
		logger.Errorf("httpStochRSI Error:%s", err)
		return nil
	}
	var stochRSI []float64
	err = json.Unmarshal(data, &stochRSI)
	if err != nil {
		logger.Errorf("json.Unmarshal Error:%s", err)
		return nil
	}
	return stochRSI
}
