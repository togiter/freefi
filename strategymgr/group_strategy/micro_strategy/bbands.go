package micro_strategy

import (
	"fmt"
	common "freefi/strategymgr/common"
	"time"

	"github.com/markcheno/go-talib"
	"github.com/mitchellh/mapstructure"
)

type BBandsParams struct {
	//时间周期
	InTimePeriod int `json:"inTimePeriod"`
	//标准差
	Deviation float64 `json:"deviation"`
	//用来判断在up<=>mid<=>down之间震荡k线数量
	KLineNum *int `json:"avgKLineNum"`
	//移动平均类型 默认是SMA
	MAType *int `json:"maType"`
	//价格类型 默认是close
	PriceType *string `json:"priceType"`
}

func bbandsCheck(params BBandsParams) bool {
	return params.InTimePeriod > 0 && params.Deviation > 0
}

func ExecuteBBands(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}

	var bbandsParams BBandsParams
	err = mapstructure.Decode(params.Params, &bbandsParams)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("bbandsParams params to struct Error:%v", err)
		return ret, err
	}
	if !bbandsCheck(bbandsParams) {
		return ret, fmt.Errorf("bbandsParams check Error")
	}
	maType := talib.SMA
	if bbandsParams.MAType != nil {
		maType = talib.MaType(*bbandsParams.MAType)
	}
	priceType := InClose
	// if bbandsParams.PriceType != nil {
	// 	priceType = *bbandsParams.PriceType
	// }
	ups, mids, lows := talib.BBands(getPrices(klines, priceType), bbandsParams.InTimePeriod, bbandsParams.Deviation, bbandsParams.Deviation, maType)
	kLen := len(ups)
	tradeSide := common.TradeSideNone
	fomo := 0
	mark := ""
	if kLen < 3 {
		mark = fmt.Sprintf("指标(%s=>%v)计算结果异常~len(ups) <= 3 ", params.Name, params.Params)
		return ret, fmt.Errorf("%s", mark)
	}
	hdlSize := 7
	if bbandsParams.KLineNum != nil {
		hdlSize = *bbandsParams.KLineNum
	}
	//(平均)价格在mid线下方震荡，然后收盘价格上穿mid线，做多，收盘价格下穿low线，做空
	//(平均)价格在mid线上方震荡，然后收盘价格下穿mid线，做空，收盘价格上穿up线，做多
	//价格从up线上方下穿up线，做空，从low线下方上穿low线，做多
	hdlKls := klines[kLen-hdlSize:]
	hdlUps := ups[kLen-hdlSize:]
	hdlMids := mids[kLen-hdlSize:]
	hdlLows := lows[kLen-hdlSize:]
	latestK := hdlKls[hdlSize-1]
	latestUp := hdlUps[hdlSize-1]
	latestMid := hdlMids[hdlSize-1]
	latestLow := hdlLows[hdlSize-1]
	//1:[mid,up]KLineNum, -1:[min,low]KLineNum, 0:震荡
	dir := 0
	//判断[n-size:n-1)
	if Price_Up_Reverse(hdlKls, hdlUps) {
		//上穿up线再反弹下来
		tradeSide = common.TradeSideShort
		mark = "价格上穿up后反弹,建议空"
	} else if Price_Down_Reverse(hdlKls, hdlLows) {
		//下穿low线再反弹上来
		tradeSide = common.TradeSideLong
		mark = "价格下穿down线后反弹,建议多"
	} else if BetweenMidAndDown(hdlKls[:hdlSize-1], hdlMids[:hdlSize-1], hdlLows[:hdlSize-1]) { //
		dir = -1
		//mid线下震荡
		if latestK.C_O_Price() > 0.0 && latestK.MidPrice() >= latestMid {
			//阳线且上穿mid线
			tradeSide = common.TradeSideLong
			// txp = common.TradeShortClose
			mark = "价格在【mid,down】中间震荡后上穿mid线,建议多(平空)"
		} else if latestK.C_O_Price() < 0.0001 && latestK.MidPrice() <= latestLow {
			//阴线且下穿low线
			tradeSide = common.TradeSideShort
			mark = "价格在【mid,down】中间震荡后下穿down线,建议空(平多)"
		}
	} else if BetweenMidAndUp(hdlKls[:hdlSize-1], hdlMids[:hdlSize-1], hdlUps[:hdlSize-1]) {
		dir = 1
		//mid线上震荡
		if latestK.C_O_Price() > 0.0 && latestK.MidPrice() >= latestUp {
			//阳线且上穿up线
			tradeSide = common.TradeSideLong
			mark = "价格在【mid,up】中间震荡后上穿up线,建议多(平空)"
		} else if latestK.C_O_Price() < 0.0001 && latestK.MidPrice() <= latestMid {
			//阴线且下穿mid线
			tradeSide = common.TradeSideShort
			//    txp = common.TradeLongClose
			mark = "价格在【mid,up】中间震荡后下穿mid线,建议空(平多)"
		}
	} else {
		tradeSide = Price_Mid_Reverse(hdlKls, hdlMids)
		mark = "上(下)穿mid线,建议x"
	}
	ret.TradeSuggest.TradeSide = tradeSide
	ret.TradeSuggest.FomoLevel = fomo
	ret.TradeSuggest.Mark = mark
	ret.Opts = make(map[string]interface{})
	ret.Opts["dir"] = dir
	return ret, nil
}

// // 计算趋势强度
func DetermineTrendStrength(upperBand, lowerBand float64) float64 {
	bandWidth := upperBand - lowerBand
	return bandWidth
}

// Price_Up_Reverse 上穿up线并回调
func Price_Up_Reverse(klines []common.KLine, ups []float64) bool {
	kLen := len(klines)
	if kLen != len(ups) || kLen < 3 {
		return false
	}
	kn, kn_1, kn_2 := klines[kLen-1], klines[kLen-2], klines[kLen-3]
	upn, upn_1, upn_2 := ups[kLen-1], ups[kLen-2], ups[kLen-3]
	if (kn_1.High > upn_1 || kn_2.High > upn_2) && kn.Close <= upn && kn.C_O_Price() <= 0.00001 {
		return true
	}

	return false

}

// Price_Down_Reverse 下穿low线并回调
func Price_Down_Reverse(klines []common.KLine, lows []float64) bool {
	kLen := len(klines)
	if kLen != len(lows) || kLen < 3 {
		return false
	}
	kn, kn_1, kn_2 := klines[kLen-1], klines[kLen-2], klines[kLen-3]
	lown, lown_1, lown_2 := lows[kLen-1], lows[kLen-2], lows[kLen-3]
	if (kn_1.Low < lown_1 || kn_2.Low < lown_2) && kn.Close >= lown && kn.C_O_Price() >= 0.00001 {
		return true
	}

	return false

}

// Price_Mid_Reverse 上(下)穿mid线并回调
func Price_Mid_Reverse(klines []common.KLine, mids []float64) (tx common.TradeSide) {
	kLen := len(klines)
	if kLen != len(mids) || kLen < 3 {
		return
	}
	kn, kn_1, kn_2 := klines[kLen-1], klines[kLen-2], klines[kLen-3]
	midn, midn_1, midn_2 := mids[kLen-1], mids[kLen-2], mids[kLen-3]
	if kn_1.Low < midn_1 && kn_2.AvgPrice() > midn_2 && kn.AvgPrice() >= midn && kn.C_O_Price() >= 0.00001 {
		//下穿中线并反弹
		return common.TradeSideLong //common.TradeShortPre
	} else if kn_1.High > midn_1 && kn_2.AvgPrice() < midn_2 && kn.AvgPrice() <= midn && kn.C_O_Price() <= 0.00001 {
		//上穿中线并反弹
		return common.TradeSideShort
	}

	return common.TradeSideNone

}

// BetweenMidAndDown 中线和底线之间震荡
func BetweenMidAndDown(klines []common.KLine, mids, downs []float64) bool {
	kLen := len(klines)
	if kLen != len(mids) || kLen != len(downs) {
		return false
	}
	ret := true
	for i := 0; i < kLen; i++ {
		kl := klines[i]
		mid := mids[i]
		down := downs[i]
		if kl.AvgPrice() > mid || kl.AvgPrice() < down {
			return false
		}
	}
	return ret
}

// BetweenMidAndUp 中线与up线之间震荡
func BetweenMidAndUp(klines []common.KLine, mids, ups []float64) bool {
	kLen := len(klines)
	if kLen != len(mids) || kLen != len(ups) {
		return false
	}
	ret := true
	for i := 0; i < kLen; i++ {
		kl := klines[i]
		mid := mids[i]
		up := ups[i]
		if kl.AvgPrice() < mid || kl.AvgPrice() > up {
			return false
		}
	}
	return ret
}
