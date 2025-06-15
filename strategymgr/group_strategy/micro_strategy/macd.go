package micro_strategy

import (
	// "encoding/json"
	"fmt"
	common "freefi/strategymgr/common"
	"time"

	"github.com/mitchellh/mapstructure"
)

/*
*

基本用法
1. MACD金叉：DIF由下向上突破 DEA，为买入信号。
2. MACD死叉：DIF由上向下突破 DEA，为卖出信号。
3. MACD 绿转红：MACD 值由负变正，市场由空头转为多头。
4. MACD 红转绿：MACD 值由正变负，市场由多头转为空头。
5. DIFF 与 DEA 均为正值,即都在零轴线以上时，大势属多头市场，DIFF 向上突破 DEA，可作买入信号。
6. DIFF 与 DEA 均为负值,即都在零轴线以下时，大势属空头市场，DIFF 向下跌破 DEA，可作卖出信号。
7. 当 DEA 线与 K 线趋势发生背离时为反转信号。
8. DEA 在盘整局面时失误率较高,但如果配合RSI 及KDJ指标可适当弥补缺点。
*/
type MACDParams struct {
	FastPeriod   int `json:"fastPeriod"`
	SlowPeriod   int `json:"slowPeriod"`
	SignalPeriod int `json:"signalPeriod"`
	//非标准参数
	//单边行情大小限制
	ContinueSize int `json:"continueSize"`
	//单边行情强度限制
	ContinueFomo int `json:"continueFomo"`
}

func macdCheck(mp MACDParams) bool {
	return mp.FastPeriod > 0 && mp.SlowPeriod > 0 && mp.SignalPeriod > 0 && mp.FastPeriod < mp.SlowPeriod && mp.SignalPeriod < mp.SlowPeriod
}
func ExecuteMACD(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}
	tradeSide := common.TradeSideNone
	fomo := 0
	var macdParams MACDParams
	err = mapstructure.Decode(params.Params, &macdParams)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("macd params to struct Error:", err)
		return ret, err
	}
	if !macdCheck(macdParams) {
		return ret, fmt.Errorf("macd params check Error")
	}
	difs, deas, macds := Macd(klines, macdParams.FastPeriod, macdParams.SlowPeriod, macdParams.SignalPeriod, 0)
	difLen := len(difs)
	deaLen := len(deas)
	maLen := len(macds)
	if difLen <= 4 || deaLen <= 4 || maLen <= 4 {
		ret.TradeSuggest.Mark = fmt.Sprintf("指标(%s)参数(%s)计算结果有误~difLen == 4 || deaLen == 4 || maLen == 4", params.Name, params.Params)
		return
	}
	dif, dif_1, dif_2, dif_3 := difs[difLen-1], difs[difLen-2], difs[difLen-3], difs[difLen-4]
	dea, dea_1, dea_2, _ := deas[deaLen-1], deas[deaLen-2], deas[deaLen-3], deas[deaLen-4]
	maV, maV_1, maV_2 := macds[maLen-1], macds[maLen-2], macds[maLen-3]

	markV := ""

	//单边行情大小限制
	continueSize := 8
	if macdParams.ContinueSize > 0 {
		continueSize = macdParams.ContinueSize
	}
	//单边行情强度限制
	continueFomo := 3
	if macdParams.ContinueFomo > 0 {
		continueFomo = macdParams.ContinueFomo
	}
	continueCount, dir := continueCountDir(macds)
	//单边行情刚开启
	if continueCount <= continueFomo {
		fomo = 1
	}

	if dif >= dea && dif_1 < dea_1 {
		//1. 向上穿(金)插
		tradeSide = common.TradeSideLong
		fomo = 1
		markV = "向上穿(金)插"
	} else if dif <= dea && dif_1 > dea_1 {
		//2. 向下死叉
		tradeSide = common.TradeSideShort
		fomo = 1
		markV = "向下死叉"
	} else if !params.Legacy {
		if dif_3 > dif_2 && dif_2 >= dif_1 && dif > dif_1 && continueCount >= continueSize && dir == -1 {
			tradeSide = common.TradeSideLong
			markV = fmt.Sprintf("MA值连续%d次为负,方向首次向上反弹", continueCount)

		} else if dif_3 < dif_2 && dif_2 <= dif_1 && dif < dif_1 && continueCount >= continueSize && dir == 1 {

			tradeSide = common.TradeSideShort
			markV = fmt.Sprintf("MA值连续%d次为正,方向首次向下反弹", continueCount)

		} else if dif_3 <= dif_2 && dif_2 >= dif_1 && dif <= dif_1 && continueCount >= continueSize && dir == 1 {
			//3.顶部反弹持续向下
			tradeSide = common.TradeSideShort
			fomo = 1
			markV = fmt.Sprintf("MA值连续%d次为正,方向持续向下反弹", continueCount)

		} else if dif_3 >= dif_2 && dif_2 <= dif_1 && dif > dif_1 && continueCount >= continueSize && dir == -1 {
			//4.底部反弹持续向上
			tradeSide = common.TradeSideLong
			fomo = 1
			markV = fmt.Sprintf("MA值连续%d次为负,方向持续向上反弹", continueCount)
		}
	}

	ret.TradeSuggest.TradeSide = tradeSide
	ret.TradeSuggest.FomoLevel = fomo
	ret.TradeSuggest.Mark = fmt.Sprintf("%s,Dif(l-1,2,3):%.5f,%.5f,%.5f;Dea(l-1,2,3):%.5f,%.5f,%.5f;MACD(l-1,2,3):%.5f,%.5f,%.5f", markV, dif, dif_1, dif_2, dea, dea_1, dea_2, maV, maV_1, maV_2)
	ret.Opts = make(map[string]interface{})
	ret.Opts["dir"] = -1
	if dif > dea {
		ret.Opts["dir"] = 1
	}
	return ret, nil
}

// continueCountDir 持续行情及方向
func continueCountDir(mas []float64) (cCount int, dir int) {
	mLen := len(mas)
	if mLen == 0 {
		return
	}
	latestM := mas[mLen-1]
	if latestM > 0 {
		dir = 1
		for i := mLen - 1; i >= 0; i-- {
			if mas[i] >= 0 { //持续同方向
				cCount++
				continue
			}
			return
		}

	} else {
		dir = -1
		for i := mLen - 1; i >= 0; i-- {
			if mas[i] <= 0 { //持续同方向
				cCount++
				continue
			}
			return
		}

	}
	return
}
