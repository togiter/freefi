package micro_strategy

import (
	// "encoding/json"
	"fmt"
	common "freefi/strategymgr/common"
	"time"

	"github.com/mitchellh/mapstructure"
)

type Cur2MaxParams struct {
	//当前值和最大/最小值的差值
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
		return fmt.Errorf("MaxMinDiffRate must be greater than 0")
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
	kls := klines[len(klines)-c2m.Limit:]
	trade, mark := calDiff(kls, c2m)
	ret.TradeSuggest.TradeSide = trade
	ret.TradeSuggest.Mark = mark
	return ret, nil
}

type KMode struct {
	Kline common.KLine
	Index int
}

// 1. 遍历数组获得最大最小值
// 2. 计算最大最小值和当前值的差值
// 3. 计算最大值和最小值距离当前值的相近程度，去最近
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
	curKline := maxKline
	for i := 0; i < kLen; i++ {
		idx := i
		k := nKlines[i]
		//更新最大值
		if k.High > maxKline.Kline.High {
			maxKline = KMode{
				Kline: k,
				Index: idx,
			}
		}
		//更新最小值
		if k.Low < minKline.Kline.Low {
			minKline = KMode{
				Kline: k,
				Index: idx,
			}
		}
	}
	cur := curKline.Kline.Close
	//当前价格到最近波峰或波谷的价差
	cur2Min := (cur - minKline.Kline.Low) / cur
	cur2Max := (maxKline.Kline.High - cur) / cur
	if minKline.Index > maxKline.Index {
		if cur2Min > c2m.Cur2MaxMinDiffRate {
			return common.TradeSideShort, fmt.Sprintf("当前值(%v)-最小值(%v) = %v ", cur, minKline.Kline.Low, cur2Min)
		}
		if cur2Max > c2m.Cur2MaxMinDiffRate {
			return common.TradeSideLong, fmt.Sprintf("最大值(%v) - 当前值(%v) = %v ", maxKline.Kline.High, curKline.Kline.Close, cur2Max)
		}
	} else {
		if cur2Max > c2m.Cur2MaxMinDiffRate {
			return common.TradeSideLong, fmt.Sprintf("最大值(%v) - 当前值(%v) = %v ", maxKline.Kline.High, curKline.Kline.Close, cur2Max)
		}
		if cur2Min > c2m.Cur2MaxMinDiffRate {
			return common.TradeSideShort, fmt.Sprintf("当前值(%v)-最小值(%v) = %v ", cur, minKline.Kline.Low, cur2Min)
		}
	}

	return common.TradeSideNone, ""
}
