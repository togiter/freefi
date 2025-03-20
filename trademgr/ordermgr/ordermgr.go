package ordermgr

import (
	"fmt"
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"math"
	"strings"
	"time"
)

//1. 接收策略建议
//2. 处理订单(下单/撤单/平仓)
//3. 分派监控订单状态和止盈止损

type IOrderMgr interface {
	IsDone() bool
	Stop() error
	Work() error
	Update(strateMsg StrategyMsg) error
}

type OrderMgr struct {
	orders    []*accmgr.Order
	strateMsg StrategyMsg
	//沉默状态(该状态不接受策略建议)
	isSilent     bool
	params       TradeParams
	orderMonitor IOrderMonitor
}

// Work implements IOrderMgr.
func (o *OrderMgr) Work() error {
	return o.orderMonitor.Start()
}

// Exit implements IOrderMgr.
func (o *OrderMgr) Stop() error {
	o.orderMonitor.Stop()
	return nil
}

// IsDone implements IOrderMgr.
func (o *OrderMgr) IsDone() bool {
	return o.orderMonitor.OrderEmpty()
}

// 是否满足止损止盈策略
func (o *OrderMgr) canCloseStrateMsg(strateMsg StrategyMsg) (canClose bool, tradeSide string) {

	//1.输入总策略为空(分支策略判断是否平仓)
	strategyRet := strateMsg.StrategyRet
	if strategyRet.TradeSuggest.TradeSide != common.TradeSideNone {
		return
	}
	//2.当前策略不为空 && 当前有订单
	if len(strategyRet.GroupStrategyRets) == 0 {
		return
	}
	//0.提取止盈止损策略,如果没有配就返回false
	cfg := o.params.ClosePositionParams
	if cfg == nil {
		return
	}
	curSide := o.strateMsg.StrategyRet.TradeSuggest.TradeSide
	//3.遍历分组策略(当前只适用单个分组策略)
	gs := strategyRet.GroupStrategyRets[cfg.GroupKPeroid]
	if gs == nil {
		return
	}
	//4.遍历分组策略的每个策略
	for _, micro := range *cfg.Strategies {
		microSugguest := gs.MicroStrategyRets[micro]
		tSide := microSugguest.TradeSuggest.TradeSide
		if tSide != common.TradeSideNone && tSide != curSide {
			logger.Info("%s满足平仓策略(%s)", strateMsg.DataSource.Symbol, tSide)
			return true, tSide
		}
	}
	effectCount := 0
	for _, micro := range *cfg.Strategies {
		microSugguest := gs.MicroStrategyRets[micro]
		tSide := microSugguest.TradeSuggest.TradeSide
		if tSide != common.TradeSideNone && tSide != curSide {
			effectCount += 1
			tradeSide = tSide
			continue
		}
	}
	//满足全部微策略指标
	if effectCount > 0 && effectCount == len(*cfg.Strategies) {
		logger.Info("%s满足全部微策略的平仓策略(%d)....", strateMsg.DataSource.Symbol, effectCount)
		canClose = true
		return
	}

	// for _, gs := range *strateMsg.GroupSuggests {
	// 	//5.遍历分组策略的每个策略

	// }

	return
}

// 平仓流程: 取消挂单=>关闭持仓
func (o *OrderMgr) closeOrders(tradeSide string) (done bool, err error) {
	//1. 是否有挂单，清除挂单
	accMgr := accmgr.NewAccMgr()
	baseParams := accmgr.BaseOrderParams{
		Symbol:   o.params.BaseParams.Symbol,
		Exchange: o.params.BaseParams.Exchange,
		Market:   o.params.BaseParams.Market,
	}
	status := common.OrderStatusTypeNew
	ords, err := accMgr.GetOrders(accmgr.GetOrderParams{
		BaseOrderParams: baseParams,
		Status:          &status,
	})
	if err == nil && len(ords) > 0 {
		err = accMgr.CancelOrders(accmgr.CancelOrderParams{
			BaseOrderParams: baseParams,
			OrderID:         nil,
		})
		if err != nil {
			logger.Errorf("cancel orders error: %v", err)
		} else {
			logger.Infof("cancel all %s orders success", baseParams.Symbol)
		}
	}
	//反向侧平仓
	rSide := common.ReserveSide(tradeSide)
	//2. 是否有持仓，平仓
	positions, err := accMgr.GetPositions(accmgr.PositionParams{
		BaseOrderParams: baseParams,
		Side:            &rSide,
	})
	if err != nil || len(positions) == 0 {
		logger.Errorf("get positions error: %v", err)
	} else {
		position := positions[0]
		if utils.ToFloat64(position.Qty) > MinPriceValue {
			err = accMgr.CloseOrders(accmgr.CloseOrderParams{
				BaseOrderParams: baseParams,
				PositionSide:    rSide,
				StopPrice:       nil,
				Qty:             &position.Qty,
			})
			if err != nil {
				logger.Errorf("close orders error: %v", err)
				err = fmt.Errorf("close orders error: %v", err)
				return
			}
			done = true
			logger.Infof("close all %s position(qty:%s) success", baseParams.Symbol, position.Qty)
		}
	}
	return
}

// Update implements IOrderMgr.
func (o *OrderMgr) Update(strate StrategyMsg) error {
	//静默状态不接受策略建议
	if o.isSilent { //todo： 静默时间
		logger.Info("order manager is in silent mode, ignore strategy msg")
		return nil
	}
	strategyRet := strate.StrategyRet
	if strategyRet.TradeSuggest.TradeSide == o.strateMsg.StrategyRet.TradeSuggest.TradeSide {
		logger.Info("The same trade side, ignore")
		return nil
	}
	if strategyRet.TradeSuggest.TradeSide == common.TradeSideNone {

		//可能使用局部(部分)策略止损止盈
		canClose, tradeSide := o.canCloseStrateMsg(strate)
		if canClose {
			logger.Infof("%s 可以平仓", strate.DataSource.Symbol)
			done, err := o.closeOrders(tradeSide)
			if err != nil {
				logger.Errorf("close orders error: %v", err)
				return err
			}
			o.strateMsg = StrategyMsg{}
			if done {
				o.silent(SilentType_CloseOrder)
			}
			return nil
		}

		logger.Info("order manager received trade suggest with trade side none, ignore")
		return nil

	}

	err := o.handlerOrders(strate)
	if err != nil {
		logger.Errorf("handler orders error: %v", err)
		return err
	}

	//检测订单
	if o.orderMonitor.OrderEmpty() {
		logger.Info("order manager has no orders, ignore strategy msg")
		o.strateMsg = StrategyMsg{}
		o.orderMonitor.Stop()
		return nil
	}
	o.strateMsg = strate
	o.silent(SilentType_NewOrder)
	return nil
}

func (o *OrderMgr) getAvaliableBalance(isSell bool, sMsg StrategyMsg) (string, error) {
	pair := strings.ToUpper(sMsg.DataSource.Symbol)
	coins := strings.Split(pair, "-")
	if len(coins) == 0 {
		return "0", fmt.Errorf("invalid symbol: %s", o.params.BaseParams.Symbol)
	}
	market := o.params.BaseParams.Market
	sym := coins[len(coins)-1] //默认查询usdt
	if market == common.MarketSpot {
		//现货 && 空单才查询现货余额
		if isSell {
			sym = coins[0] //sell,查询现货余额
		}
	}
	accs := accmgr.NewAccMgr()
	// if market != common.MarketSpot && len(o.orders) > 0 {
	// 	//非现货先取消挂单
	// 	err := accs.CancelOrders(accmgr.CancelOrderParams{
	// 		BaseOrderParams: accmgr.BaseOrderParams{
	// 			Symbol:   o.params.BaseParams.Symbol,
	// 			Exchange: o.params.BaseParams.Exchange,
	// 			Market:   o.params.BaseParams.Market,
	// 		},
	// 	})
	// 	if err != nil {
	// 		logger.Errorf("cancel all orders error: %v", err)
	// 	}
	// }
	bals, err := accs.GetBalances(accmgr.GetBalanceParams{
		BaseOrderParams: accmgr.BaseOrderParams{
			Symbol:   sym,
			Exchange: o.params.BaseParams.Exchange,
			Market:   o.params.BaseParams.Market,
		},
	})
	if err != nil || len(bals) == 0 {
		return "0", fmt.Errorf("get balances error or balance is 0: %v", err)
	}
	logger.Infof("%s getAvaliableBalanceed strategy msg: %+v", sMsg.DataSource.Symbol, bals[0])
	return bals[0].Available, nil
}

func (o *OrderMgr) handleOrderBefore(sMsg StrategyMsg) error {

	_, err := o.closeOrders(sMsg.StrategyRet.TradeSuggest.TradeSide)
	logger.Infof("%scloseOrders... %v", sMsg.DataSource.Symbol, err)
	o.orderMonitor.Stop()
	return nil
}

func (o *OrderMgr) handlerOrders(strateMsg StrategyMsg) error {
	if err := o.handleOrderBefore(strateMsg); err != nil {
		return fmt.Errorf("handle order before error:%v", err)
	}
	strateRet := strateMsg.StrategyRet
	bal, err := o.getAvaliableBalance(strateRet.TradeSuggest.TradeSide == common.TradeSideShort, strateMsg)
	if err != nil || bal == "0" || bal == "" {
		logger.Errorf("get avaliable balance error/0: %v", err)
		return fmt.Errorf("get avaliable balance error/0: %v", err)
	}
	txDir := 1 //买入方向
	if strateRet.TradeSuggest.TradeSide == common.TradeSideShort {
		txDir = -1 //卖出方向
	}
	baseParams := accmgr.BaseOrderParams{
		Symbol:   o.params.BaseParams.Symbol,
		Exchange: o.params.BaseParams.Exchange,
		Market:   o.params.BaseParams.Market,
	}

	logger.Infof("%s baseparams: %v", baseParams.Symbol, baseParams)

	//开单价格
	oPrice := strateRet.TradeSuggest.Price
	//Market price参数无效
	if o.params.TradeType == common.OrderTypeLimit {
		curPrice, err := getPrice(baseParams)
		if err != nil {
			logger.Errorf("get price error: %v", err)
			return err
		}
		if curPrice > 0.00000001 { //precision handler
			oPrice = curPrice
		}
	}
	if oPrice < 0.00000001 {
		logger.Errorf("invalid price: %v", oPrice)
		return fmt.Errorf("invalid price: %v", oPrice)
	}
	//最小USDT数量可购买
	minUsdtQty := o.params.MinUsdtQty
	//杠杆倍数
	leverRate := o.params.LeverRate
	//资金利用率
	effectiveMoney := math.Min(1.0, math.Max(float64(o.params.PositionUseRate), 0.1))
	logger.Info("avaliable effectiveMoney: ", effectiveMoney)
	extBalance := utils.ToFloat64(bal)
	if strateMsg.DataSource.Market != common.MarketSpot {
		extBalance = extBalance * leverRate * effectiveMoney
	} else {
		extBalance = extBalance * effectiveMoney
	}
	logger.Info("avaliable balance: ", extBalance)
	canTrade := extBalance //spot sell
	if strateRet.TradeSuggest.TradeSide == common.TradeSideLong {
		if extBalance < math.Max(minUsdtQty, 20.0) {
			return fmt.Errorf("not enough minUsdtQty(%.5f): %.5f", minUsdtQty, extBalance)
		}
	} else {
		if extBalance < o.params.MinTokenQty {
			return fmt.Errorf("not enough minTokenQty(%.5f): %.5f", o.params.MinTokenQty, extBalance)
		}
	}

	if baseParams.Market != common.MarketSpot || strateRet.TradeSuggest.TradeSide == common.TradeSideLong {
		canTrade = extBalance / oPrice //buy
	}
	logger.Infof("canTrade balance: %.5f,market:%s,tside:%s", canTrade, baseParams.Market, strateRet.TradeSuggest.TradeSide)
	if canTrade < 0.00001 {
		return fmt.Errorf("not enough balance: %.5f", canTrade)
	}

	//分批挂单数量
	maxC := o.params.OrdersCount
	//分批价格增量
	priceIncr := o.params.PriceIncr
	//分批数量增量
	amountIncr := o.params.QtyIncr

	//交易类型 限价/市价
	tradeT := strings.ToUpper(o.params.TradeType)
	//开单数量
	amounts := calAmountsByTotal(utils.ToFloat64(canTrade), amountIncr, maxC, o.params.StrategyType)

	oPrice = oPrice * (1.0 - float64(txDir)*o.params.InitPricePer)
	acc := accmgr.NewAccMgr()
	qtyPrecision := int(math.Min(4, float64(o.params.QtyPrecision)))
	pricePrecision := int(math.Min(5, float64(o.params.PricePrecision)))

	winRate := 0.0
	lossRate := 0.0
	if o.params.ClosePositionParams != nil {
		winRate = o.params.ClosePositionParams.WinRate
		lossRate = o.params.ClosePositionParams.LossRate
	}
	ords := make([]*accmgr.Order, 0)
	for i := 0; i < len(amounts); i++ {
		amount := amounts[i]
		qty := utils.ToFixed(amount, qtyPrecision)
		priC := utils.ToFixed(oPrice, pricePrecision)
		order, err := acc.CreateOrder(accmgr.PlaceOrderParams{
			BaseOrderParams: baseParams,
			Side:            strateRet.TradeSuggest.TradeSide,
			Type:            tradeT,
			Price:           priC,
			Qty:             qty,
			LeverRate:       leverRate,
		})
		if err != nil {
			logger.Errorf("create order(amount: %s, price: %s) error: %v", qty, priC, err)
			continue
		}
		// o.orders = append(o.orders, order)
		if baseParams.Market == common.MarketSpot && order.Type == common.OrderTypeMarket && order.Side == common.TradeSideShort {
			//现货&市价&&卖出单
			logger.Infof("现货卖出单(%+v) success", order)
			continue
		}

		order.Ext = &accmgr.OrderExt{
			IsWinOrLoss: true,
		}
		if winRate > common.MinFloatValue {
			winPrice := utils.ToFixed(oPrice*(1.0+float64(txDir)*winRate), pricePrecision)
			order.Ext.WinPrice = &winPrice
		}
		if lossRate > common.MinFloatValue {
			lossPrice := utils.ToFixed(oPrice*(1.0-float64(txDir)*lossRate), pricePrecision)
			order.Ext.LossPrice = &lossPrice
		}
		if o.params.TimeParams.TimeoutCancelPeriodX > common.MinFloatValue {

			timeout := time.Now().Add(time.Duration(o.params.TimeParams.TimeoutCancelPeriodX*o.params.TimeParams.KPeriod*60) * time.Second).Unix()
			order.Ext.Timeout = &timeout
		}
		if order.BaseOrderParams.Symbol == "" {
			order.BaseOrderParams = baseParams
		}
		logger.Infof("create order(%+v),Ext(w: %s, l: %s,timeout:%s) success", order, *order.Ext.WinPrice, *order.Ext.LossPrice, utils.TimeFmt(*order.Ext.Timeout, ""))
		ords = append(ords, order)
		oPrice = oPrice * (1.0 - float64(txDir)*priceIncr)
	}
	if len(ords) > 0 {
		o.orderMonitor.ResetToOrders(ords)
	}
	return nil
}

func (o *OrderMgr) silent(sType SilentType) {

	silenceTimeX := 0.0
	if sType == SilentType_NewOrder {
		silenceTimeX = o.params.TimeParams.OpenOrderNoOpWaitPeriodX
	} else if sType == SilentType_TimeoutCancelOrder {
		silenceTimeX = o.params.TimeParams.TimeoutCancelPeriodX
	} else if sType == SilentType_CloseOrder {
		silenceTimeX = o.params.ClosedOrderNoOpWaitPeriodX
	}
	silenceTimeX = silenceTimeX * o.params.TimeParams.KPeriod
	if silenceTimeX <= 0.0001 {
		return
	}
	o.isSilent = true

	timeAfter := time.After(time.Second * time.Duration(silenceTimeX*60.0))
	endSec := time.Now().Unix() + int64(silenceTimeX*60.0)
	logger.Infof("\n开始(%s)静默:%s,预计结束时间:%s\n", sType.String(), time.Now().Format("2006-01-02 15:04:05"), utils.TimeFmt(int64(endSec), ""))
	curTime, _ := <-timeAfter
	logger.Infof("\n结束(%s)静默:%s-%s\n", sType.String(), curTime, time.Now().Format("2006-01-02 15:04:05"))
	o.isSilent = false
}

func (o *OrderMgr) unsilent() {
	o.isSilent = false
}

func NewOrderMgr(tradeParams TradeParams) IOrderMgr {
	return &OrderMgr{
		orders: make([]*accmgr.Order, 0),
		// strateMsg: strateMsg,
		params:       tradeParams,
		orderMonitor: NewOrderMonitor(),
	}
}
