package micro_strategy

import (
	// "encoding/json"
	"fmt"
	common "freefi/strategymgr/common"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/utils"
	"time"

	"github.com/mitchellh/mapstructure"
)

const (
	MaxType = "MAX"
	MinType = "MIN"
)

type PeakAndValley struct {
	//滑动窗口大小
	WindowSize int `json:"windowSize"`
	//当前值和最小/最大的差值阈值，满足这个值这可以做空(curK - minK)/做多(maxK - curK)
	DiffRate      float64 `json:"diffRate"`
	ToleranceRate float64 `json:"toleranceRate"` // 容忍率，允许的最大最小值差值
	//差值阈值，在这个值之内均可做空/做多
	ToleranceDiffK int `json:"toleranceDiffK"` // 差值阈值，满足这个值才做空/做多
	//最大k线
	Limit *int `json:"limit"`
}

func checkPeakAndValley(params PeakAndValley) error {
	if params.WindowSize < 3 {
		return fmt.Errorf("window must be greater than or equal to 3")
	}
	return nil
}

// 1. 计算每段滑动窗口的最大和最小值及其索引，时间.每一个波峰(波峰)，其前后一定是波谷(波峰)，如此交替。
// 如果在相邻窗口没有找到更高的波峰或者波峰，则不断持续下去每个窗口
// 2. 纠正/去重，相邻滑动窗口的最大值可能只有，甚至不是波峰/波谷，需要判断是否波峰/波谷
// 3. 计算最大值和最小值之间的差值，判断是否做空/做多
func ExecutePeakAndValley(klines []common.KLine, params MicroStrategyParams) (ret *MicroStrategyRet, err error) {
	ret = &MicroStrategyRet{
		TradeSuggest: common.TradeSuggest{
			TradeSide:  common.TradeSideNone,
			CreateTime: time.Now().Unix(),
		},
	}
	var peakAndValleyParams PeakAndValley
	err = mapstructure.Decode(params.Params, &peakAndValleyParams)
	// err = json.Unmarshal([]byte(params.Params), &macdParams)
	if err != nil {
		err = fmt.Errorf("masRsiParamscd params to struct Error:%v", err)
		return ret, err
	}
	if err := checkPeakAndValley(peakAndValleyParams); err != nil {
		return ret, err
	}
	if peakAndValleyParams.Limit != nil && *peakAndValleyParams.Limit > 0 {
		// kLen := len(klines)
		// klines = klines[kLen-int(*peakAndValleyParams.Limit):]
	}
	tradeSide := common.TradeSideNone
	window := peakAndValleyParams.WindowSize
	//价格类型
	// closePrices := getPrices(klines, InClose)
	// 计算最高值和最低值及其索引和差值
	values := flitterPeakAndValleys(klines, window)
	if len(values) < 2 {
		return ret, nil
	}
	for i := len(values) - 1; i >= 0; i-- {
		kine := klines[values[i].Index]
		logger.Infof("%v combine kine Close=%v,time(%d): %v", values[i].Type, kine.Close, kine.CloseTime, utils.TimeFmt(kine.CloseTime/1000, "2006-01-02 15:04:05"))
	}

	ret.TradeSuggest.TradeSide = tradeSide
	return ret, nil
}

// 实现：
// 获取指定k线数量在指定滑动窗口内的最大值和最小值，作为支撑位和压力位
// 逐次计算最大最小值的差
// 返回当前k线距离最近最大/最小值的差
// ValueIndex 结构体表示值及其索引和类型

// ValueIndex 结构体表示值、索引、类型和前后差值
type ValueIndex struct {
	Value      float64
	Index      int
	Type       string  // "max" 或 "min"
	PrevDiff   float64 // 前后差值
	MaxMinDiff float64 // 最大值和最小值之间的差值
	Kline      common.KLine
}

// MaxMinAndIndices - 返回指定时间段内的最高值、最低值及其索引和差值
func MaxMinAndIndices(inReal []float64, inTimePeriod int) (maxValues []ValueIndex, minValues []ValueIndex, combinedValues []ValueIndex) {

	if inTimePeriod < 2 {
		return maxValues, minValues, combinedValues // 如果时间段小于2，直接返回空数组
	}

	nbInitialElementNeeded := inTimePeriod - 1
	startIdx := nbInitialElementNeeded
	today := startIdx
	trailingIdx := startIdx - nbInitialElementNeeded
	highestIdx := -1
	lowestIdx := -1
	highest := 0.0
	lowest := 0.0

	// 初始化前一个最大值和最小值
	var prevMax, prevMin float64
	var firstIteration bool = true

	for today < len(inReal) {
		tmp := inReal[today]

		// 计算最高值及其索引
		if highestIdx < trailingIdx {
			highestIdx = trailingIdx
			highest = inReal[highestIdx]
			i := highestIdx + 1
			for i <= today {
				tmp = inReal[i]
				if tmp > highest {
					highestIdx = i
					highest = tmp
				}
				i++
			}
		} else if tmp >= highest {
			highestIdx = today
			highest = tmp
		}

		// 计算最低值及其索引
		if lowestIdx < trailingIdx {
			lowestIdx = trailingIdx
			lowest = inReal[lowestIdx]
			i := lowestIdx + 1
			for i <= today {
				tmp = inReal[i]
				if tmp < lowest {
					lowestIdx = i
					lowest = tmp
				}
				i++
			}
		} else if tmp <= lowest {
			lowestIdx = today
			lowest = tmp
		}

		// 存储最高值和最低值及差值
		var maxDiff, minDiff, maxMinDiff float64

		if !firstIteration {
			maxDiff = highest - prevMax    // 计算前后最大值的差值
			minDiff = lowest - prevMin     // 计算前后最小值的差值
			maxMinDiff = highest - prevMin // 计算最大值和最小值之间的差值
		}

		// 添加最大值和最小值到各自的结果数组
		maxValues = append(maxValues, ValueIndex{Value: highest, Index: highestIdx, Type: MaxType, PrevDiff: maxDiff, MaxMinDiff: maxMinDiff})
		minValues = append(minValues, ValueIndex{Value: lowest, Index: lowestIdx, Type: MinType, PrevDiff: minDiff, MaxMinDiff: maxMinDiff})

		// 将最大值和最小值交替添加到合并数组
		combinedValues = append(combinedValues, ValueIndex{Value: highest, Index: highestIdx, Type: MaxType, PrevDiff: maxDiff, MaxMinDiff: maxMinDiff})
		combinedValues = append(combinedValues, ValueIndex{Value: lowest, Index: lowestIdx, Type: MinType, PrevDiff: minDiff, MaxMinDiff: maxMinDiff})

		// 更新前一个最大值和最小值
		prevMax = highest
		prevMin = lowest
		firstIteration = false

		today++
		trailingIdx++
	}

	return maxValues, minValues, combinedValues // 返回最高值、最低值和合并列表
}

func getVals(combinedValues []ValueIndex) (maxV *ValueIndex, minV *ValueIndex, firstIsMax bool) {
	cLen := len(combinedValues)
	if cLen == 0 {
		return
	}
	if cLen == 1 {
		tmpV := &(combinedValues[0])
		if tmpV.Type == MaxType {
			maxV = tmpV
			firstIsMax = true
		} else {
			minV = tmpV
		}
		return
	}
	tmpV := &(combinedValues[cLen-1])
	if tmpV.Type == MaxType {
		firstIsMax = true
		maxV = tmpV
		minV = &(combinedValues[cLen-2])
	} else {
		minV = tmpV
		maxV = &(combinedValues[cLen-2])
	}
	return
}

func pushVal(maxV *ValueIndex, minV *ValueIndex, combinedValues []ValueIndex) []ValueIndex {
	if maxV == nil && minV == nil {
		return combinedValues
	}
	cLen := len(combinedValues)
	preMax, preMin, firstIsMax := getVals(combinedValues)
	//[],[max1],[min1],[...max1,min1,max2,min2,...maxN,minN]
	if maxV == nil && minV != nil {
		if cLen == 0 {
			combinedValues = append(combinedValues, *minV)
		} else {
			if firstIsMax {
				combinedValues = append(combinedValues, *minV)
			} else {
				//上一个也是minV， 替换或遗弃
				if minV.Value <= preMin.Value {
					combinedValues = append(combinedValues[cLen-1:], *minV)
				}
			}
		}
	} else if maxV != nil && minV == nil {
		if cLen == 0 {
			combinedValues = append(combinedValues, *maxV)
		} else {
			if !firstIsMax {
				combinedValues = append(combinedValues, *maxV)
			} else {
				//上一个也是minV， 替换或遗弃
				if maxV.Value >= preMax.Value {
					combinedValues = append(combinedValues[cLen-1:], *maxV)
				}
			}
		}
	} else if maxV != nil && minV != nil {
		if cLen == 0 {
			if maxV.Index < minV.Index {
				combinedValues = append(combinedValues, *maxV)
				combinedValues = append(combinedValues, *minV)
			} else {
				combinedValues = append(combinedValues, *minV)
				combinedValues = append(combinedValues, *maxV)
			}
		} else {
			if firstIsMax { //一定有preMax
				if maxV.Index < minV.Index {//max在前

				} else {

				}
			} else { //不一定有preMax,一定有preMin
				if maxV.Index < minV.Index {

				} else {

				}
			}
		}
	}

	return combinedValues
}

// MaxMinAndIndices - 返回指定时间段内的最高值、最低值及其索引和差值
func flitterPeakAndValleys(klines []common.KLine, windowSize int) (combinedValues []ValueIndex) {
	if windowSize < 2 {
		return // 如果时间段小于2，直接返回空数组
	}
	kLen := len(klines)
	//1. 分别计算每个窗口的最大最小值
	tmpKlines := klines[:]
	for i := 1; i < kLen; i += windowSize {
		if i+windowSize >= kLen { //边界检查
			windowSize = kLen - i - 1
		}

		// endIdx := int(math.Min(float64(kLen-1), float64(i+windowSize)))
		logger.Infof("size: %d, window start index: %d, window size: %d", kLen, i-1, windowSize)
		windowKlines := tmpKlines[i-1 : i+windowSize]
		//2. 获取当前窗口的最大最小值(波峰波谷)
		maxV, minV := flitterPeakAndValley(windowKlines, 1.01)
		if maxV == nil && minV == nil {
			continue
		}
		combinedValues = pushVal(maxV, minV, combinedValues)
	}
	return
}

func flitterPeakAndValley(klines []common.KLine, toleranceRate float64) (maxValue *ValueIndex, minValue *ValueIndex) {
	if len(klines) < 3 {
		return nil, nil
	}
	kLen := len(klines)
	initK := klines[kLen-1]
	maxV := initK.Close
	minV := initK.Close
	// maxIdx := 0
	// minIdx := 0

	//分别计算每个窗口的最大最小值
	for i := 1; i < kLen-1; i++ {
		prev := klines[i-1]
		curr := klines[i]
		next := klines[i+1]
		//获取最大值
		if curr.High > maxV {
			maxV = curr.High
			// maxIdx = i
		}
		//获取最小
		if curr.Low < minV {
			minV = curr.Low
			// minIdx = i
		}

		//获取波峰
		if curr.High > prev.High && curr.High > next.High {
			// if curr.Close > maxValue.Value*(1+toleranceRate) {
			maxValue = &ValueIndex{Value: curr.High, Index: i, Type: "max", Kline: curr}
			// }
		}
		//获取波谷
		if curr.Low < prev.Low && curr.Low < next.Low {
			// if curr.Close < minValue.Value*(1-toleranceRate) {
			minValue = &ValueIndex{Value: curr.Low, Index: i, Type: "min", Kline: curr}
			// }
		}
	}
	return
}

// 判断能否作为波谷或者波峰插入到结果数组中
// 前面有波峰，当前波峰大于前面波峰，或者前面有波谷，当前波峰小于前面波谷
func inputPeakOrValley(preMaxV *ValueIndex, preMinV *ValueIndex, curMaxV *ValueIndex, curMinV *ValueIndex, diffRate float64, isPeak bool) (can bool) {
	if isPeak {

	} else {

	}
	return true
}

// 比较两个波峰或者波谷
func cmp(preMaxV *ValueIndex, preMinV *ValueIndex, curMaxV *ValueIndex, curMinV *ValueIndex, diffRate float64, isPeak bool) bool {

	return true
}
