package main

import (
	"encoding/json"
	"fmt"
	"freefi/trademgr/ordermgr"
	"freefi/trademgr/pkg/logger"
	"strings"

	"github.com/robfig/cron/v3"
)

type ITradeMgr interface {
	ExecuteTrade(sugMsg []byte) error
	AddOrderMgr(symbol string, orderMgr ordermgr.IOrderMgr) error
	RemoveOrderMgr(symbol string) error
	CronExecute()
}

type TradeMgr struct {
	//symbol=>orders
	orderMgrs map[string]ordermgr.IOrderMgr
}

// CronExecute implements ITradeMgr.
func (tm *TradeMgr) CronExecute() {
	loopInterval := fmt.Sprintf("@every %dm", 5)
	c := cron.New()
	c.AddFunc(loopInterval, tm.loopCheck)
	c.Start()
	defer c.Stop()
	select {}
}

// AddOrderMgr implements ITradeMgr.
func (tm *TradeMgr) AddOrderMgr(symbol string, orderMgr ordermgr.IOrderMgr) error {
	if tm.orderMgrs[symbol] == nil {
		tm.orderMgrs[symbol] = orderMgr
		logger.Info("TradeMgr.AddOrderMgr: added order manager for symbol: ", symbol)
	}
	return nil
}

// RemoveOrderMgr implements ITradeMgr.
func (tm *TradeMgr) RemoveOrderMgr(symbol string) error {
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

// LoopCheck implements ITradeMgr.
func (tm *TradeMgr) loopCheck() {
	for symbol, orderMgr := range tm.orderMgrs {
		if orderMgr.IsDone() {
			tm.RemoveOrderMgr(symbol)
		}
	}
	logger.Info("TradeMgr.loopCheck: executed")
}

func NewTradeMgr() ITradeMgr {
	return &TradeMgr{
		orderMgrs: make(map[string]ordermgr.IOrderMgr),
	}
}

func (tm *TradeMgr) ExecuteTrade(sugMsg []byte) error {
	// TODO: implement trade execution logic
	var strateMsg ordermgr.StrategyMsg
	var err error
	err = json.Unmarshal(sugMsg, &strateMsg)
	if err != nil {
		logger.Error("TradeMgr.ExecuteTrade failed to unmarshal strateMsg: ", err)
		return err
	}
	logger.Infof("TradeMgr.ExecuteTrade: received strategy message: %+v", string(sugMsg))
	symbol := strings.ToUpper(strateMsg.DataSource.Symbol)

	if tm.orderMgrs[symbol] == nil {
		iOrdMgr := ordermgr.NewOrderMgr()
		tm.AddOrderMgr(symbol, iOrdMgr)
	}
	err = tm.orderMgrs[symbol].Update(strateMsg)
	if err != nil {
		logger.Error("TradeMgr.ExecuteTrade failed: ", err)
	}
	return err
}
