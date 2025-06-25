package ordermgr

import (
	"fmt"
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"math"
	"strings"
	"sync"
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
	// Monitor
	strateMsg StrategyMsg
	params    TradeParams
	done      chan bool
	isSilent  bool
	running   bool
	orders    []*accmgr.Order
	lock      sync.RWMutex
	accMgr    accmgr.IAccMgr
}

// Work implements IOrderMgr.
func (o *OrderMgr) Work() error {
	val := o.params.TimeParams.OrderStatusCheckTicker
	if o.running {
		return fmt.Errorf("%v order monitor is already running", o.GetInfo())
	}
	o.running = true
	logger.Infof("%v orders(%d) monitor starting ", o.GetInfo(), len(o.orders))
	loopT := 25 * time.Second
	if val > 0 {
		loopT = time.Duration(val) * time.Second
	}
	for {
		select {
		case <-o.done:
			o.running = false
			logger.Infof("%v order monitor stopped", o.GetInfo())
			return nil
		default:
			time.Sleep(loopT)
			o.handleTicker()
		}
	}
}

func (o *OrderMgr) handleTicker() {
	kline, err := o.getKLines(5)

	// baseParams := o.orders[0].BaseOrderParams
	// price, err := o.accMgr.GetPrice(baseParams)
	if err != nil {
		logger.Errorf("get(%v) price error:%v", o.GetInfo(), err)
		return
	}
	isClose, _ := o.closeVolidity(*kline)
	if isClose {
		return
	}
	o.handleTickerLoop(kline.Close)
}

// Exit implements IOrderMgr.
func (o *OrderMgr) Stop() error {
	if o.running {
		o.done <- true
		o.running = false
		return nil
	}
	return fmt.Errorf("%v order monitor is not running", o.GetInfo())
}

// IsDone implements IOrderMgr.
func (o *OrderMgr) IsDone() bool {
	return len(o.orders) == 0
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
			logger.Errorf("%v cancel orders error: %v", o.GetInfo(), err)
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
		logger.Errorf("%v get positions error: %v", baseParams.Symbol, err)
	} else {
		position := *(positions[0])
		qtyStr := position.Qty
		if utils.ToFloat64(qtyStr) >= o.params.MinTokenQty {
			qty := utils.ToFixedFloor(qtyStr, o.params.QtyPrecision)

			err = accMgr.CloseOrders(accmgr.CloseOrderParams{
				BaseOrderParams: baseParams,
				TradeSide:       tradeSide,
				StopPrice:       nil,
				Qty:             &qty,
			})
			if err != nil {
				logger.Errorf("%v close orders error: %v", o.GetInfo(), err)
				err = fmt.Errorf("%v close orders error: %v", o.GetInfo(), err)
				return
			}
			done = true
			logger.Infof("close all %s position(qty:%s) success", baseParams.Symbol, position.Qty)
		}
	}
	return
}

func (o *OrderMgr) GetInfo() string {
	bs := o.params.BaseParams
	return fmt.Sprintf("%s-%s", bs.Symbol, bs.Market)
}

// Update implements IOrderMgr.
func (o *OrderMgr) Update(strate StrategyMsg) error {
	//静默状态不接受策略建议
	if o.isSilent {
		logger.Infof("%v order manager is in silent mode, ignore strategy msg", o.GetInfo())
		return nil
	}
	curSide := o.curSide() //o.strateMsg.StrategyRet.TradeSuggest.TradeSide
	//出交易决策不换，其他都换
	tmpStrate := strate
	o.strateMsg = tmpStrate
	o.strateMsg.StrategyRet.TradeSuggest.TradeSide = curSide

	close, tradeSide := o.closeJudge(strate, curSide)
	if close {
		logger.Infof("%s 可以平仓", strate.DataSource.Symbol)
		done, err := o.closeOrders(tradeSide)
		if err != nil {
			logger.Errorf("%v close orders error: %v", o.GetInfo(), err)
			return err
		}
		o.strateMsg = StrategyMsg{}
		if done {
			o.clearOrders()
			o.silent(SilentType_CloseOrder)
		}
		return nil
	}
	newSide := strate.StrategyRet.TradeSuggest.TradeSide
	if newSide == common.TradeSideNone || curSide == newSide {
		return nil
	}

	err := o.handlerOrders(strate)
	if err != nil {
		logger.Errorf("%v handler orders error: %v", o.GetInfo(), err)
		return err
	}

	//检测订单
	if o.IsDone() {
		logger.Infof("%v order manager has no orders, ignore strategy msg", o.GetInfo())
		o.strateMsg = StrategyMsg{}
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
	bals, err := o.accMgr.GetBalances(accmgr.GetBalanceParams{
		BaseOrderParams: accmgr.BaseOrderParams{
			Symbol:   sym,
			Exchange: o.params.BaseParams.Exchange,
			Market:   o.params.BaseParams.Market,
		},
	})
	if err != nil || len(bals) == 0 {
		return "0", fmt.Errorf("%v get balances error or balance is 0: %v", o.GetInfo(), err)
	}
	logger.Infof("%s getAvaliableBalanceed strategy msg: %+v", sMsg.DataSource.Symbol, bals[0])
	bal := utils.ToFixedFloor(bals[0].Available, o.params.QtyPrecision)
	return bal, nil
}

func (o *OrderMgr) handleOrderBefore(sMsg StrategyMsg) error {
	// _, err := o.closeOrders(sMsg.StrategyRet.TradeSuggest.TradeSide)
	// logger.Infof("%scloseOrders... %v", sMsg.DataSource.Symbol, err)
	return nil
}

func (o *OrderMgr) handlerOrders(strateMsg StrategyMsg) error {
	if err := o.handleOrderBefore(strateMsg); err != nil {
		return fmt.Errorf("%v handle order before error:%v", o.GetInfo(), err)
	}
	strateRet := strateMsg.StrategyRet
	bal, err := o.getAvaliableBalance(strateRet.TradeSuggest.TradeSide == common.TradeSideShort, strateMsg)
	if err != nil || bal == "0" || bal == "" {
		logger.Errorf("%v get avaliable balance error/0: %v", o.GetInfo(), err)
		return fmt.Errorf(" %v get avaliable balance error/0: %v", o.GetInfo(), err)
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

	logger.Infof("%s base params: %v", baseParams.Symbol, baseParams)

	//开单价格
	oPrice := strateRet.TradeSuggest.Price
	//Market price参数无效
	if o.params.TradeType == common.OrderTypeLimit {
		curPrice, err := o.accMgr.GetPrice(baseParams)
		if err != nil {
			logger.Errorf("%v get price error: %v", o.GetInfo(), err)
			return err
		}
		if curPrice > common.MinFloatValue { //precision handler
			oPrice = curPrice
		}
	}
	if oPrice < common.MinFloatValue {
		logger.Errorf("%v invalid price: %v", o.GetInfo(), oPrice)
		return fmt.Errorf(" %v invalid price: %v", o.GetInfo(), oPrice)
	}
	//最小USDT数量可购买
	minUsdtQty := o.params.MinUsdtQty
	//杠杆倍数
	leverRate := o.params.LeverRate
	//资金利用率
	effectiveMoney := math.Min(1.0, math.Max(float64(o.params.PositionUseRate), 0.1))
	extBalance := utils.ToFloat64(bal)
	if strateMsg.DataSource.Market != common.MarketSpot {
		extBalance = extBalance * leverRate * effectiveMoney
	} else {
		extBalance = extBalance * effectiveMoney
	}
	canTrade := extBalance //spot sell
	if strateRet.TradeSuggest.TradeSide == common.TradeSideLong {
		if extBalance < math.Max(minUsdtQty, 20.0) {
			return fmt.Errorf("%v not enough minUsdtQty(%.5f): %.5f", o.GetInfo(), minUsdtQty, extBalance)
		}
	} else {
		if extBalance < o.params.MinTokenQty {
			return fmt.Errorf("%v not enough minTokenQty(%.5f): %.5f", o.GetInfo(), o.params.MinTokenQty, extBalance)
		}
	}

	if baseParams.Market != common.MarketSpot || strateRet.TradeSuggest.TradeSide == common.TradeSideLong {
		canTrade = extBalance / oPrice //buy
	}
	logger.Infof("%v canTrade balance: %.5f,market:%s,tside:%s", o.GetInfo(), canTrade, baseParams.Market, strateRet.TradeSuggest.TradeSide)
	if canTrade < 0.001 {
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
	qtyPrecision := int(math.Min(4, float64(o.params.QtyPrecision)))
	pricePrecision := int(math.Min(5, float64(o.params.PricePrecision)))
	ords := make([]*accmgr.Order, 0)
	for i := 0; i < len(amounts); i++ {
		amount := amounts[i]
		qty := utils.ToFixedFloor(amount, qtyPrecision)
		priC := utils.ToFixedFloor(oPrice, pricePrecision)
		order, err := o.accMgr.CreateOrder(accmgr.PlaceOrderParams{
			BaseOrderParams: baseParams,
			Side:            strateRet.TradeSuggest.TradeSide,
			Type:            tradeT,
			Price:           priC,
			Qty:             qty,
			LeverRate:       leverRate,
		})
		if err != nil {
			logger.Errorf("%v create order(amount: %s, price: %s) error: %v", o.GetInfo(), qty, priC, err)
			continue
		}
		// o.orders = append(o.orders, order)
		if baseParams.Market == common.MarketSpot && order.Type == common.OrderTypeMarket && order.Side == common.TradeSideShort {
			//现货&市价&&卖出单
			logger.Infof("%v 现货卖出单(%+v) success", o.GetInfo(), order)
			continue
		}

		order.Ext = &accmgr.OrderExt{}
		winRate := 0.0
		lossRate := 0.0
		if o.params.ClosePositionParams != nil {
			winRate = o.params.ClosePositionParams.WinRate
			lossRate = o.params.ClosePositionParams.LossRate
		}
		if winRate > common.MinFloatValue {
			winPrice := utils.ToFixed(oPrice*(1.0+float64(txDir)*winRate), pricePrecision)
			order.Ext.WinPrice = winPrice
		}
		if lossRate > common.MinFloatValue {
			lossPrice := utils.ToFixed(oPrice*(1.0-float64(txDir)*lossRate), pricePrecision)
			order.Ext.LossPrice = lossPrice
		}
		tParams := o.params.TimeParams
		if tParams.TimeoutCancelPeriodX > common.MinFloatValue {

			timeout := time.Now().Add(time.Duration(tParams.TimeoutCancelPeriodX*tParams.KPeriod*60) * time.Second).Unix()
			order.Ext.Timeout = timeout
		}
		if order.BaseOrderParams.Symbol == "" {
			order.BaseOrderParams = baseParams
		}
		logger.Infof("%v create order(%+v),Ext(w: %s, l: %s,timeout:%s) success", o.GetInfo(), order, order.Ext.WinPrice, order.Ext.LossPrice, utils.TimeFmt(order.Ext.Timeout, ""))
		ords = append(ords, order)
		oPrice = oPrice * (1.0 - float64(txDir)*priceIncr)
	}
	if len(ords) > 0 {
		o.appenOrders(ords)
	}
	return nil
}

// / close 为true，仅平仓
// / close false,tradeSide != none 建仓
func (o *OrderMgr) closeJudge(strateMsg StrategyMsg, curSide string) (close bool, tradeSide string) {
	tradeSide = common.TradeSideNone
	//如果当前没有持仓，就返回false

	newSide := strateMsg.StrategyRet.TradeSuggest.TradeSide
	if curSide == common.TradeSideNone {
		return
	}
	logger.Infof("%v curSide: %v,newSide: %v", strateMsg.DataSource.Symbol, curSide, newSide)
	//如果当前持仓和新策略相同，不用进行下一步操作
	if curSide == newSide {
		return
	}
	if curSide == common.TradeSideNone {
		tradeSide = newSide
		return
	}

	if newSide != common.TradeSideNone {
		close = true
		tradeSide = newSide
		return
	}

	//如果主策略为none，判断平仓策略
	if newSide == common.TradeSideNone {
		nodes := strateMsg.StrategyRet.GroupStrategyRets
		closeParams := o.params.ClosePositionParams
		close, tradeSide = closeBySpecifieds(curSide, nodes, closeParams.Specifieds)
		if close {
			logger.Infof("%v 指定策略(%+v)止损触发", o.params.Symbol, closeParams.Specifieds)
			return
		}
		curSideTimestamp := o.strateMsg.StrategyRet.TradeSuggest.CreateTime
		close, tradeSide = closeByDelays(curSide, curSideTimestamp, nodes, closeParams.Delays)
		if close {
			logger.Infof("%v 延时策略(%+v)止损触发(%v)", o.GetInfo(), closeParams.Delays, tradeSide)
		}
		return
	}
	return
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
	o.silentType(sType, silenceTimeX)
}

func (o *OrderMgr) curSide() string {
	cur := o.strateMsg.StrategyRet.TradeSuggest.TradeSide
	if cur != common.TradeSideNone {
		return cur
	}
	bp := o.params.BaseParams
	positions, err := o.accMgr.GetPositions(accmgr.PositionParams{
		BaseOrderParams: accmgr.BaseOrderParams{
			Market:   bp.Market,
			Symbol:   bp.Symbol,
			Exchange: bp.Exchange,
		},
	})
	if err != nil || len(positions) == 0 {
		return cur
	}
	logger.Infof("%v side %v position %+v", bp.Symbol, positions[0].Side, positions[0].Qty)
	return positions[0].Side
}

func (o *OrderMgr) closeVolidity(kline common.KLine) (isClose bool, err error) {
	vParams := o.params.ClosePositionParams.QuickVolidities
	if vParams == nil || len(*vParams) == 0 {
		return
	}
	curSide := o.strateMsg.StrategyRet.TradeSuggest.TradeSide
	nodes := o.strateMsg.StrategyRet.GroupStrategyRets
	close, tradeSide := closeByVolidity(curSide, nodes, vParams, kline)
	done := false
	if close {
		logger.Infof("%v 满足平仓条件Volidity ", o.GetInfo())
		done, err = o.closeOrders(tradeSide)
		if err != nil {
			logger.Errorf("%v close orders error: %v", o.GetInfo(), err)
			return
		}
		if done {
			o.strateMsg = StrategyMsg{}
			isClose = done
			o.clearOrders()
			o.silent(SilentType_CloseOrder)
		}
	}
	return
}

func NewOrderMgr(tradeParams TradeParams) *OrderMgr {
	return &OrderMgr{
		params:   tradeParams,
		isSilent: false,
		running:  false,
		orders:   make([]*accmgr.Order, 0),
		done:     make(chan bool, 1),
		accMgr:   accmgr.NewAccMgr(),
	}
}
