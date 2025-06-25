package micro_strategy

import (
	"freefi/strategymgr/common"

	gotart "github.com/iamjinlei/go-tart"
	gotalib "github.com/markcheno/go-talib"
	// cgotalib "github.com/kjx98/cgo-talib"
)

type DataSort int

const (
	//倒序
	SortReverse DataSort = iota
	//正序
	SortPositive
)

func MaType(t int) gotalib.MaType {
	switch t {
	case 0:
		return gotalib.SMA
	case 1:
		return gotalib.EMA
	case 2:
		return gotalib.WMA
	case 3:
		return gotalib.DEMA
	case 4:
		return gotalib.TEMA
	case 5:
		return gotalib.TRIMA
	case 6:
		return gotalib.KAMA
	case 7:
		return gotalib.MAMA
	case 8:
		return gotalib.T3MA
	default:
		return gotalib.SMA
	}

}

func StochRsi(data []common.KLine, inPeriod int, kn, dn int, maType int) (k, d []float64) {
	cls := reData(data, InClose, SortPositive)
	return gotalib.StochRsi(cls, inPeriod, kn, dn, MaType(maType))
	//return gotart.StochRsiArr(cls,int64(inPeriod),int64(kn),gotart.SMA,int64(dn))
	// return cgotalib.StochRsi(cls,int32(inPeriod),int32(kn),int32(dn),int32(maType))
}
func StochRsi1(data []common.KLine, inPeriod int, kn, dn int, maType int) (k, d []float64) {
	cls := reData(data, InClose, SortPositive)
	// return gotalib.StochRsi(cls,inPeriod,kn,dn,MaType(maType))
	return gotart.StochRsiArr(cls, int64(inPeriod), int64(kn), gotart.SMA, int64(dn))
	// return cgotalib.StochRsi(cls,int32(inPeriod),int32(kn),int32(dn),int32(maType))
}

func Obv(data []common.KLine) []float64 {
	closes := reData(data, InClose, SortPositive)
	vols := reData(data, InVol, SortPositive)
	return gotart.ObvArr(closes, vols)
	//return gotalib.Obv(closes,vols)
}

func Kdj(data []common.KLine, n1, n2, n3 int) (k, d, j []float64) {
	kdjObj := NewKdj(n1, n2, n3)
	sortType := SortPositive
	highs := reData(data, InHigh, sortType)
	lows := reData(data, InLow, sortType)
	closes := reData(data, InClose, sortType)
	return kdjObj.KdjCall(highs, lows, closes)
}

func Mfi(data []common.KLine, inTimePeriod int) []float64 {
	return gotalib.Mfi(reData(data, InHigh, SortPositive), reData(data, InLow, SortPositive), reData(data, InClose, SortPositive), reData(data, InVol, SortPositive), inTimePeriod)
}

func Sar(data []common.KLine, acceleration float64, maximum float64) []float64 {
	//go-tart没有sar
	return gotalib.Sar(realData(data, InHigh), realData(data, InLow), acceleration, maximum)
	// return cgotalib.Sar(realData(data,InHigh),realData(data,InLow),acceleration,maximum)
}

func Dmi(data []common.KLine, period int) (dx, pdi, mdi, adx, adxr []float64) {
	inHighs := realData(data, InHigh)
	inLows := realData(data, InLow)
	// inCloses := realData(data,InClose)
	inHighs_1 := reData(data, InHigh, SortPositive)
	inLows_1 := reData(data, InLow, SortPositive)
	inCloses_1 := reData(data, InClose, SortPositive)
	dx = gotart.DxArr(inHighs_1, inLows_1, inCloses_1, int64(period))
	pdi = gotalib.PlusDM(inHighs, inLows, period)
	mdi = gotalib.MinusDM(inHighs, inLows, period)
	adx = gotart.AdxArr(inHighs_1, inLows_1, inCloses_1, int64(period))
	adxr = gotart.AdxRArr(inHighs_1, inLows_1, inCloses_1, int64(period))
	return
}

func Cci(data []common.KLine, inTimePeriod int) []float64 {
	return gotart.CciArr(reData(data, InHigh, SortPositive), reData(data, InLow, SortPositive), reData(data, InClose, SortPositive), int64(inTimePeriod))
}
func WillR(data []common.KLine, inTimePeriod int) []float64 {
	return gotart.WillRArr(reData(data, InHigh, SortPositive), reData(data, InLow, SortPositive), reData(data, InClose, SortPositive), int64(inTimePeriod))
	// return gotalib.WillR(realData(data, InHigh),realData(data, InLow),realData(data, InClose),inTimePeriod)//gotart.WillRArr(reData(data, InHigh,SortPositive),reData(data, InLow,SortPositive),reData(data, InClose,SortPositive),int64(inTimePeriod))
	//return cgotalib.WillR(realData(data, InHigh),realData(data, InLow),realData(data, InClose),int32(inTimePeriod))//gotart.WillRArr(reData(data, InHigh,SortPositive),reData(data, InLow,SortPositive),reData(data, InClose,SortPositive),int64(inTimePeriod))
}

func Ma(data []common.KLine, inTimePeriod int, maType int, priceTy int) []float64 {
	return gotart.MaArr(gotart.MaType(maType), reData(data, priceTy, SortPositive), int64(inTimePeriod))
	//return gotalib.Ma(realData(data, priceTy)), inTimePeriod, gotalib.MaType(maType))
}

func Atr(data []common.KLine, inTimePeriod int) []float64 {
	var (
		inHigh  []float64
		inLow   []float64
		inClose []float64
	)

	for i := len(data) - 1; i >= 0; i-- {
		k := data[i]
		inHigh = append(inHigh, k.High)
		inLow = append(inLow, k.Low)
		inClose = append(inClose, k.Close)
	}

	return gotalib.Atr(inHigh, inLow, inClose, inTimePeriod)
}

func Macd(data []common.KLine, inFastPeriod int, inSlowPeriod int, inSignalPeriod int, priceTy int) (DIF, DEA, MACD []float64) {
	// var macd []float64
	//dif, dea, hist := gotalib.Macd(realData(data, InClose), inFastPeriod, inSlowPeriod, inSignalPeriod)
	dif, dea, hist := gotart.MacdArr(reData(data, InClose, SortPositive), int64(inFastPeriod), int64(inSlowPeriod), int64(inSignalPeriod))
	return dif, dea, hist
}

// 布林带
func BBands(data []common.KLine, inTimePeriod int, deviation float64, maType int, priceTy int) (up, middle, low []float64) {
	return gotart.BBandsArr(gotart.MaType(maType), reData(data, priceTy, SortPositive), int64(inTimePeriod), deviation, deviation)
	//return gotalib.BBands(realData(data,priceTy)), inTimePeriod, deviation, deviation, gotalib.MaType(maType))
}

func Rsi(data []common.KLine, inTimePeriod int, priceTy int) []float64 {
	return gotart.RsiArr(reData(data, priceTy, SortPositive), int64(inTimePeriod))
	//return gotalib.Rsi(reData(data, priceTy),SortPositive), inTimePeriod)
}

func realData(data []common.KLine, priceTy int) []float64 {
	return reData(data, priceTy, SortReverse)
}

func reData(data []common.KLine, priceTy int, sort DataSort) []float64 {
	var inReal []float64
	if sort == SortReverse {
		for i := len(data) - 1; i >= 0; i-- {
			k := data[i]
			switch priceTy {
			case InClose:
				inReal = append(inReal, k.Close)
			case InHigh:
				inReal = append(inReal, k.High)
			case InLow:
				inReal = append(inReal, k.Low)
			case InOpen:
				inReal = append(inReal, k.Open)
			case InVol:
				inReal = append(inReal, k.Volume)
			default:
				panic("please set ema type")
			}
		}
	} else {
		for i := 0; i < len(data); i++ {
			k := data[i]
			switch priceTy {
			case InClose:
				inReal = append(inReal, k.Close)
			case InHigh:
				inReal = append(inReal, k.High)
			case InLow:
				inReal = append(inReal, k.Low)
			case InOpen:
				inReal = append(inReal, k.Open)
			case InVol:
				inReal = append(inReal, k.Volume)
			default:
				panic("please set ema type")
			}
		}
	}

	return inReal
}
