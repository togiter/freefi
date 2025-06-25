package group_strategy

import (
	common "freefi/strategymgr/common"
	"freefi/strategymgr/group_strategy/micro_strategy"
	"freefi/strategymgr/pkg/logger"
	"math"
	"strings"
	"time"
)

type GroupStrategyParams struct {
	Name               string            `json:"name" yaml:"name"`
	Status             int               `json:"status" yaml:"status"` //0正常  1，禁用
	Type               int               `json:"type" yaml:"type"`     //  策略类型 0 主策，1辅策
	PassRate           float64           `json:"passRate" yaml:"yaml"` //  目标通过率
	CloseRate          float64           `json:"closeRate" yaml:"closeRate"`
	Delays             map[string]string `json:"delays" yaml:"delays"`
	Required           bool              `json:"required" yaml:"required"`
	IgnoreOpposition   bool              `json:"ignoreOpposition" yaml:"ignoreOpposition"`
	IsDictatorship     bool              `json:"isDictatorship" yaml:"isDictatorship"`
	FomoLevel          int               `json:"fomoLevel" yaml:"fomoLevel"` //交易等级/none,pre,pro
	VolatilityKlineNum int               `json:"volatilityKlineNum" yaml:"volatilityKlineNum"`
	// Ext              *GroupStrategyExt                             `json:"ext"`
	MicroStrategies map[string]micro_strategy.MicroStrategyParams `json:"microStrategies" yaml:"microStrategies"` //  微策略参数
}

type GroupStrategyRet struct {
	Params            *GroupStrategyParams                        `json:"params"`            // 策略参数
	TradeSuggest      common.TradeSuggest                         `json:"tradeSuggest"`      // 交易建议
	MicroStrategyRets map[string]*micro_strategy.MicroStrategyRet `json:"microStrategyRets"` // name->微策略结果
	Opts              map[string]interface{}                      `json:"opts"`
}

// SuggestForBuild 建仓策略
// 如果满足独裁策略，直接返回独裁，否则:
// 如果必选条件不满足，返回不满足；否则：
// 如果满足百分比，返回交易建议；否则：
// 返回其他特定条件
//
// SuggestForBuild 建仓策略
// 如果满足独裁策略，直接返回独裁，否则:
// 如果必选条件不满足，返回不满足；否则：
// 如果满足百分比，返回交易建议；否则：
// 返回其他特定条件

// SuggestForClose 平仓策略
// 如果满足建仓策略，直接返回，否则:
// 如果满足指定百分比，返回平仓建议；否则：
// 如果满足特定微策略条件，返回指定微策略

func (gsr *GroupStrategyRet) GetMicroStrateRet(name string) *micro_strategy.MicroStrategyRet {
	msr, ok := gsr.MicroStrategyRets[strings.ToLower(name)]
	if !ok {
		return nil
	}
	return msr
}

// func (gsr *GroupStrategyRet) DelaySuggest(name string, preS common.TradeSuggest) (sg common.TradeSuggest) {
// 	startTime := preS.CreateTime
// 	if startTime == 0 || len(gsr.Params.DelayMicroStrate) == 0 {
// 		return
// 	}
// 	kPeriod := gsr.Params.KPeriod
// 	delayX := gsr.Params.DelayX
// 	if delayX <= 0.000001 {
// 		delayX = 2.5
// 	}
// 	startDelayTime := startTime + utils.ToInt64(delayX*float64(kPeriod*60))
// 	//TODO: 暂设为和延时等待一样
// 	endDelayTime := startTime + utils.ToInt64(10.0*delayX*float64(kPeriod*60))
// 	curTime := time.Now().Unix()
// 	//在时间段【startDelayTime，endDelayTime】提供延时确认
// 	if curTime < startDelayTime || curTime > endDelayTime {
// 		logger.Warnf("策略组(%s)当前时间(%s)不在延时范围[%s,%s]", gsr.Params.Name, utils.TimeFmt(curTime, ""), utils.TimeFmt(startDelayTime, ""), utils.TimeFmt(endDelayTime, ""))
// 		return
// 	}

// 	msRetp := gsr.GetMicroStrateRet(name)
// 	//如果没有之前的下单策略，就没有延迟确认的说法，直接返回本策略组的策略
// 	if msRetp == nil {
// 		return
// 	}
// 	//获取延时策略结果
// 	delayStrateRet := *msRetp
// 	sg = delayStrateRet.TradeSuggest
// 	txp := delayStrateRet.TradeSuggest.StrictFomoLevel(gsr.Params.FomoLevel)
// 	sg.TradeSide = txp
// 	sg.CreateTime = time.Now().Unix()
// 	return

// }

func (gsr GroupStrategyRet) IsRequirePassed() bool {
	return !gsr.Params.Required || (gsr.Params.Required && gsr.TradeSuggest.TradeSide != common.TradeSideNone)
}

func (gsr *GroupStrategyRet) MakeFinalTrade() {
	defer func() {
		gsr.TradeSuggest.CreateTime = time.Now().Unix()
		if gsr.TradeSuggest.TradeSide != common.TradeSideNone {
			logger.Infof("策略组(%s - %s)最终交易建议(%s)", gsr.Params.Name, gsr.TradeSuggest.TradeSide)
		}
	}()
	//计算百分比
	//做空做多策略数
	shortCount := 0
	longCount := 0
	mSLen := len(gsr.MicroStrategyRets)
	passRate := math.Max(gsr.Params.PassRate, 0.5)
	requiredNotPass := false
	for _, msr := range gsr.MicroStrategyRets {
		if mSLen == 1 {
			gsr.TradeSuggest = msr.TradeSuggest
			return
		}
		is, suggest := msr.DiactorShipSuggest()
		if is && suggest.TradeSide != common.TradeSideNone {
			logger.Infof("策略组(%s)独裁策略(%s-%s)满足", gsr.Params.Name, msr.Params.Name, suggest.TradeSide)
			gsr.TradeSuggest = suggest
			return
		}
		if !msr.IsRequirePassed() {
			requiredNotPass = true
			continue
		}

		//过滤等级,//方向上的同向化过滤
		fomo := gsr.Params.FomoLevel
		txP := msr.TradeSuggest.StrictFomoLevel(fomo)
		if txP == common.TradeSideLong {
			logger.Infof("策略组(%s)微策略(%s)建议买入", gsr.Params.Name, msr.Params.Name)
			longCount++
		} else if txP == common.TradeSideShort {
			logger.Infof("策略组(%s)微策略(%s)建议卖出", gsr.Params.Name, msr.Params.Name)
			shortCount++
		}
	}
	if requiredNotPass {
		logger.Warnf("策略组(%s)未通过必选要求", gsr.Params.Name)
		gsr.TradeSuggest.TradeSide = common.TradeSideNone
		gsr.TradeSuggest.Mark = "策略依赖的微策略未通过"
		return
	}

	finalP := common.TradeSideNone
	if float64(shortCount)/float64(mSLen) >= passRate {
		finalP = common.TradeSideShort
		logger.Infof("策略组(%s)通过率(%.2f)满足卖出", gsr.Params.Name, passRate)
	} else if float64(longCount)/float64(mSLen) >= passRate {
		finalP = common.TradeSideLong
		logger.Infof("策略组(%s)通过率(%.2f)满足买入", gsr.Params.Name, passRate)

	}
	gsr.TradeSuggest.TradeSide = finalP
}
