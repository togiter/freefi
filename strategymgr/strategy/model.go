package strategy

import (
	"freefi/strategymgr/common"
	"freefi/strategymgr/group_strategy"
	"freefi/strategymgr/pkg/logger"
)

const (
	MAIN_ONLY   = 1
	MAIN1_ONLY  = 2
	CONTR_ONLY  = 3
	CONTR1_ONLY = 4
)

type DataSource struct {
	Exchange string `json:"exchange" yaml:"exchange"`
	Symbol   string `json:"symbol" yaml:"symbol"`
	Market   string `json:"market" yaml:"market"`
	Limit    int    `json:"limit" yaml:"limit"`
	Ticker   int    `json:"ticker" yaml:"ticker"`
}

type StrategyParams struct {
	Name             string                                        `json:"name" yaml:"name"`
	Status           int                                           `json:"status" yaml:"status"`
	DataSource       DataSource                                    `json:"dataSource" yaml:"dataSource"`
	GroupStrateies   map[string]group_strategy.GroupStrategyParams `json:"groupStrategies" yaml:"groupStrategies"`
	Combine          int                                           `json:"combine" yaml:"combine"` //联合模式
	PassRate         float64                                       `json:"passRate" yaml:"passRate"`
	IgnoreOpposition bool                                          `json:"ignoreOpposition" yaml:"ignoreOpposition"`

}

type StrategyRet struct {
	Combine           int                                        `json:"combine"` //联合模式
	Params            *StrategyParams                            `json:"params"`
	TradeSuggest      common.TradeSuggest                        `json:"tradeSuggest"`
	GroupStrategyRets map[int64]*group_strategy.GroupStrategyRet `json:"groupStrategyRets"`
}

func (s *StrategyRet) MakeFinalTrade() {
	s.MakeGroups()
}

func (s *StrategyRet) MakeGroups() {
	gLen := len(s.GroupStrategyRets)
	if gLen == 0 {
		logger.Warnf("策略%s 没有策略组??", s.Params.Name)
		return
	}
	mainG, helpG := getMainAndHelpK(s.GroupStrategyRets)
	logger.Warnf("main %v,side %v", mainG.Params.Name, mainG.TradeSuggest.TradeSide)

	//只有一个策略组
	if helpG == nil {
		s.TradeSuggest = mainG.TradeSuggest
		return
	}
	logger.Warnf("help %v,side %v", helpG.Params.Name, helpG.TradeSuggest.TradeSide)

	//有两个策略组
	if (mainG.TradeSuggest.TradeSide == common.TradeSideLong && helpG.TradeSuggest.TradeSide == common.TradeSideLong) ||
		(mainG.TradeSuggest.TradeSide == common.TradeSideShort && helpG.TradeSuggest.TradeSide == common.TradeSideShort) {
		s.TradeSuggest = mainG.TradeSuggest
		logger.Infof("策略%s 联合买入/卖出", s.Params.Name)
	} else if s.Params.Combine == 1 { //二选一
		if mainG.TradeSuggest.TradeSide != common.TradeSideNone && helpG.TradeSuggest.TradeSide == common.TradeSideNone {
			s.TradeSuggest = mainG.TradeSuggest
			logger.Infof("策略%s - %s %s", s.Params.Name, mainG.Params.Name, s.TradeSuggest.TradeSide)
		} else if mainG.TradeSuggest.TradeSide == common.TradeSideNone && helpG.TradeSuggest.TradeSide != common.TradeSideNone {
			s.TradeSuggest = helpG.TradeSuggest
			logger.Infof("策略%s - %s %s", s.Params.Name, helpG.Params.Name, s.TradeSuggest.TradeSide)
		}
	}
}

func getMainAndHelpK(s map[int64]*group_strategy.GroupStrategyRet) (*group_strategy.GroupStrategyRet, *group_strategy.GroupStrategyRet) {
	bigPeriod := int64(0)
	smallPeriod := int64(1000000000)
	gLen := len(s)
	for period, gs := range s {
		gs.MakeFinalTrade()
		if gLen == 1 || (gs.TradeSuggest.TradeSide != common.TradeSideNone && gs.Params.IsDictatorship) {
			return gs, nil
		}
		if !gs.IsRequirePassed() {
			logger.Warnf("策略%s 组%s 必选条件未满足", gs.Params.Name, gs.Params.Name)
			gs.TradeSuggest.TradeSide = common.TradeSideNone
			return gs, nil
		}

		if period > bigPeriod {
			bigPeriod = period
		}
		if period < smallPeriod {
			smallPeriod = period
		}
	}
	logger.Warnf("big %v small %v", bigPeriod, smallPeriod)

	bigGroup := s[bigPeriod]
	smallGroup := s[smallPeriod]
	return bigGroup, smallGroup
}
