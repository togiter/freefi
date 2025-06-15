package micro_strategy

import (
	// "encoding/json"
	"fmt"
	common "freefi/strategymgr/common"
	"freefi/strategymgr/pkg/logger"
	"math"
	"time"

	"github.com/mitchellh/mapstructure"
)

type Cur2MaxParams struct {
	//
	Cur2MaxMinDiffRate float64 `json:"cur2MaxMinDiffRate"`
	//最大最小值涨跌幅
	MaxMinDiffRate float64 `json:"maxMinDiffRate"`
	//是否使用收盘价作为计算？ 默认使用最高(波峰)/最低价(波谷)
	UseClose bool `json:"useClose"`
	//最大k线
	Limit int `json:"limit"`
}

func (cm *Cur2MaxParams) check() error {
	if cm.MaxMinDiffRate <= 0 {
		return fmt.Errorf("ThresHoldDiff must be greater than 0")
	}
	if cm.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}
	return nil
}

// 获取当前价格到最近波峰或波谷的价差，根据价差作为一个限制联合其他策略进行交易操作，不建议单独使用
func ExecuteCur2Max(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}
	var c2m Cur2MaxParams
	err = mapstructure.Decode(params.Params, &c2m)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("Cur2Max params to struct Error:%v", err)
		return ret, err
	}
	if err := c2m.check(); err != nil {
		return ret, err
	}
	kls := klines[len(klines)-c2m.Limit-1:]
	trade, mark := calDiff(kls, c2m)
	ret.TradeSuggest.TradeSide = trade
	ret.TradeSuggest.Mark = mark
	return ret, nil
}

type KMode struct {
	Kline common.KLine
	Index int
}

func calDiff(nKlines []common.KLine, c2m Cur2MaxParams) (common.TradeSide, string) {
	kLen := len(nKlines)
	if kLen < c2m.Limit {
		return common.TradeSideNone, fmt.Sprintf("kline len less than limit:%d", c2m.Limit)
	}
	maxKline := KMode{
		Kline: nKlines[kLen-1],
		Index: 0,
	}
	minKline := maxKline
	peakKline := maxKline
	valleyKline := maxKline
	for i := kLen - 2; i > 0; i-- {
		k := nKlines[i]
		k_1 := nKlines[i-1]
		k1 := nKlines[i+1]
		//更新最大值
		if k.High > maxKline.Kline.High {
			maxKline = KMode{
				Kline: k,
				Index: i,
			}
		}
		//更新最小值
		if k.Low < minKline.Kline.Low {
			minKline = KMode{
				Kline: k,
				Index: i,
			}
		}
		//更新波峰
		if k.High >= k1.High && k.High >= k_1.High {
			if peakKline.Kline.High < k.High {
				peakKline = KMode{
					Kline: k,
					Index: i,
				}
			}
		}
		//更新波谷
		if k.Low <= k1.Low && k.Low <= k_1.Low {
			valleyKline = KMode{
				Kline: k,
				Index: i,
			}
		}
	}
	if peakKline.Kline.High < maxKline.Kline.High {
		peakKline = maxKline
	}
	if valleyKline.Kline.Low > minKline.Kline.Low {
		valleyKline = minKline
	}

	cur := nKlines[kLen-1].Close
	//当前价格到最近波峰或波谷的价差
	cur2Min := cur - valleyKline.Kline.Low
	cur2Max := maxKline.Kline.High - cur
	//绝对值
	max2MinDiff := math.Abs(peakKline.Kline.High - valleyKline.Kline.Low)
	logger.Infof("cur:%f,cur2Min(%f-%f):%f,cur2Max(%f-%f):%f, max2MinDiff(%f-%f):%f", cur, cur, valleyKline.Kline.Low, cur2Min, maxKline.Kline.High, cur, cur2Max, peakKline.Kline.High, valleyKline.Kline.Low, max2MinDiff)
	//如果当前价格属于波峰范围
	if peakKline.Index > valleyKline.Index {
		//相对值 最大
		cur2MaxRate := math.Abs(cur-peakKline.Kline.High) / peakKline.Kline.High
		//最大值/最小值的波动率
		max2MinRate := max2MinDiff / valleyKline.Kline.Low
		if cur2MaxRate/max2MinRate < c2m.Cur2MaxMinDiffRate && max2MinRate >= c2m.MaxMinDiffRate { //误差在阀值范围内，并且满足绝对值涨跌幅
			logger.Infof("cur2Max:%f, MaxMinDiffRate:%f,cur2MaxRate:%f, max2MinRate:%f", cur2Max, c2m.MaxMinDiffRate, cur2MaxRate, max2MinRate)
			return common.TradeSideShort, fmt.Sprintf("cur2Max:%f, thresHoldDiff:%f", cur2Max, c2m.MaxMinDiffRate)
		}
	} else {
		//相对值
		cur2MaxRate := math.Abs(cur-valleyKline.Kline.Low) / valleyKline.Kline.Low
		//最大值/最小值的波动率
		max2MinRate := max2MinDiff / peakKline.Kline.High
		if cur2Min/max2MinRate < c2m.Cur2MaxMinDiffRate && max2MinRate >= c2m.MaxMinDiffRate { //误差在阀值范围内，并且满足绝对值涨跌幅
			logger.Infof("111cur2Max:%f, MaxMinDiffRate:%f,cur2MaxRate:%f, max2MinRate:%f", cur2Max, c2m.MaxMinDiffRate, cur2MaxRate, max2MinRate)
			return common.TradeSideLong, fmt.Sprintf("cur2Max:%f, thresHoldDiff:%f", cur2Max, c2m.MaxMinDiffRate)
		}
	}
	return common.TradeSideNone, ""
}
