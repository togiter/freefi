package ordermgr

import (
	"fmt"
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"sync"
	"time"
)

type IMonitor interface {
	Start(loopInterval *int) error
	Stop() error
	OrderEmpty() bool
	AddOrders(order []*accmgr.Order) error
}

type Monitor struct {
	//沉默状态(该状态不接受策略建议)
	isSilent bool
	running  bool
	done     chan bool
	orders   []*accmgr.Order
	lock     sync.RWMutex
	accMgr   accmgr.IAccMgr
}

func NewMonitor(accMgr accmgr.IAccMgr) IMonitor {
	return &Monitor{
		running: false,
		orders:  make([]*accmgr.Order, 0),
		done:    make(chan bool),
		lock:    sync.RWMutex{},
		accMgr:  accMgr,
	}
}

// 是否吃单/超时/止盈止损
func (m *Monitor) updateOrder(ord *accmgr.Order) error {
	if ord == nil {
		return fmt.Errorf("order is nil")
	}
	ords, err := m.accMgr.GetOrders(accmgr.GetOrderParams{
		BaseOrderParams: ord.BaseOrderParams,
		OrderID:         &ord.ID,
	})
	if err != nil || len(ords) == 0 {
		logger.Errorf("get order(%d) error:%v", ord.ID, err)
		return nil
	}
	ord.Status = ords[0].Status
	if ord.Status == common.OrderStatusTypeNew || ord.Status == common.OrderStatusTypePartiallyFilled {
		isCancel, _ := m.cancelOrderIfTimeout(ord)
		if isCancel {
			ord.Status = common.OrderStatusTypeCanceled
		}
	}
	return nil
}

func (m *Monitor) AddOrders(orders []*accmgr.Order) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.orders = append(m.orders, orders...)
	if !m.running {
		go m.Start(nil)
	}
	return nil
}

func (m *Monitor) OrderEmpty() bool {
	return len(m.orders) == 0
}

func (m *Monitor) Start(loopInterval *int) error {
	if m.running {
		return fmt.Errorf("order monitor is already running")
	}
	m.running = true
	logger.Infof("orders(%d) monitor starting ", len(m.orders))
	loopT := 25 * time.Second
	if loopInterval != nil && *loopInterval > 0 {
		loopT = time.Duration(*loopInterval) * time.Second
	}
	for {
		select {
		case <-m.done:
			m.running = false
			logger.Info("order monitor stopped")
			return nil
		default:
			time.Sleep(loopT)
			m.handleTicker()
		}
	}
}

func (m *Monitor) handleTicker() {
	if len(m.orders) == 0 {
		return
	}
	m.lock.RLock()
	defer m.lock.RUnlock()
	baseParams := m.orders[0].BaseOrderParams
	price, err := m.accMgr.GetPrice(baseParams)
	if err != nil {
		logger.Errorf("get(%v) price error:%v", baseParams, err)
		return
	}
	// logger.Infof("(%s,%s) price: %v", baseParams.Market, baseParams.Symbol, price)
	if price < common.MinFloatValue {
		logger.Errorf(" %v price 异常", baseParams)
		return
	}

	tmpOrds := make([]*accmgr.Order, 0)
	for _, order := range m.orders {

		err := m.updateOrder(order)
		if err != nil {
			logger.Errorf("update order(%d) error:%v", order.ID, err)
			// continue
		}
		if order.Status == common.OrderStatusTypeCanceled || order.Status == common.OrderStatusTypeExpired {
			//取消或过期
			continue
		}
		if order.Status == common.OrderStatusTypeFilled {
			if order.BaseOrderParams.Market == common.MarketSpot && order.Side == common.TradeSideShort {
				//现货卖出单 不需要止盈止损
				continue
			}
			//已经吃单，
			_, isWinOrLoss := m.winloss(price, order)
			if isWinOrLoss {
				//止盈止损成功，更新订单状态
				order.Status = common.OrderStatusTypeExpired
				logger.Infof("止盈止损成功，更新订单状态(%s,%s) price: %v,%v", baseParams.Market, baseParams.Symbol, price, order.Ext)
				continue
			}
		}
		tmpOrds = append(tmpOrds, order)
	}
	m.orders = tmpOrds
	if len(m.orders) == 0 {
		err := m.Stop()
		if err != nil {
			logger.Errorf("stop order monitor error:%v", err)
		}
		m.silent(SilentType_CloseOrder, 30.0)
	}
}

func (m *Monitor) winloss(price float64, order *accmgr.Order) (*accmgr.Order, bool) {
	if order.Ext == nil {
		return nil, false
	}
	var placeParams *accmgr.CloseOrderParams
	lossPrice := common.MinFloatValue
	winPrice := common.MinFloatValue
	if order.Ext.WinPrice != nil {

		winPrice = utils.ToFloat64(*order.Ext.WinPrice)
	}
	if order.Ext.LossPrice != nil {
		lossPrice = utils.ToFloat64(*order.Ext.LossPrice)
	}

	if order.Side == common.TradeSideLong {
		//做多，判断价格是否低于止盈价格而成交
		if (winPrice > common.MinFloatValue && price >= winPrice) ||
			(lossPrice > common.MinFloatValue && price <= lossPrice) {
			logger.Warnf("止盈止损成功，更新订单状态(%s,%s) price:(win: %.5f,loss: %.5f,cur: %.5f)", order.BaseOrderParams.Market, order.BaseOrderParams.Symbol, winPrice, lossPrice, price)
			//止损单
			//止盈单
			// side := common.TradeSideShort
			if order.BaseOrderParams.Market != common.MarketSpot {
				// side = common.TradeSideCloseBuy
			}
			placeParams = &accmgr.CloseOrderParams{
				BaseOrderParams: order.BaseOrderParams,
				PositionSide:    common.TradeSideLong,
				Qty:             &order.Qty,
			}
		}
	} else if order.Side == common.TradeSideShort {
		if order.BaseOrderParams.Market == common.MarketSpot {
			logger.Warnf("现货卖出单 不需要止损")
			return nil, false
		}
		// side := common.TradeSideCloseSell
		//做空，判断价格是否高于止损价格而成交
		if (lossPrice > common.MinFloatValue && price >= lossPrice) ||
			(winPrice > common.MinFloatValue && price <= winPrice) {
			placeParams = &accmgr.CloseOrderParams{
				BaseOrderParams: order.BaseOrderParams,
				PositionSide:    common.TradeSideShort,
				Qty:             &order.Qty,
			}
		}
	}

	if placeParams != nil {
		err := m.accMgr.CloseOrders(*placeParams)
		if err != nil {
			logger.Errorf("(%s,%s)CloseOrders order(amount: %s, price: %s) error: %v", placeParams.Market, placeParams.Symbol, placeParams.Qty, price, err)
			return nil, false
		}
		logger.Infof("(%s,%s)CloseOrders order(amount: %s, price: %s, ID: %s) 成功", placeParams.Market, placeParams.Symbol, placeParams.Qty, price, order.ID)
		return nil, true
	}
	return nil, false
}

func (m *Monitor) Stop() error {
	if m.running {
		m.done <- true
		m.running = false
		return nil
	}
	return fmt.Errorf("order monitor is not running")
}

func (m *Monitor) cancelOrderIfTimeout(order *accmgr.Order) (bool, error) {
	if order.Status == common.OrderStatusTypeFilled {
		return false, nil
	}
	if order.Ext == nil || order.Ext.Timeout == nil {
		return false, nil
	}
	timeout := *(order.Ext.Timeout)
	if timeout < time.Now().Unix() {
		err := m.accMgr.CancelOrders(accmgr.CancelOrderParams{
			BaseOrderParams: order.BaseOrderParams,
			OrderID:         &order.ID,
		})
		if err != nil {
			logger.Errorf("(%s,%s)cancel order(ID: %d) error: %v", order.BaseOrderParams.Market, order.BaseOrderParams.Symbol, order.ID, err)
			return false, err
		}
		logger.Warnf("(%s,%s)cancel order(ID: %d) timeout", order.BaseOrderParams.Market, order.BaseOrderParams.Symbol, order.ID)
		return true, nil
	}
	return false, nil
}

func (m *Monitor) silent(sType SilentType, silenceTimeX float64) {
	if silenceTimeX <= 0.0001 {
		return
	}
	m.isSilent = true

	timeAfter := time.After(time.Second * time.Duration(silenceTimeX*60.0))
	endSec := time.Now().Unix() + int64(silenceTimeX*60.0)
	logger.Infof("\n开始(%s)静默:%s,预计结束时间:%s\n", sType.String(), time.Now().Format("2006-01-02 15:04:05"), utils.TimeFmt(int64(endSec), ""))
	curTime, _ := <-timeAfter
	logger.Infof("\n结束(%s)静默:%s-%s\n", sType.String(), curTime, time.Now().Format("2006-01-02 15:04:05"))
	m.isSilent = false
}

func (m *Monitor) unsilent() {
	m.isSilent = false
}
