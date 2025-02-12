package ordermgr

import (
	"fmt"
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"math"
	"strings"
)

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

// Update implements IOrderMgr.
func (o *OrderMgr) Update(strateMsg StrategyMsg) error {
	if o.isSilent {
		logger.Info("order manager is in silent mode, ignore strategy msg")
		return nil
	}
	if strateMsg.TradeSuggest.TradeSide == common.TradeSideNone {
		logger.Info("order manager received trade suggest with trade side none, ignore")
		return nil
	}
	if strateMsg.TradeSuggest.TradeSide == o.strateMsg.TradeSuggest.TradeSide {
		logger.Infof("order manager received same trade suggest(%v), ignore", strateMsg.TradeSuggest)
		return nil
	}
	err := o.handlerOrders(strateMsg)
	if err != nil {
		logger.Errorf("handler orders error: %v", err)
		return err
	}

	//检测订单
	if o.orderMonitor.OrderEmpty() {
		logger.Info("order manager has no orders, ignore strategy msg")
		o.strateMsg = StrategyMsg{}
		return nil
	}
	o.strateMsg = strateMsg
	return nil
}

func (o *OrderMgr) getAvaliableBalance(isSell bool, sMsg StrategyMsg) (string, error) {
	pair := strings.ToUpper(sMsg.DataSource.Symbol)
	coins := strings.Split(pair, "-")
	if len(coins) == 0 {
		return "0", fmt.Errorf("invalid symbol: %s", o.params.Symbol)
	}

	sym := coins[len(coins)-1] //默认查询usdt
	if sMsg.DataSource.Market == common.MarketSpot {
		//现货 && 空单才查询现货余额
		if isSell {
			sym = coins[0] //sell,查询现货余额
		}
	}
	accs := accmgr.NewAccMgr()
	if o.params.Market != common.MarketSpot && len(o.orders) > 0 {
		//非现货先取消挂单
		err := accs.CancelOrders(accmgr.CancelOrderParams{
			BaseOrderParams: accmgr.BaseOrderParams{
				Symbol:   sMsg.DataSource.Symbol,
				Exchange: sMsg.DataSource.Exchange,
				Market:   sMsg.DataSource.Market,
			},
		})
		if err != nil {
			logger.Errorf("cancel all orders error: %v", err)
		}
	}

	bals, err := accs.GetBalances(accmgr.GetBalanceParams{
		BaseOrderParams: accmgr.BaseOrderParams{
			Symbol:   sym,
			Exchange: sMsg.DataSource.Exchange,
			Market:   sMsg.DataSource.Market,
		},
	})
	if err != nil || len(bals) == 0 {
		return "0", fmt.Errorf("get balances error or balance is 0: %v", err)
	}
	logger.Infof("getAvaliableBalanceed strategy msg: %+v", bals[0])
	return bals[0].Available, nil
}

func (o *OrderMgr) handleOrderBefore(sMsg StrategyMsg) error {
	return nil
}

func (o *OrderMgr) handlerOrders(strateMsg StrategyMsg) error {
	if err := o.handleOrderBefore(strateMsg); err != nil {
		return fmt.Errorf("handle order before error:%v", err)
	}
	bal, err := o.getAvaliableBalance(strateMsg.TradeSuggest.TradeSide == common.TradeSideSell, strateMsg)
	if err != nil || bal == "0" || bal == "" {
		logger.Errorf("get avaliable balance error/0: %v", err)
		return fmt.Errorf("get avaliable balance error/0: %v", err)
	}
	txDir := 1 //买入方向
	if strateMsg.TradeSuggest.TradeSide == common.TradeSideSell {
		txDir = -1 //卖出方向
	}
	baseParams := accmgr.BaseOrderParams{
		Symbol:   o.params.Symbol,
		Exchange: o.params.Exchange,
		Market:   o.params.Market,
	}

	//开单价格
	oPrice := strateMsg.TradeSuggest.Price
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
	if strateMsg.TradeSuggest.TradeSide == common.TradeSideBuy {
		if extBalance < math.Max(minUsdtQty, 20.0) {
			return fmt.Errorf("not enough minUsdtQty(20.0,%.5f): %.5f", minUsdtQty, extBalance)
		}

	}

	if baseParams.Market != common.MarketSpot || strateMsg.TradeSuggest.TradeSide == common.TradeSideBuy {
		canTrade = extBalance / oPrice //buy
	}
	logger.Infof("canTrade balance: %.5f,market:%s,tside:%s", canTrade, baseParams.Market, strateMsg.TradeSuggest.TradeSide)
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

	winRate := math.Max(0.005, o.params.StopWinRate)
	lossRate := math.Max(0.005, o.params.StopLossRate)
	winPrice := ""
	lossPrice := ""
	for i := 0; i < len(amounts); i++ {
		amount := amounts[i]
		qty := utils.ToFixed(amount, qtyPrecision)
		priC := utils.ToFixed(oPrice, pricePrecision)
		order, err := acc.CreateOrder(accmgr.PlaceOrderParams{
			BaseOrderParams: baseParams,
			Side:            strateMsg.TradeSuggest.TradeSide,
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
		winPrice = utils.ToFixed(oPrice*(1.0+float64(txDir)*winRate), pricePrecision)
		lossPrice = utils.ToFixed(oPrice*(1.0-float64(txDir)*lossRate), pricePrecision)

		order.Ext = &accmgr.OrderExt{
			WinPrice:    &winPrice,
			LossPrice:   &lossPrice,
			LeverRate:   &leverRate,
			IsWinOrLoss: false,
		}
		if order.BaseOrderParams.Symbol == "" {
			order.BaseOrderParams = baseParams
		}
		logger.Infof("create order(%+v) success", order)
		o.orderMonitor.AddOrder(order)
		oPrice = oPrice * (1.0 - float64(txDir)*priceIncr)

	}

	return nil
}

func NewOrderMgr(tradeParams TradeParams) IOrderMgr {
	return &OrderMgr{
		orders: make([]*accmgr.Order, 0),
		// strateMsg: strateMsg,
		params:       tradeParams,
		orderMonitor: NewOrderMonitor(),
	}
}
