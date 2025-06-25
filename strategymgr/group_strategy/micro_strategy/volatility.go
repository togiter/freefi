package micro_strategy

import (
	"fmt"
	common "freefi/strategymgr/common"
	"math"
	"time"

	"github.com/mitchellh/mapstructure"
)

type VolatiParams struct {
	//平均K线数量
	EffectKLineNum int64 `json:"effectKLineNum"`
	//波动幅度因子 ,VolatilityFactor * avg(SUM(EffectKLineNum_KLINEs))
	VFactor float64 `json:"vFactor"`
	//DiffType 0: |close-open|, 1: low-high
	DiffType int `json:"diffType"`
}

// 波动性,暴涨 暴跌
func ExecuteVolatility(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}
	tradeSide := common.TradeSideNone
	fomo := 0
	var volatiParams VolatiParams
	err = mapstructure.Decode(params.Params, &volatiParams)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("volatiParams params to struct Error:%v", err)
		return ret, err
	}
	if len(klines) < int(volatiParams.EffectKLineNum) {
		return ret, fmt.Errorf("klines len less than EffectKLineNum")
	}
	maxVolati := 0.0 //最大波动
	minVolati := 0.0 //最小波动
	maxIdx := 0
	minIdx := 0
	kMap := make(map[int]float64)
	newKlines := klines[len(klines)-int(volatiParams.EffectKLineNum):]
	for i := int(volatiParams.EffectKLineNum) - 1; i > 0; i-- {

		diff := math.Abs(newKlines[i].Close - newKlines[i].Open)
		if volatiParams.DiffType == 1 {
			//low-high
			diff = math.Abs(newKlines[i].High - newKlines[i].Low)
		}
		if diff > maxVolati {
			maxVolati = diff
			maxIdx = i
		}
		if diff < minVolati || minVolati <= 0.00000001 {
			minVolati = diff
			minIdx = i
		}
		kMap[i] = diff
	}
	sum := 0.0
	for i := 0; i < int(volatiParams.EffectKLineNum); i++ {
		if i == maxIdx || i == minIdx {
			//最大最小值移除
			continue
		}
		sum += kMap[i]
	}
	avg := volatiParams.VFactor * (sum / float64(volatiParams.EffectKLineNum-2)) //2最大值&最小值
	oneK := newKlines[len(newKlines)-1]
	newVolati := oneK.Close - oneK.Open
	if volatiParams.DiffType == 1 {
		//low-high
		newVolati = oneK.High - oneK.Low
	}
	mark := ""
	if math.Abs(newVolati) > avg { //超过波动阈值
		if newVolati > 0 { //阳线
			tradeSide = common.TradeSideLong
			mark = fmt.Sprintf("【阳线】当前k线波动(%.5f)明显比过去(%d)平均波动(%.5f)大", newVolati, volatiParams.EffectKLineNum-2, avg)
		} else {
			tradeSide = common.TradeSideShort
			mark = fmt.Sprintf("【阴线】当前k线波动(%.5f)明显比过去(%d)平均波动(%.5f)大", newVolati, volatiParams.EffectKLineNum-2, avg)

		}
		fomo = 1

	}

	// logger.Infof("volatility new(%.5f) avg(%.5f) max(%.5f) min(%.5f)mark: %s", newVolati, avg, maxVolati, minVolati, mark)
	ret.TradeSuggest.TradeSide = tradeSide
	ret.TradeSuggest.Mark = mark
	ret.TradeSuggest.FomoLevel = fomo
	return ret, nil

}
