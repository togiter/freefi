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
	maxValues, minValues, values := flitterPeakAndValleys(klines, window)
	if len(maxValues) < 2 || len(minValues) < 2 {
		return ret, nil
	}
	for i := len(values) - 1; i >= 0; i-- {
		kine := klines[values[i].Index]
		logger.Infof("%v combine kine Close=%v,time(%d): %v", values[i].Type, kine.Close, kine.CloseTime, utils.TimeFmt(kine.CloseTime/1000, "2006-01-02 15:04:05"))
	}
	for i := len(maxValues) - 1; i >= 0; i-- {
		kine := maxValues[i].Kline
		logger.Infof("%v maxValues kine Close=%v,time(%d): %v", values[i].Type, kine.Close, kine.CloseTime, utils.TimeFmt(kine.CloseTime/1000, "2006-01-02 15:04:05"))
	}
	for i := len(minValues) - 1; i >= 0; i-- {
		kine := minValues[i].Kline
		logger.Infof("%v minValues kine Close=%v,time(%d): %v", values[i].Type, kine.Close, kine.CloseTime, utils.TimeFmt(kine.CloseTime/1000, "2006-01-02 15:04:05"))
	}
	// maxDiff := maxValues[0].MaxMinDiff
	// minDiff := minValues[0].MaxMinDiff
	// if maxDiff > minDiff {
	// 	tradeSide = common.TradeSideShort
	// } else if minDiff > maxDiff {
	// 	tradeSide = common.TradeSideLong
	// }
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
		maxValues = append(maxValues, ValueIndex{Value: highest, Index: highestIdx, Type: "max", PrevDiff: maxDiff, MaxMinDiff: maxMinDiff})
		minValues = append(minValues, ValueIndex{Value: lowest, Index: lowestIdx, Type: "min", PrevDiff: minDiff, MaxMinDiff: maxMinDiff})

		// 将最大值和最小值交替添加到合并数组
		combinedValues = append(combinedValues, ValueIndex{Value: highest, Index: highestIdx, Type: "max", PrevDiff: maxDiff, MaxMinDiff: maxMinDiff})
		combinedValues = append(combinedValues, ValueIndex{Value: lowest, Index: lowestIdx, Type: "min", PrevDiff: minDiff, MaxMinDiff: maxMinDiff})

		// 更新前一个最大值和最小值
		prevMax = highest
		prevMin = lowest
		firstIteration = false

		today++
		trailingIdx++
	}

	return maxValues, minValues, combinedValues // 返回最高值、最低值和合并列表
}

// MaxMinAndIndices - 返回指定时间段内的最高值、最低值及其索引和差值
func flitterPeakAndValleys(klines []common.KLine, windowSize int) (maxValues []ValueIndex, minValues []ValueIndex, combinedValues []ValueIndex) {
	if windowSize < 2 {
		return maxValues, minValues, combinedValues // 如果时间段小于2，直接返回空数组
	}
	kLen := len(klines)
	var preMaxV, preMinV *ValueIndex
	//分别计算每个窗口的最大最小值
	tmpKlines := klines[:]
	for i := 0; i < kLen; i += windowSize {
		logger.Infof("size: %d, window start index: %d, window size: %d", kLen, i, windowSize)
		windowKlines := tmpKlines[i : i+windowSize]
		//获取当前窗口的最大最小值
		maxV, minV := flitterPeakAndValley(windowKlines, 1.01)
		//1.判断有没有最大值和最小值，如果有就插入
		if maxV == nil && minV == nil {
			continue
		}

		//获取前一maxV和minV
		if len(maxValues) > 0 {
			pmax := maxValues[len(maxValues)-1]
			preMaxV = &pmax
		}
		if len(minValues) > 0 {
			pmin := minValues[len(minValues)-1]
			preMinV = &pmin
		}
		logger.Infof("maxV: (%s, %v), minV: (%s, %v)", utils.TimeFmt(maxV.Kline.OpenTime/1000, "2006-01-02 15:04:05"), maxV.Kline.Close, utils.TimeFmt(minV.Kline.OpenTime/1000, "2006-01-02 15:04:05"), minV.Kline.Close)
		if maxV != nil && minV != nil {
			if maxV.Index < minV.Index { //1.1 波峰在前面.[max,min]
				if preMaxV == nil { //1.1.1前面没有波峰 [...x,min_1]
					maxValues = append(maxValues, *maxV)
					combinedValues = append(combinedValues, *maxV) //[...x,min_1,max,min]

				} else if maxV.Value >= preMaxV.Value {
					//1.1.2 前面有波峰 [x,max_1],但是小于当前波峰。
					if preMaxV.Index > preMinV.Index {
						//替换。[x,max，min]
						maxValues = append(maxValues[len(maxValues)-1:], *maxV)
						combinedValues = append(combinedValues, *maxV)
					} else {
						//ignore
					}
				}
				//1.1.3 前面有波峰 [x,max_1],但是大于当前波峰。
			} else if maxV.Index > minV.Index { //1.2 最小值在前面[min,max]

				if preMinV == nil { //1.2.1 前面没有波谷 [...x]
					minValues = append(minValues, *minV) //[...x,min,max]
					combinedValues = append(combinedValues, *minV)
				} else if minV.Value <= preMinV.Value { //1.2.1 前面没有波谷 [...x,min]
					//1.2.2 前面有波谷 [x,min_1],但是当前波谷小于前波谷。
					if preMaxV == nil || preMinV.Index > preMaxV.Index {
						//todo:
						//替换
						minValues = append(minValues[len(minValues)-1:], *minV)
						combinedValues = append(combinedValues, *minV)

					} else {
						//ignore
					}
				}
			}
		} else if maxV != nil && minV == nil { //[max]
			if preMaxV == nil {
				maxValues = append(maxValues, *maxV) //[max]
				combinedValues = append(combinedValues, *maxV)
			} else if maxV.Value >= preMaxV.Value { //
				if preMinV == nil || preMaxV.Index > preMinV.Index {
					//前面没有波谷，或者前面波谷在前面波峰的后面并且当前。
					//替换
					maxValues = append(maxValues[len(maxValues)-1:], *maxV)
					combinedValues = append(combinedValues, *maxV)
				} else {
					//ignore
				}
			}

		} else if maxV == nil && minV != nil {
			if preMinV == nil {
				minValues = append(minValues, *minV)
				combinedValues = append(combinedValues, *minV)

			} else if minV.Value <= preMinV.Value { //更大
				if preMaxV == nil || preMinV.Index > preMaxV.Index {
					//替换
					minValues = append(minValues[len(minValues)-1:], *minV)
					combinedValues = append(combinedValues, *minV)

				} else {
					//ignore
				}
			}
		}

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
		if curr.Close > prev.Close && curr.Close > next.Close {
			// if curr.Close > maxValue.Value*(1+toleranceRate) {
			maxValue = &ValueIndex{Value: curr.Close, Index: i, Type: "max", Kline: curr}
			// }
		}
		//获取波谷
		if curr.Close < prev.Close && curr.Close < next.Close {
			// if curr.Close < minValue.Value*(1-toleranceRate) {
			minValue = &ValueIndex{Value: curr.Close, Index: i, Type: "min", Kline: curr}
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
