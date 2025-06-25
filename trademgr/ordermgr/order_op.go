package ordermgr

import (
	"fmt"
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"time"
)

func (o *OrderMgr) clearOrders() {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.orders = make([]*accmgr.Order, 0)
}
func (o *OrderMgr) appenOrders(orders []*accmgr.Order) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.orders = append(o.orders, orders...)
	if !o.running {
		go o.Work()
	}
}

// 是否吃单/超时/止盈止损
func (o *OrderMgr) updateOrder(ord *accmgr.Order) error {
	if ord == nil {
		return fmt.Errorf("%v order is nil", o.GetInfo())
	}
	ords, err := o.accMgr.GetOrders(accmgr.GetOrderParams{
		BaseOrderParams: ord.BaseOrderParams,
		OrderID:         &ord.ID,
	})
	if err != nil || len(ords) == 0 {
		logger.Errorf("%v get order(%d) error:%v", o.GetInfo(), ord.ID, err)
		return nil
	}
	ordStatus := ords[0].Status
	if ordStatus == common.OrderStatusTypeFilled && (ord.Status == common.OrderStatusTypeNew || ord.Status == common.OrderStatusTypePartiallyFilled) {
		//从吃单开始(相近时间)算
		o.strateMsg.StrategyRet.TradeSuggest.CreateTime = time.Now().Unix()
	}
	ord.Status = ordStatus
	if ord.Status == common.OrderStatusTypeNew || ord.Status == common.OrderStatusTypePartiallyFilled {
		isCancel, _ := o.cancelOrderIfTimeout(ord)
		if isCancel {
			ord.Status = common.OrderStatusTypeCanceled
		}
	}

	return nil
}

func (o *OrderMgr) cancelOrderIfTimeout(order *accmgr.Order) (bool, error) {
	if order.Status == common.OrderStatusTypeFilled {
		return false, nil
	}
	if order.Ext == nil {
		return false, nil
	}
	timeout := order.Ext.Timeout
	if timeout > 0 && timeout < time.Now().Unix() {
		err := o.accMgr.CancelOrders(accmgr.CancelOrderParams{
			BaseOrderParams: order.BaseOrderParams,
			OrderID:         &order.ID,
		})
		if err != nil {
			logger.Errorf("%s cancel order(ID: %d) error: %v", o.GetInfo(), order.ID, err)
			return false, err
		}
		logger.Warnf("%s cancel order(ID: %d) timeout", o.GetInfo(), order.ID)
		return true, nil
	}
	return false, nil
}

func (o *OrderMgr) handleTickerLoop(price float64) {
	if len(o.orders) == 0 {
		return
	}
	o.lock.RLock()
	defer o.lock.RUnlock()

	// logger.Infof("(%s,%s) price: %v", baseParams.Market, baseParams.Symbol, price)
	if price < common.MinFloatValue {
		logger.Errorf(" %v price 异常", o.GetInfo())
		return
	}

	tmpOrds := make([]*accmgr.Order, 0)
	for _, order := range o.orders {

		err := o.updateOrder(order)
		if err != nil {
			logger.Errorf("update order(%d) error:%v", order.ID, err)
			continue
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
			_, isWinOrLoss := o.winloss(price, order)
			if isWinOrLoss {
				//止盈止损成功，更新订单状态
				order.Status = common.OrderStatusTypeExpired
				logger.Infof("止盈止损成功，更新订单状态(%s) price: %v,%+v", o.GetInfo(), price, *(order.Ext))
				continue
			}
		}
		tmpOrds = append(tmpOrds, order)
	}
	o.orders = tmpOrds
	// if len(o.orders) == 0 {
	// 	err := o.Stop()
	// 	if err != nil {
	// 		logger.Errorf("stop order monitor error:%v", err)
	// 	}
	// 	o.silent(SilentType_CloseOrder, 30.0)
	// }
}

func (o *OrderMgr) winloss(price float64, order *accmgr.Order) (*accmgr.Order, bool) {
	if order.Ext == nil {
		return nil, false
	}
	var placeParams *accmgr.CloseOrderParams
	winPrice := utils.ToFloat64(order.Ext.WinPrice)
	lossPrice := utils.ToFloat64(order.Ext.LossPrice)

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
				TradeSide:       common.TradeSideShort,
				Qty:             &(order.Qty),
			}
		}
	} else if order.Side == common.TradeSideShort {
		if order.BaseOrderParams.Market == common.MarketSpot {
			logger.Warnf("%v 现货卖出单 不需要止损", o.GetInfo())
			return nil, false
		}
		// side := common.TradeSideCloseSell
		//做空，判断价格是否高于止损价格而成交
		if (lossPrice > common.MinFloatValue && price >= lossPrice) ||
			(winPrice > common.MinFloatValue && price <= winPrice) {
			placeParams = &accmgr.CloseOrderParams{
				BaseOrderParams: order.BaseOrderParams,
				TradeSide:       common.TradeSideLong,
				Qty:             &(order.Qty),
			}
		}
	}

	if placeParams != nil {
		err := o.accMgr.CloseOrders(*placeParams)
		if err != nil {
			logger.Errorf("(%s,%s)CloseOrders order(amount: %v, price: %s) error: %v", placeParams.Market, placeParams.Symbol, *(placeParams.Qty), price, err)
			return nil, false
		}
		logger.Infof("(%s,%s)CloseOrders order(amount: %v, price: %s, ID: %s) 成功", placeParams.Market, placeParams.Symbol, *(placeParams.Qty), price, order.ID)
		return nil, true
	}
	return nil, false
}

func (o *OrderMgr) silentType(sType SilentType, silenceTimeX float64) {
	if silenceTimeX <= 0.0001 {
		return
	}
	o.isSilent = true

	timeAfter := time.After(time.Second * time.Duration(silenceTimeX*60.0))
	endSec := time.Now().Unix() + int64(silenceTimeX*60.0)
	logger.Infof("\n %v 开始(%s)静默:%s,预计结束时间:%s\n", o.GetInfo(), sType.String(), time.Now().Format("2006-01-02 15:04:05"), utils.TimeFmt(int64(endSec), ""))
	curTime, _ := <-timeAfter
	logger.Infof("\n%v 结束(%s)静默:%s-%s\n", o.GetInfo(), sType.String(), curTime, time.Now().Format("2006-01-02 15:04:05"))
	o.isSilent = false
}

func (o *OrderMgr) getKLines(period int64) (*common.KLine, error) {
	// 调用数据接口获取数据
	params := o.params
	quicks := params.ClosePositionParams.QuickVolidities
	if quicks != nil && len(*quicks) > 0 {
		qs := *quicks
		q := qs[0]
		if q.NodeKPeriod > 0 {
			period = q.NodeKPeriod
		}

	}
	dataParams := accmgr.KLineParams{
		Base: accmgr.Base{
			Market:   params.BaseParams.Market,
			Symbol:   params.BaseParams.Symbol,
			Exchange: params.BaseParams.Exchange,
			Limit:    1,
			Period:   int(period),
		},
	}
	klines, err := o.accMgr.GetKLines(dataParams)
	if err != nil {
		return nil, err
	}
	k := klines[0]
	return &k, nil
}
