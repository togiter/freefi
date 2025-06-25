package strategy

import (
	"fmt"
	"freefi/strategymgr/common"
	"freefi/strategymgr/datamgr"
	"freefi/strategymgr/group_strategy"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/redis"
	"freefi/strategymgr/pkg/utils"
	"time"
)

const (
	PubChannel = "trade_suggest_channel"
)

type PubMsg struct {
	DataSource  DataSource
	StrategyRet StrategyRet
}

type IStrategy interface {
	Work() error
	Stop() error
	UpdateParams(params *StrategyParams) error
	Name() string
}

type Strategy struct {
	params    *StrategyParams
	stopCh    chan bool
	kLines    map[int][]common.KLine
	isWorking bool
}

func NewStrategy(params *StrategyParams) IStrategy {
	return &Strategy{
		params:    params,
		isWorking: false,
		stopCh:    make(chan bool),
		kLines:    make(map[int][]common.KLine),
	}
}
func (s *Strategy) Name() string {
	return s.params.Name
}

func (s *Strategy) Stop() error {
	s.stopCh <- true
	s.isWorking = false
	return nil
}

func (s *Strategy) isReload(params StrategyParams) bool {
	//1. ticker
	//2. groupStrategies
	//3. symbol
	//4. market
	if params.DataSource.Ticker != s.params.DataSource.Ticker ||
		len(params.GroupStrateies) != len(s.params.GroupStrateies) ||
		params.DataSource.Symbol != s.params.DataSource.Symbol ||
		params.DataSource.Market != s.params.DataSource.Market {
		return true
	}

	for i, gs := range params.GroupStrateies {
		//5. 策略组的前后状态
		if gs.Status != s.params.GroupStrateies[i].Status {
			return true
		}
		//6. k线
		period := int(utils.ToInt64(i))
		if s.kLines[period] == nil || len(s.kLines[period]) == 0 {
			return true
		}
	}
	return false
}

func (s *Strategy) UpdateParams(params *StrategyParams) error {
	if s.isReload(*params) {
		logger.Infof("%v strategy ReWorking", params.Name)
		s.Stop()
		go s.Work()
	} else {
		s.handleFlow(params.DataSource.Limit)
	}
	s.params = params
	return nil
}

func (s *Strategy) Work() error {
	if s.isWorking {
		logger.Warnf("%v is already working", s.params.Name)
		return fmt.Errorf("%v is already working", s.params.Name)
	}
	logger.Infof("%v Strategy Working...", s.params.Name)

	s.kLines = make(map[int][]common.KLine)
	s.isWorking = true
	s.handleFlow(s.params.DataSource.Limit)
	for {
		select {
		case <-s.stopCh:
			s.isWorking = false
			logger.Infof("%v Strategy Stop Success!!", s.params.Name)
			return nil
		default:
			time.Sleep(time.Duration(s.params.DataSource.Ticker) * time.Second)
			s.handleFlow(1) // 每次处理1条最新数据
		}
	}
}

func (s *Strategy) handleFlow(dataLimit int) {
	sRet, err := s.execute(dataLimit)
	if err != nil {
		logger.Warnf("Error in handleFlow: ", err)
		return
	}
	sRet.MakeFinalTrade()
	pubMsg(*sRet)
}

func pubMsg(msg StrategyRet) {
	// msg.Params = nil
	// for _, groupStrategyRet := range msg.GroupStrategyRets {
	// 	groupStrategyRet.Params = nil
	// 	for _, microStrategyRet := range groupStrategyRet.MicroStrategyRets {
	// 		microStrategyRet.Params = nil
	// 	}
	// }
	go redis.Publish(PubChannel, PubMsg{
		DataSource:  msg.Params.DataSource,
		StrategyRet: msg,
	})

}

func (s *Strategy) handleData(period int, limit int) ([]common.KLine, error) {
	// 调用数据接口获取数据
	params := s.params
	if limit == 0 {
		limit = params.DataSource.Limit
	}
	dataObj := datamgr.NewDataMgr()
	dataParams := datamgr.KLineParams{
		Base: datamgr.Base{
			Market:   params.DataSource.Market,
			Symbol:   params.DataSource.Symbol,
			Exchange: params.DataSource.Exchange,
			Limit:    limit,
			Period:   period,
		},
	}
	klines, err := dataObj.GetKLines(dataParams)
	if err != nil {
		return nil, err
	}
	if limit == params.DataSource.Limit {
		s.kLines[period] = klines
		logger.Infof("%v Strategy 首次请求 kLines: %d", s.params.Name, len(s.kLines[period]))
	} else {
		kLen := len(s.kLines[period])
		if s.kLines[period][kLen-1].OpenTime == klines[limit-1].OpenTime {
			logger.Infof("更新k线(openTime: %d,%d): closeTime(%d,%d)", s.kLines[period][kLen-1].OpenTime, klines[limit-1].OpenTime, s.kLines[period][kLen-1].CloseTime, klines[limit-1].CloseTime)

			//更新k线
			s.kLines[period] = append(s.kLines[period][:kLen-limit], klines...)

		} else {
			logger.Infof("替换k线(openTime: %d,%d): closeTime(%d,%d)", s.kLines[period][kLen-1].OpenTime, klines[limit-1].OpenTime, s.kLines[period][kLen-1].CloseTime, klines[limit-1].CloseTime)

			//替换k线
			s.kLines[period] = append(s.kLines[period][limit:], klines...)

		}
	}
	return s.kLines[period], nil
}

func (s *Strategy) execute(limit int) (sRet *StrategyRet, err error) {
	sRet = &StrategyRet{
		GroupStrategyRets: make(map[int64]*group_strategy.GroupStrategyRet),
	}
	sRet.Params = s.params
	for period, groupStrategyParams := range s.params.GroupStrateies {
		// 调用数据接口获取数据

		data, err1 := s.handleData(int(utils.ToInt64(period)), limit)
		if err1 != nil {
			err = fmt.Errorf("execute group strategy  GetKLines error %v", err1)
			return
		}
		groupRet, err1 := group_strategy.Execute(data, groupStrategyParams)
		if err1 != nil {
			err = fmt.Errorf("group_strategy.Execute error:%v", err1)
			return
		}
		if groupRet != nil {
			sRet.GroupStrategyRets[utils.ToInt64(period)] = groupRet
		}
		// // 将 interface{} 转换为 JSON
		// jsonData, _ := json.Marshal(data)
		// fmt.Println(string(jsonData))
	}

	return
}
