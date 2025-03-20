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

const (
	MinPriceValue = 0.000000001
)

//1. 检测交易对的价格变化，如果价格变化超过某个阈值，则触发回调函数。
//2. 接收来自交易所的订单状态更新，并根据订单状态更新本地订单状态。
/*
订单检测功能:
1. 轮询订单状态，判断是否已经成交(限价)
2. 判断成交订单是否已经满足止盈止损
3. 止盈止损模式包括逐单模式，全仓模式，全仓模式的止盈止损为逐单的一定百分比(eg: 60%)
4. 止盈止损后的状态，全仓模式=》重置所有状态。逐单模式，清除已完成的订单。
5. 突发干扰情况,比如突然暴涨暴跌，但是为达止盈止损的情况下的处理方式。
*/

/**
网格策略参数:
网格数量 points,相当于count下单数量
网格价格偏移: 相当于priceIncr,
网格下单数量: 相当于amountIncr

马丁策略参数:
最大下单数量 count
首次下单数量： amount
n次增量: amountIncr, 一半是上次数量的两倍

**
*/

type Callback func(ord *accmgr.Order)

type IOrderMonitor interface {
	Start() error
	Stop() error
	Clear(baseParams accmgr.BaseOrderParams) error
	OrderEmpty() bool
	AddOrder(order *accmgr.Order) error
	UpdateOrders() error
	ResetToOrders(orders []*accmgr.Order) error
}
type OrderMonitor struct {
	running bool
	done    chan bool
	orders  []*accmgr.Order
	lock    sync.RWMutex
	accMgr  accmgr.IAccMgr
}

func (om *OrderMonitor) UpdateOrders() error {
	if len(om.orders) == 0 {
		return nil
	}
	defer om.lock.Unlock()
	om.lock.Lock()
	for i, order := range om.orders {
		if order.Type == common.OrderTypeLimit {
			ord, err := om.accMgr.GetOrders(accmgr.GetOrderParams{
				BaseOrderParams: order.BaseOrderParams,
				OrderID:         &order.ID,
			})
			if err != nil || len(ord) == 0 {
				logger.Errorf("get order(%d) error:%v", order.ID, err)
				continue
			}
			if ord[0].Status != common.OrderStatusTypeCanceled && ord[0].Status != common.OrderStatusTypeExpired {
				index := i
				om.orders[index] = ord[0]
				logger.Infof("update order(%+v) error:%v", ord[0], err)
			}
		}
	}
	return nil
}
func (om *OrderMonitor) ResetToOrders(orders []*accmgr.Order) error {
	om.lock.Lock()
	defer om.lock.Unlock()
	om.orders = nil
	om.orders = orders
	if !om.running {
		go om.Start()
	}
	return nil
}

// Clear implements IOrderMonitor.
func (om *OrderMonitor) Clear(baseParams accmgr.BaseOrderParams) error {
	om.Stop()
	om.orders = make([]*accmgr.Order, 0)
	//非现货先取消挂单
	err := om.accMgr.CancelOrders(accmgr.CancelOrderParams{
		BaseOrderParams: baseParams,
	})
	if err != nil {
		logger.Errorf("cancel all orders error: %v", err)
		return err
	}

	return nil
}

func NewOrderMonitor() IOrderMonitor {
	return &OrderMonitor{
		running: false,
		done:    make(chan bool),
		orders:  make([]*accmgr.Order, 0),
		accMgr:  accmgr.NewAccMgr(),
	}
}

func (om *OrderMonitor) AddOrder(order *accmgr.Order) error {
	om.lock.Lock()
	defer om.lock.Unlock()
	om.orders = append(om.orders, order)
	if !om.running {
		go om.Start()
	}
	return nil
}

func (om *OrderMonitor) OrderEmpty() bool {
	return len(om.orders) == 0
}

func (om *OrderMonitor) Start() error {
	if om.running {
		return fmt.Errorf("order monitor is already running")
	}
	om.running = true
	logger.Infof("orders(%d) monitor starting ", len(om.orders))
	for {
		select {
		case <-om.done:
			om.running = false
			logger.Info("order monitor stopped")
			return nil
		default:
			time.Sleep(25 * time.Second)
			om.handleTicker()
		}
	}
}

func (om *OrderMonitor) handleTicker() {
	if len(om.orders) == 0 {
		return
	}
	om.lock.RLock()
	defer om.lock.RUnlock()
	baseParams := om.orders[0].BaseOrderParams
	price, err := om.accMgr.GetPrice(baseParams)
	if err != nil {
		logger.Errorf("get(%v) price error:%v", baseParams, err)
		return
	}
	// logger.Infof("(%s,%s) price: %v", baseParams.Market, baseParams.Symbol, price)
	if price < MinPriceValue {
		logger.Errorf(" %v price 异常", baseParams)
		return
	}

	tmpOrds := make([]*accmgr.Order, 0)
	//1. 如果是LIMIT，判断是否已经成交(如果是做多，判断价格是否低于订单价格而成交，如果做空，判断价格是否高于订单价格而成交)
	//2. 如果是MARKET，判断是否满足止盈止损价格
	//3. 如果满足，则回调函数，并更新订单状态
	for _, order := range om.orders {
		if order.Status == common.OrderStatusTypeFilled {
			if order.BaseOrderParams.Market == common.MarketSpot && order.Side == common.TradeSideShort {
				//现货卖出单 不需要止盈止损
				continue
			}
			//已经吃单，
			_, isWinOrLoss := om.winloss(price, order)
			if isWinOrLoss {
				//止盈止损成功，更新订单状态
				order.Status = common.OrderStatusTypeFilled
				logger.Infof("止盈止损成功，更新订单状态(%s,%s) price: %v,%v", baseParams.Market, baseParams.Symbol, price, order.Ext)
				continue
			}
			tmpOrds = append(tmpOrds, order)
			continue
		}
		// ordPrice := utils.ToFloat64(order.Price)
		//1.如果是限价单，先判断是否成交
		if order.Type == common.OrderTypeLimit {
			// if (order.Side == common.TradeSideLong && price < ordPrice) ||
			// 	order.Side == common.TradeSideShort && price > ordPrice {
			//做多，判断价格是否低于订单价格而成交
			//做空，判断价格是否高于订单价格而成交
			// order.Status = common.OrderStatusTypeFilled
			// logger.Infof("限价单Buy/Sell成功，更新订单状态(%s,%s) price: %v,%v", baseParams.Market, baseParams.Symbol, price, order.Ext)
			//todo: 应该要去拉订单验证一下
			ords, err := om.accMgr.GetOrders(accmgr.GetOrderParams{
				BaseOrderParams: order.BaseOrderParams,
				OrderID:         &order.ID,
			})
			if err != nil || len(ords) == 0 {
				logger.Errorf("monitor get order(%d) error:%v", order.ID, err)
				continue
			}
			if ords[0].Status == common.OrderStatusTypeCanceled || ords[0].Status == common.OrderStatusTypeExpired {
				continue
			}
			ext := order.Ext
			order = ords[0]
			order.Ext = ext

			// } else {
			if order.Status == common.OrderStatusTypeNew || order.Status == common.OrderStatusTypePartiallyFilled {
				isCancel, _ := om.cancelOrderIfTimeout(order)
				if isCancel {
					continue
				}
			}

		}
		tmpOrds = append(tmpOrds, order)
	}
	om.orders = tmpOrds
	if len(om.orders) == 0 {
		err := om.Stop()
		if err != nil {
			logger.Errorf("stop order monitor error:%v", err)
		}
	}
}

func (om *OrderMonitor) winloss(price float64, order *accmgr.Order) (*accmgr.Order, bool) {
	if order.Ext == nil || order.Ext.WinPrice == nil || order.Ext.LossPrice == nil {
		return nil, false
	}
	var placeParams *accmgr.CloseOrderParams
	lossPrice := MinPriceValue
	winPrice := MinPriceValue
	if order.Ext.WinPrice != nil {

		winPrice = utils.ToFloat64(*order.Ext.WinPrice)
	}
	if order.Ext.LossPrice != nil {
		lossPrice = utils.ToFloat64(*order.Ext.LossPrice)
	}

	if order.Side == common.TradeSideLong {
		//做多，判断价格是否低于止盈价格而成交
		if (winPrice > MinPriceValue && price >= winPrice) ||
			(lossPrice > MinPriceValue && price <= lossPrice) {
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
		if (lossPrice > MinPriceValue && price >= lossPrice) ||
			(winPrice > MinPriceValue && price <= winPrice) {
			placeParams = &accmgr.CloseOrderParams{
				BaseOrderParams: order.BaseOrderParams,
				PositionSide:    common.TradeSideShort,
				Qty:             &order.Qty,
			}
		}
	}

	if placeParams != nil {
		err := om.accMgr.CloseOrders(*placeParams)
		if err != nil {
			logger.Errorf("(%s,%s)CloseOrders order(amount: %s, price: %s) error: %v", placeParams.Market, placeParams.Symbol, placeParams.Qty, price, err)
			return nil, false
		}
		logger.Infof("(%s,%s)CloseOrders order(amount: %s, price: %s, ID: %s) 成功", placeParams.Market, placeParams.Symbol, placeParams.Qty, price, order.ID)
		return nil, true
	}
	return nil, false
}

func (om *OrderMonitor) Stop() error {
	if om.running {
		om.done <- true
		return nil
	}
	return fmt.Errorf("order monitor is not running")
}

func (om *OrderMonitor) cancelOrderIfTimeout(order *accmgr.Order) (bool, error) {
	if order.Status == common.OrderStatusTypeFilled {
		return false, nil
	}
	if order.Ext == nil || order.Ext.Timeout == nil {
		return false, nil
	}
	timeout := *(order.Ext.Timeout)
	if timeout < time.Now().Unix() {
		err := om.accMgr.CancelOrders(accmgr.CancelOrderParams{
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
