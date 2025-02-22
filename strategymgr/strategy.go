package main

import (
	"encoding/json"
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
	DataSource   DataSource
	TradeSuggest common.TradeSuggest
}

type IStrategy interface {
	Work() error
	Stop() error
	UpdateParams(params StrategyParams) error
	Name() string
}

type Strategy struct {
	params    StrategyParams
	stopCh    chan bool
	kLines    map[int][]common.KLine
	isWorking bool
}

func NewStrategy(params StrategyParams) IStrategy {
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

func (s *Strategy) UpdateParams(params StrategyParams) error {
	if s.isReload(params) {
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
	// 将 interface{} 转换为 JSON
	if sRet.TradeSuggest.TradeSide != common.TradeSideNone {
		go redis.Publish(PubChannel, PubMsg{
			DataSource:   s.params.DataSource,
			TradeSuggest: sRet.TradeSuggest,
		})
		jsonData, _ := json.Marshal(sRet)
		logger.Warnf("handleFlow: %+v\n", string(jsonData))

	}

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
	if s.kLines[period] == nil || len(s.kLines[period]) < limit {
		s.kLines[period] = klines
	} else {
		kLen := len(s.kLines[period])
		// fmt.Printf("=>%+v\n", s.kLines[period][kLen-1])
		s.kLines[period] = append(s.kLines[period][:kLen-1], klines...)
		// fmt.Printf("==>%+v\n", s.kLines[period][kLen-1])
	}
	return s.kLines[period], nil
}

func (s *Strategy) execute(limit int) (sRet *StrategyRet, err error) {
	logger.Infof("Executing %s with params: %d\n", s.params.Name, len(s.params.GroupStrateies))
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

func checkParams(params StrategyParams) error {
	if len(params.GroupStrateies) > 2 {
		return fmt.Errorf("GroupStrateies length should be less than 2")
	}
	return nil
}
