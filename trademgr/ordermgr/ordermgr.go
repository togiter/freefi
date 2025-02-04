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

type IOrderMgr interface {
	IsDone() bool
	Stop() error
	Work() error
	Update(strateMsg StrategyMsg) error
}

type OrderMgr struct {
	orders    []*accmgr.Order
	strateMsg StrategyMsg
	stopCh    chan bool
	isWorking bool
	//沉默状态(该状态不接受策略建议)
	isSilent bool
	params   TradeParams
}

// Work implements IOrderMgr.
func (o *OrderMgr) Work() error {
	if o.isWorking {
		return fmt.Errorf("order manager is already working")
	}
	o.isWorking = true
	internalTicker := o.params.OrderStatusCheckTicker
	if internalTicker == 0 {
		internalTicker = 60 //seconds
	}
	for {
		select {
		case <-o.stopCh:
			logger.Infof("%s order manager stopped", o.params.Symbol)
			return nil
		default:
			time.Sleep(time.Duration(internalTicker) * time.Second)
			o.tickerHandler()
		}
	}
}

// Exit implements IOrderMgr.
func (o *OrderMgr) Stop() error {
	o.isWorking = false
	o.stopCh <- true
	// close(o.stopCh)
	return nil
}

// IsDone implements IOrderMgr.
func (o *OrderMgr) IsDone() bool {
	return len(o.orders) == 0
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
		logger.Error("handler orders error: %v", err)
		return err
	}
	//检测订单
	if len(o.orders) > 0 {
		o.strateMsg = strateMsg
		go o.Work()
	} else {
		logger.Info("order manager has no orders, ignore strategy msg")
		o.strateMsg = StrategyMsg{}
	}

	return nil
}

func (o *OrderMgr) getAvaliableBalance(isBuyButSell bool, sMsg StrategyMsg) (string, error) {
	coins := strings.Split(sMsg.DataSource.Symbol, "-")
	if len(coins) == 0 {
		return "0", fmt.Errorf("invalid symbol: %s", o.params.Symbol)
	}
	sym := coins[0]
	if isBuyButSell {
		sym = coins[len(coins)-1]
	}
	accs := accmgr.NewAccMgr()
	if o.params.Market != common.MarketSpot {
		//非现货先取消挂单
		err := accs.CancelOrders(accmgr.CancelOrderParams{
			BaseOrderParams: accmgr.BaseOrderParams{
				Symbol:   sMsg.DataSource.Symbol,
				Exchange: sMsg.DataSource.Exchange,
				Market:   sMsg.DataSource.Market,
			},
		})
		if err != nil {
			logger.Error("cancel all orders error: %v", err)
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
		logger.Error("get balances error/0: %v", err)
		return "0", err
	}
	logger.Infof("getAvaliableBalanceed strategy msg: %+v", bals[0])
	return bals[0].Available, nil
}

func (o *OrderMgr) handleOrderBefore(sMsg StrategyMsg) error {
	accs := accmgr.NewAccMgr()
	if o.params.Market != common.MarketSpot {
		//非现货先取消挂单
		err := accs.CancelOrders(accmgr.CancelOrderParams{
			BaseOrderParams: accmgr.BaseOrderParams{
				Symbol:   sMsg.DataSource.Symbol,
				Exchange: sMsg.DataSource.Exchange,
				Market:   sMsg.DataSource.Market,
			},
		})
		if err != nil {
			logger.Error("cancel all orders error: %v", err)
			return err
		}
	}
	return nil
}

func (o *OrderMgr) handlerOrders(strateMsg StrategyMsg) error {
	if err := o.handleOrderBefore(strateMsg); err != nil {
		return fmt.Errorf("handle order before error")
	}
	bal, err := o.getAvaliableBalance(strateMsg.TradeSuggest.TradeSide == common.TradeSideBuy, strateMsg)
	if err != nil || bal == "0" || bal == "" {
		logger.Error("get avaliable balance error/0: %v", err)
		return fmt.Errorf("get avaliable balance error/0: %v", err)
	}
	txDir := 1 //买入方向
	if strateMsg.TradeSuggest.TradeSide == common.TradeSideSell {
		txDir = -1 //卖出方向
	}
	//分批挂单数量
	maxC := o.params.OrdersCount
	//分批价格增量
	priceIncr := o.params.PriceIncr
	//分批数量增量
	amountIncr := o.params.QtyIncr
	//杠杆倍数
	leverRate := o.params.LeverRate
	//交易类型 限价/市价
	tradeT := strings.ToUpper(o.params.TradeType)
	//开单数量
	amounts := calAmountsByTotal(utils.ToFloat64(bal), amountIncr, maxC, o.params.StrategyType)
	//开单价格
	oPrice := strateMsg.TradeSuggest.Price
	//Market price参数无效
	if o.params.TradeType == common.OrderTypeLimit {
		curPrice, err := getPrice(accmgr.BaseOrderParams{
			Symbol:   o.params.Symbol,
			Exchange: o.params.Exchange,
			Market:   o.params.Market,
		})
		if err != nil {
			logger.Error("get price error: %v", err)
			return err
		}
		if curPrice > 0.00000001 { //precision handler
			oPrice = curPrice
		}
		if oPrice < 0.00000001 {
			logger.Error("invalid price: %v", oPrice)
			return fmt.Errorf("invalid price: %v", oPrice)
		}
	}
	oPrice = oPrice * (1.0 - float64(txDir)*o.params.InitPricePer)
	acc := accmgr.NewAccMgr()
	qtyPrecision := int(math.Max(2, float64(o.params.QtyPrecision)))
	pricePrecision := int(math.Max(3, float64(o.params.PricePrecision)))

	for i := 0; i < len(amounts); i++ {
		amount := amounts[i]
		order, err := acc.CreateOrder(accmgr.PlaceOrderParams{
			BaseOrderParams: accmgr.BaseOrderParams{
				Symbol:   o.params.Symbol,
				Exchange: o.params.Exchange,
				Market:   o.params.Market,
			},
			Side:      strateMsg.TradeSuggest.TradeSide,
			Type:      tradeT,
			Price:     utils.ToPrecision(oPrice, pricePrecision),
			Qty:       utils.ToPrecision(amount, qtyPrecision),
			LeverRate: leverRate,
		})
		if err != nil {
			logger.Error("create order(amount: %v, price: %v) error: %v", amount, oPrice, err)
			continue
		}
		logger.Info("create order(amount: %v, price: %v) success: %v", amount, oPrice, order)
		o.orders = append(o.orders, order)
		oPrice = oPrice * (1.0 - float64(txDir)*priceIncr)

	}
	logger.Info("handlerOrders orders: %v", o.orders)
	if len(o.orders) == 0 {
		return fmt.Errorf("create order error: %v", err)
	}
	return nil
}


func NewOrderMgr() IOrderMgr {
	return &OrderMgr{
		orders: make([]*accmgr.Order, 0),
		// strateMsg: strateMsg,
		stopCh: make(chan bool),
	}
}

func (o *OrderMgr) tickerHandler() {
	logger.Info("tickerHandler...")
	o.handlerOrderStatusChange()
	o.handlerOrderStopLossWin()
}

// 轮训订单状态变更，考虑ws替代
func (o *OrderMgr) handlerOrderStatusChange() {

}

// 检测止盈止损
func (o *OrderMgr) handlerOrderStopLossWin() {
	
}
