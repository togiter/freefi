package micro_strategy

import (
	"fmt"
	common "freefi/strategymgr/common"
	"time"

	"github.com/mitchellh/mapstructure"
)

/*
*
一般来说，K值在20左右水平，从D值右方向上交叉D值时，为短期买进讯号；K值在80左右水平，从D值右方向下交叉D值时，为短期卖出讯号。更高层次的用法还包括：

1、K值形成一底比一底高的现象，并且在50以下的低水平，由下往上连续两次交叉D值时，股价涨幅往往会很大。
2、K值形成一底比一底低的现象，并且在50以上的高水平，由上往下连续两次交叉D值时，股价跌幅往往会很大。
3、K值高于80超买区时，短期股价容易向下回档；低于20超卖区时，短期股价容易向上反弹。然而，KDJ在实际使用过程中也存在种种“破绽”,

	比如：K值进入超买或超卖区后经常会发生徘徊和“钝化”现象，令投资者手足无措；股价短期波动剧烈或瞬间行情波动太大时，使用KD值交叉讯号买卖，经常发生买在高点、卖在低点的窘境。

另外，KDJ指标还常常具有如下分析用途：

1、J值﹥100,特别是连续3天大于100,则股价往往出现短期头部；

2、J值﹤0,特别是连续3天小于0,则股价往往又出现短期底部。值得投资者注意的是：J值的讯号不会经常出现，一旦出现，则可靠度相当高。

	在我们周围，有很多经验丰富的老投资者专门寻找J值的讯号，来把握股票的最佳买卖点，而这个讯号可以说是KDJ指标的精华所在。
*/
type KDJParams struct {
	KPeriod     int     `json:"kPeriod"`
	DPeriod     int     `json:"dPeriod"`
	JPeriod     int     `json:"jPeriod"`
	OverBuyVal  float64 `json:"overBuyVal"`
	OverSellVal float64 `json:"overSellVal"`
}

func ExecuteKDJ(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		Params: &params,
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}
	var kdjParams KDJParams
	err = mapstructure.Decode(params.Params, &kdjParams)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("kdj params to struct Error:", err)
		fmt.Println(err)
		return ret, err
	}

	if err = kdjParamsCheck(kdjParams); err != nil {
		return ret, fmt.Errorf("kdj params check Error %v", err)
	}
	tradeSide := common.TradeSideNone
	fomo := 0
	ks, ds, js := Kdj(klines, kdjParams.KPeriod, kdjParams.DPeriod, kdjParams.JPeriod)
	kLen := len(ks)
	if kLen <= 4 { //判断kdj斜率和收窄
		return ret, fmt.Errorf("KDJ params check Error: kdj len must > 4")
	}
	k, k_1, k_2, k_3 := ks[kLen-1], ks[kLen-2], ks[kLen-3], ks[kLen-4]
	d, d_1, d_2 := ds[kLen-1], ds[kLen-2], ds[kLen-3]
	j, j_1, j_2, j_3 := js[kLen-1], js[kLen-2], js[kLen-3], js[kLen-4]

	overBuyVal := 80.0
	overSellVal := 20.0
	mark := fmt.Sprintf("K(l-1,2,3):%.3f-%.3f-%.3f,D(l-1,2,3):%.3f,%.3f,%.3f,J(l-1,2,3):%.3f,%.3f,%.3f", k, k_1, k_2, d, d_1, d_2, j, j_1, j_2)
	if kdjParams.OverBuyVal > 0 {
		overBuyVal = kdjParams.OverBuyVal
	}
	if kdjParams.OverSellVal > 0 {
		overSellVal = kdjParams.OverSellVal
	}
	if j_1 > k_1 && j <= k && k_1 > overBuyVal {
		//1 死叉
		tradeSide = common.TradeSideShort
		fomo = 1
		mark = "=>KDJ 死叉" + mark

	} else if j_1 < k_1 && j >= k && k_1 <= overSellVal {
		//2.金叉
		tradeSide = common.TradeSideLong
		fomo = 1
		mark = "=>KDJ 金叉" + mark
	} else if !params.Legacy {
		if j_2 >= j_1 && j_1 > j && k_1 > overBuyVal {
			//3.高位反弹后持续下跌 || 死叉后持续下跌
			tradeSide = common.TradeSideShort
			mark = "=>KDJ 高位反弹后持续下跌 || 死叉后持续下跌" + mark
			if (j_2 >= j_3 && j_1 > k_1) || (j_3 >= k_3 && j_2 <= k_2) {
				//顶部反转后 ｜｜ 死叉后
				fomo = 1
			}

		} else if j_1 >= j_2 && j > j_1 && k_1 <= overSellVal {
			//4. 低位反弹并持续上扬 ｜｜ 金叉后持续上扬
			tradeSide = common.TradeSideLong

			mark = "=>KDJ 低位反弹并持续上扬 ｜｜ 金叉后持续上扬" + mark
			if (j_3 >= j_2 && j_1 <= k_1) || (j_3 <= k_3 && j_2 >= k_2) {
				fomo = 1
			}

		} else if j_1 > j && j_1 >= j_2 && k_1 >= overBuyVal {
			//5.高位反弹
			tradeSide = common.TradeSideShort
			mark = "=>KDJ 高位反弹" + mark
		} else if j_1 < j && j_1 <= j_2 && k_1 <= overSellVal {
			//6.低位反弹
			tradeSide = common.TradeSideShort
			mark = "=>KDJ 低位反弹" + mark
		}
	}
	ret.TradeSuggest.TradeSide = tradeSide
	ret.TradeSuggest.FomoLevel = fomo
	ret.TradeSuggest.Mark = mark
	return ret, nil
}

func kdjParamsCheck(params KDJParams) error {
	if params.KPeriod <= 0 || params.DPeriod <= 0 || params.JPeriod <= 0 {
		return fmt.Errorf("KDJ params check Error: KPeroid, DPeroid, JPeroid must > 0")
	}
	return nil
}
