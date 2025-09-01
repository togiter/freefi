package group_strategy

import (
	common "freefi/strategymgr/common"
	"freefi/strategymgr/group_strategy/micro_strategy"
	"freefi/strategymgr/pkg/logger"
	"math"
	"sync"
	"time"
)

func Execute(kLines []common.KLine, params GroupStrategyParams) (gsRet *GroupStrategyRet, err error) {
	gsRet = &GroupStrategyRet{
		MicroStrategyRets: make(map[string]*micro_strategy.MicroStrategyRet),
		Params:            &params,
	}
	kLen := len(kLines)
	if kLen == 0 {
		return
	}
	var waitGroup sync.WaitGroup
	var mu sync.Mutex
	for name, ms := range params.MicroStrategies {
		waitGroup.Add(1)
		go func(name string, ms micro_strategy.MicroStrategyParams) {
			defer waitGroup.Done()
			msRet, err := micro_strategy.Execute(kLines, ms)
			if err != nil {
				logger.Errorf("Execute %s group micro strategy %s error: %v", params.Name, name, err)
				return
			}
			logger.Infof("Execute %s group micro strategy %s => %+v", params.Name, name, msRet.TradeSuggest)
			mu.Lock()
			gsRet.MicroStrategyRets[name] = msRet
			mu.Unlock()
		}(name, ms)

	}
	waitGroup.Wait()
	gsRet.TradeSuggest.CreateTime = time.Now().Unix()
	gsRet.TradeSuggest.Price = kLines[len(kLines)-1].Close
	vNum := params.VolatilityKlineNum
	if vNum == 0 {
		vNum = 25
	}
	//附带平均波动幅度
	vKlines := kLines[kLen-vNum:]
	if gsRet.Opts == nil {
		gsRet.Opts = make(map[string]interface{})
	}
	gsRet.Opts["avgVolatility"] = getVolatility(vKlines)
	return
}

// 获取K线内的平均波动幅度
func getVolatility(klines []common.KLine) float64 {
	kLen := len(klines)
	sum := 0.0
	for _, k := range klines {
		voliK := k.Close - k.Open
		sum += math.Abs(voliK)
	}
	return sum / float64(kLen)
}
