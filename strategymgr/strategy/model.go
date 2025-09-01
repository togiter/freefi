package strategy

import (
	"fmt"
	"freefi/strategymgr/common"
	"freefi/strategymgr/group_strategy"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/utils"
	"time"
)

type CombineType int

const (
	Normalized CombineType = iota
	TrendOnly
	ReversalOnly
	ReversalAndTrend
	ReversalAndMacd
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
	defer func() {
		s.TradeSuggest.CreateTime = time.Now().Unix()
	}()
	s.MakeGroups()
}

func (s *StrategyRet) MakeGroups() {
	gLen := len(s.GroupStrategyRets)
	if gLen == 0 {
		logger.Warnf("策略%s 没有策略组??", s.Params.Name)
		return
	}
	mainG, helpG := getMainAndHelpK(s.GroupStrategyRets)

	//只有一个策略组
	if helpG == nil {
		s.TradeSuggest = mainG.TradeSuggest
		return
	}
	if mainG.Params.Required && mainG.TradeSuggest.TradeSide == common.TradeSideNone {
		s.TradeSuggest = mainG.TradeSuggest
		return
	}
	if helpG.Params.Required && helpG.TradeSuggest.TradeSide == common.TradeSideNone {
		s.TradeSuggest = helpG.TradeSuggest
		return
	}
	//独裁
	if mainG.Params.IsDictatorship && mainG.TradeSuggest.TradeSide != common.TradeSideNone {
		s.TradeSuggest = mainG.TradeSuggest
		return
	}
	if helpG.Params.IsDictatorship && helpG.TradeSuggest.TradeSide != common.TradeSideNone {
		s.TradeSuggest = helpG.TradeSuggest
		return
	}
	//有两个策略组
	if (mainG.TradeSuggest.TradeSide == common.TradeSideLong && helpG.TradeSuggest.TradeSide == common.TradeSideLong) ||
		(mainG.TradeSuggest.TradeSide == common.TradeSideShort && helpG.TradeSuggest.TradeSide == common.TradeSideShort) {
		s.TradeSuggest = mainG.TradeSuggest
		logger.Infof("策略%s 联合%v", s.Params.Name, mainG.TradeSuggest.TradeSide)
	}

	switch s.Params.Combine {
	case int(TrendOnly):
		s.suggestByTrendOnly(mainG, helpG)
		return
	case int(ReversalAndTrend):
		s.suggestByTrendAndReversal(mainG, helpG)
		return
	case int(ReversalOnly):
		s.suggestByReversalOnly(mainG, helpG)
		return
	case int(ReversalAndMacd):
		s.suggestByMacdTrendAndReversal(mainG, helpG)
		return
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
	bigGroup := s[bigPeriod]
	smallGroup := s[smallPeriod]
	return bigGroup, smallGroup
}

func (s *StrategyRet) suggestByTrendOnly(main *group_strategy.GroupStrategyRet, help *group_strategy.GroupStrategyRet) {
	if main == nil || help == nil {
		return
	}
	if main.TradeSuggest.TradeSide != common.TradeSideNone && help.TradeSuggest.TradeSide == main.TradeSuggest.TradeSide {
		//A.Side = LONG / SHORT & B.Side = LONG / SHORT  T 1,2
		s.TradeSuggest.TradeSide = main.TradeSuggest.TradeSide
		s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v => %v", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, main.TradeSuggest.TradeSide)
		return
	}
	mMacd := main.MicroStrategyRets["macd"]
	hMacd := help.MicroStrategyRets["macd"]
	if mMacd == nil || hMacd == nil {
		return
	}
	//方向
	mDir := mMacd.Opts["dir"]
	hDir := hMacd.Opts["dir"]

	if mMacd.TradeSuggest.TradeSide == common.TradeSideLong {
		// if hMacd.TradeSuggest.TradeSide == common.TradeSideShort {
		// 	//A.Side = LONG & B.Side = SHORT 平仓 C 1
		// 	s.TradeSuggest.CloseSide = common.TradeSideShort
		// 	s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => SHORT", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
		// 	return
		// }
		if hDir != nil && utils.ToInt64(hDir) == 1 {
			//A.Side = LONG & B.Dir = 1 ; T 3
			s.TradeSuggest.TradeSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => LONG", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			return
		}
	} else if mMacd.TradeSuggest.TradeSide == common.TradeSideShort {
		// if hMacd.TradeSuggest.TradeSide == common.TradeSideLong {
		// 	//A.Side = SHORT & B.Side = LONG 平仓 C2
		// 	s.TradeSuggest.CloseSide = common.TradeSideLong
		// 	s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => LONG(close)", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
		// 	return
		// }
		if hDir != nil && utils.ToInt64(hDir) == -1 {
			//A.Side = SHORT & B.Dir = -1 ; T4
			s.TradeSuggest.TradeSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => SHORT", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			return
		}
	} else if mMacd.TradeSuggest.TradeSide == common.TradeSideNone {
		if mDir != nil && utils.ToInt64(mDir) == 1 && hMacd.TradeSuggest.TradeSide == common.TradeSideLong {
			// T5
			s.TradeSuggest.TradeSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => LONG", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			return
		}
		if mDir != nil && utils.ToInt64(mDir) == -1 && hMacd.TradeSuggest.TradeSide == common.TradeSideShort {
			// T6
			s.TradeSuggest.TradeSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => SHORT", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			return
		}

	}
	if mDir != nil && hDir != nil {
		mD := utils.ToInt64(mDir)
		hD := utils.ToInt64(hDir)
		if mD == 1 && hD == 1 {
			// C3
			s.TradeSuggest.CloseSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => LONG(close)", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)

		} else if mD == -1 && hD == -1 {
			// C4
			s.TradeSuggest.CloseSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v(dir: %v) => SHORT(close)", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, mDir)

		}
	}
}
func (s *StrategyRet) suggestByReversalOnly(main *group_strategy.GroupStrategyRet, help *group_strategy.GroupStrategyRet) {
	if main == nil || help == nil {
		return
	}
	if main.TradeSuggest.TradeSide == common.TradeSideLong {
		if help.TradeSuggest.TradeSide != common.TradeSideShort {
			//m.Side = LONG & h.Side == NONE or LONG T1
			s.TradeSuggest = main.TradeSuggest
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v=>LONG", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide)

			return
		} else {
			// close
			s.TradeSuggest.CloseSide = help.TradeSuggest.TradeSide
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v=>%v(close%v)", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, help.TradeSuggest.TradeSide, main.TradeSuggest.TradeSide)

			return
		}

	} else if main.TradeSuggest.TradeSide == common.TradeSideShort {
		if help.TradeSuggest.TradeSide != common.TradeSideLong {
			//m.Side = SHORT & h.Side == NONE or SHORT T1
			s.TradeSuggest = main.TradeSuggest
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v =>SHORT", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide)

			return
		} else {
			// close
			s.TradeSuggest.CloseSide = help.TradeSuggest.TradeSide
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v => %v(close%v)", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, help.TradeSuggest.TradeSide, main.TradeSuggest.TradeSide)

			return
		}
	} else {
		s.TradeSuggest.CloseSide = help.TradeSuggest.TradeSide
		s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v && %v->%v => %v(close)", help.Params.Name, help.TradeSuggest.TradeSide, main.Params.Name, main.TradeSuggest.TradeSide, help.TradeSuggest.TradeSide)
	}
}

func (s *StrategyRet) suggestByTrendAndReversal(main *group_strategy.GroupStrategyRet, help *group_strategy.GroupStrategyRet) {
	if main == nil || help == nil {
		return
	}
	mMacd := main.MicroStrategyRets["macd"]
	hMacd := help.MicroStrategyRets["macd"]
	var mDir interface{}
	var hDir interface{}
	if mMacd != nil {
		//方向
		mDir = mMacd.Opts["dir"]
	}
	if hMacd != nil {
		hDir = hMacd.Opts["dir"]
	}

	if help.TradeSuggest.TradeSide == common.TradeSideLong { //help决策线,main是控盘线
		if main.TradeSuggest.TradeSide == common.TradeSideLong ||
			(mDir != nil && utils.ToInt64(mDir) == 1) {
			s.TradeSuggest.TradeSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v) => LONG", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			return
		}
		if main.TradeSuggest.TradeSide == common.TradeSideShort {
			// || (mDir != nil && utils.ToInt64(mDir) == -1) {
			s.TradeSuggest.CloseSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v)=>LONG(close)", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)

			return
		}
	} else if help.TradeSuggest.TradeSide == common.TradeSideShort {
		if main.TradeSuggest.TradeSide == common.TradeSideShort ||
			(mDir != nil && utils.ToInt64(mDir) == -1) {
			s.TradeSuggest.TradeSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v) => SHORT", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			return
		}
		if main.TradeSuggest.TradeSide == common.TradeSideLong {
			//||(mDir != nil && utils.ToInt64(mDir) == 1) {
			s.TradeSuggest.CloseSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v)=>SHORT(close)", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)

			return
		}
	}
}

func (s *StrategyRet) suggestByMacdTrendAndReversal(main *group_strategy.GroupStrategyRet, help *group_strategy.GroupStrategyRet) {
	if main == nil || help == nil {
		return
	}
	mMacd := main.MicroStrategyRets["macd"]
	hMacd := help.MicroStrategyRets["macd"]
	var mDir interface{}
	var hDir interface{}
	if mMacd != nil {
		//方向
		mDir = mMacd.Opts["dir"]
	}
	if hMacd != nil {
		hDir = hMacd.Opts["dir"]
	}

	if help.TradeSuggest.TradeSide == common.TradeSideLong { //help决策线,main是控盘线
		if main.TradeSuggest.TradeSide != common.TradeSideShort {
			s.TradeSuggest.TradeSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v) => LONG", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			if mDir != nil && utils.ToInt64(mDir) == -1 {
				s.TradeSuggest.FomoLevel = -1
			}
			return
		}
		if main.TradeSuggest.TradeSide == common.TradeSideShort {
			// || (mDir != nil && utils.ToInt64(mDir) == -1) {
			s.TradeSuggest.CloseSide = common.TradeSideLong
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v)=>LONG(close)", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)

			return
		}
	} else if help.TradeSuggest.TradeSide == common.TradeSideShort {
		if main.TradeSuggest.TradeSide != common.TradeSideLong {
			s.TradeSuggest.TradeSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v) => SHORT", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)
			if mDir != nil && utils.ToInt64(mDir) == 1 {
				s.TradeSuggest.FomoLevel = -1
			}
			return
		}
		if main.TradeSuggest.TradeSide == common.TradeSideLong {
			//||(mDir != nil && utils.ToInt64(mDir) == 1) {
			s.TradeSuggest.CloseSide = common.TradeSideShort
			s.TradeSuggest.Mark = fmt.Sprintf("%v ->%v(dir: %v) && %v->%v(dir: %v)=>SHORT(close)", help.Params.Name, help.TradeSuggest.TradeSide, hDir, main.Params.Name, main.TradeSuggest.TradeSide, mDir)

			return
		}
	}
}
