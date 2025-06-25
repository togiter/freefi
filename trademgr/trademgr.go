package main

import (
	"encoding/json"
	"fmt"
	"freefi/trademgr/ordermgr"
	"freefi/trademgr/pkg/logger"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const CfgPath = "config/trade.yaml"

var tradeMgr ITradeMgr

func init() {
	tradeMgr = NewTradeMgr()
}

type ITradeMgr interface {
	AddOrderMgr(symbol string, orderMgr *ordermgr.OrderMgr) error
	RemoveOrderMgr(symbol string) error
	GetTradeMgr(sym string) *ordermgr.OrderMgr
}

type TradeMgr struct {
	//symbol=>orders
	orderMgrs map[string]*ordermgr.OrderMgr
	lock      sync.RWMutex
}

// GetTradeMgrs implements ITradeMgr.
func (tm *TradeMgr) GetTradeMgr(sym string) *ordermgr.OrderMgr {
	return tm.orderMgrs[sym]
}

// AddOrderMgr implements ITradeMgr.
func (tm *TradeMgr) AddOrderMgr(symbol string, orderMgr *ordermgr.OrderMgr) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if tm.orderMgrs[symbol] == nil {
		tm.orderMgrs[symbol] = orderMgr
		logger.Info("TradeMgr.AddOrderMgr: added order manager for symbol: ", symbol)
	}
	return nil
}

// RemoveOrderMgr implements ITradeMgr.
func (tm *TradeMgr) RemoveOrderMgr(symbol string) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if tm.orderMgrs[symbol] != nil {
		err := tm.orderMgrs[symbol].Stop()
		if err != nil {
			logger.Warnf("TradeMgr-orderMgrs-Exit for symbol:%s error: %v", symbol, err)
			return err
		}
	}
	delete(tm.orderMgrs, symbol)
	logger.Info("TradeMgr.RemoveOrderMgr for symbol: ", symbol)

	return nil
}

func NewTradeMgr() ITradeMgr {
	return &TradeMgr{
		orderMgrs: make(map[string]*ordermgr.OrderMgr),
	}
}

func SubEntner(sugMsg []byte) error {
	var strateMsg ordermgr.StrategyMsg
	var err error
	err = json.Unmarshal(sugMsg, &strateMsg)
	if err != nil {
		logger.Error("TradeMgr.ExecuteTrade failed to unmarshal strateMsg: ", err)
		return err
	}
	logger.Infof("suggest : (%s,%s)=>%s", strateMsg.DataSource.Symbol, strateMsg.DataSource.Market, strateMsg.StrategyRet.TradeSuggest.TradeSide)
	symbol := strings.ToUpper(strateMsg.DataSource.Symbol)

	if tradeMgr.GetTradeMgr(symbol) == nil {
		mapParams, err := getTradeCfg()
		if err != nil {
			logger.Error("TradeMgr.ExecuteTrade failed to get config: ", err)
			return err
		}
		params := mapParams[symbol]
		if params == nil {
			logger.Error("TradeMgr.ExecuteTrade failed to get params for symbol: ", symbol)
			return fmt.Errorf("failed to get params for symbol: %s", symbol)
		}
		iOrdMgr := ordermgr.NewOrderMgr(*params)
		tradeMgr.AddOrderMgr(symbol, iOrdMgr)
		go iOrdMgr.Work()
	}
	err = tradeMgr.GetTradeMgr(symbol).Update(strateMsg)
	if err != nil {
		logger.Error("TradeMgr.ExecuteTrade failed: ", err)
	}
	return nil

}

func getTradeCfg() (map[string]*ordermgr.TradeParams, error) {
	flieContent, err := os.ReadFile(CfgPath)
	if err != nil {
		return nil, err
	}
	var cfg map[string]*ordermgr.TradeParams
	err = yaml.Unmarshal(flieContent, &cfg)
	fmt.Printf("strategy config: %+v\n", cfg)
	return cfg, err
}

func getCfg(path *string) (map[string]*ordermgr.TradeParams, error) {
	cfgPath := "./config/trade/trade.json"
	if path != nil {
		cfgPath = *path
	}
	flie, err := os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer flie.Close()
	decoder := json.NewDecoder(flie)
	cfg := map[string]*ordermgr.TradeParams{}
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
