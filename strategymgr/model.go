package main

import (
	"freefi/strategymgr/common"
	"freefi/strategymgr/group_strategy"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/utils"
)

const (
	MAIN_ONLY   = 1
	MAIN1_ONLY  = 2
	CONTR_ONLY  = 3
	CONTR1_ONLY = 4
)

type DataSource struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Market   string `json:"market"`
	Limit    int    `json:"limit"`
	Ticker   int    `json:"ticker"`
}

type StrategyParams struct {
	Name           string                                        `json:"name"`
	Status         int                                           `json:"status"`
	Description    string                                        `json:"description"`
	DataSource     DataSource                                    `json:"dataSource"`
	GroupStrateies map[string]group_strategy.GroupStrategyParams `json:"groupStrategies"`
	Combine        int                                           `json:"combine"` //联合模式
	PassRate       float64                                       `json:"passRate"`
}

type StrategyRet struct {
	Combine           int                                        `json:"combine"` //联合模式
	Params            StrategyParams                             `json:"params"`
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

	bigPeriod := int64(0)
	smallPeriod := int64(1000000000)
	for period, gs := range s.GroupStrategyRets {
		gs.MakeFinalTrade()
		if gLen == 1 || (gs.TradeSuggest.TradeSide != common.TradeSideNone && gs.Params.Ext != nil && gs.Params.Ext.IsDictatorship) {
			//独裁策略
			s.TradeSuggest = gs.TradeSuggest
			return
		}
		if !gs.IsRequireChecked() {
			logger.Warnf("策略%s 组%s 必选条件未满足", s.Params.Name, gs.Params.Name)
			return
		}

		if period > bigPeriod {
			bigPeriod = period
		}
		if period < smallPeriod {
			smallPeriod = period
		}

	}
	logger.Infof("smallPeriod:%d, bigPeriod:%d", smallPeriod, bigPeriod)
	bigGroup := s.GroupStrategyRets[bigPeriod]
	if bigGroup == nil {
		logger.Warnf("策略%s 无有效信号", s.Params.Name)
		return
	}
	bigGroup.MakeFinalTrade()
	smallGroup := s.GroupStrategyRets[smallPeriod]
	if smallGroup == nil {
		logger.Warnf("策略%s 无有效信号", s.Params.Name)
		return
	}
	smallGroup.MakeFinalTrade()
	if bigGroup.TradeSuggest.TradeSide == common.TradeSideNone {
		macdMicro := bigGroup.MicroStrategyRets["macd"]
		if macdMicro != nil {
			if macdMicro.TradeSuggest.TradeSide == common.TradeSideNone {
				if macdMicro.Opts["dir"] != nil && utils.ToInt64(macdMicro.Opts["macd"]) > 0 {
					//MACD 在上
					bigGroup.TradeSuggest.TradeSide = common.TradeSideBuy
					logger.Infof("策略%s macd 在上", s.Params.Name)
				} else {
					//MACD 在下
					bigGroup.TradeSuggest.TradeSide = common.TradeSideSell
					logger.Infof("策略%s macd 在下", s.Params.Name)
				}

			} else {
				bigGroup.TradeSuggest = macdMicro.TradeSuggest
			}
		}
	}
	//控盘线建议观看
	if bigGroup.TradeSuggest.TradeSide == common.TradeSideNone {
		s.TradeSuggest = bigGroup.TradeSuggest
		return
	}
	if bigGroup.TradeSuggest.TradeSide == common.TradeSideBuy {
		if smallGroup.TradeSuggest.TradeSide == common.TradeSideBuy {
			logger.Infof("策略%s 联合买入", s.Params.Name)
		} else {
			logger.Warnf("策略%s buy有歧义(big:%+v, small:%+v)", s.Params.Name, bigGroup.TradeSuggest, smallGroup.TradeSuggest)
			s.TradeSuggest.TradeSide = common.TradeSideNone
		}
	} else if bigGroup.TradeSuggest.TradeSide == common.TradeSideSell {
		if smallGroup.TradeSuggest.TradeSide == common.TradeSideSell {
			logger.Infof("策略%s 联合卖出", s.Params.Name)
			s.TradeSuggest = bigGroup.TradeSuggest
		} else {
			logger.Warnf("策略%s sell有歧义(big:%+v, small:%+v)", s.Params.Name, bigGroup.TradeSuggest, smallGroup.TradeSuggest)
			s.TradeSuggest.TradeSide = common.TradeSideNone
		}
	}
}
