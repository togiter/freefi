package accmgr

import (
	"context"
	"fmt"
	"freefi/trademgr/common"
	"freefi/trademgr/config"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"math"
	"strings"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
)

func GetBinancePositions(params PositionParams) ([]*Position, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[params.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return nil, fmt.Errorf("binance API key or secret is not set")
	}
	tSide := params.Side
	symbol := strings.Replace(params.BaseOrderParams.Symbol, "-", "", -1)
	if params.Market == common.MarketSpot {
		params.Symbol = strings.Split(params.Symbol, "-")[0]
		bals, err := GetBinanceBalances(GetBalanceParams{BaseOrderParams: params.BaseOrderParams})
		if err != nil || len(bals) == 0 {
			return nil, fmt.Errorf("%v GetBinanceBalances failed:%v", params.Symbol, err)
		}
		bal := bals[0]
		poss := make([]*Position, 0)
		tSide := common.TradeSideNone
		if utils.ToFloat64(bal.Available) > common.MinFloatValue {
			tSide = common.TradeSideLong
		}
		poss = append(poss, &Position{
			BaseOrderParams:  params.BaseOrderParams,
			Side:             tSide,
			Qty:              bal.Available,
			EntryPrice:       "0",
			MarkPrice:        "0",
			UnRealizedProfit: "0",
			LeverRate:        0.0,
		})
		return poss, nil

	} else if params.Market == common.MarketFutureU {
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		positionRsp, err := futureCli.NewGetPositionRiskV2Service().Symbol(symbol).Do(context.Background())
		if err != nil {
			return nil, err
		}
		if len(positionRsp) == 0 {
			return nil, fmt.Errorf("%v position not found", params.Symbol)
		}
		posss := make([]*Position, 0)
		for _, position := range positionRsp {
			amount := math.Abs(utils.ToFloat64(position.PositionAmt))
			if amount <= common.MinFloatValue {
				continue
			}
			if tSide != nil && *tSide != position.PositionSide {
				continue
			}
			posss = append(posss, fromBNPosition(position))
		}
		return posss, nil
	} else if params.Market == common.MarketFutureB {
		deliveryCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		positionRsp, err := deliveryCli.NewGetPositionRiskService().Pair(symbol).Do(context.Background())
		if err != nil {
			return nil, err
		}
		if len(positionRsp) == 0 {
			return nil, fmt.Errorf("%v Market FutureB not found", params.Symbol)
		}
		posss := make([]*Position, 0)
		for _, position := range positionRsp {
			amount := math.Abs(utils.ToFloat64(position.PositionAmt))
			if amount <= common.MinFloatValue {
				continue
			}
			if tSide != nil && *tSide != position.PositionSide {
				continue
			}
			posss = append(posss, fromBNPosition(position))
		}
		return posss, nil
	}
	return nil, fmt.Errorf("not implemented")
}
func GetBinancePrice(getParams BaseOrderParams) (float64, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[getParams.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return 0.0, fmt.Errorf("binance API key or secret is not set")
	}
	deepLimit := 5
	symbol := strings.Replace(getParams.Symbol, "-", "", -1)
	if getParams.Market == common.MarketSpot {
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		price, err := spotCli.NewDepthService().Limit(deepLimit).Symbol(symbol).Do(context.Background()) //.NewAveragePriceService().Symbol(symbol).Do(context.Background())
		if err != nil {
			return 0.0, err
		}
		return utils.ToFloat64(price.Asks[len(price.Asks)-1].Price), nil
	} else if getParams.Market == common.MarketFutureU || getParams.Market == common.MarketFutureB {

		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		price, err := futureCli.NewDepthService().Limit(deepLimit).Symbol(symbol).Do(context.Background()) //futureCli.NewDeliveryPriceService().Pair(symbol).Do(context.Background())
		if err != nil {
			return 0.0, err
		}
		return utils.ToFloat64(price.Asks[len(price.Asks)-1].Price), nil
	}
	return 0.0, fmt.Errorf("not implemented")
}
func GetBinanceBalances(getParams GetBalanceParams) ([]*Balance, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[getParams.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return nil, fmt.Errorf("binance API key or secret is not set")
	}
	if getParams.Market == common.MarketSpot {
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		positionRsp, err := spotCli.NewGetAccountService().OmitZeroBalances(true).Do(context.Background())
		if err != nil {
			return nil, err
		}
		balances := make([]*Balance, 0)
		for _, balance := range positionRsp.Balances {
			if len(getParams.Symbol) > 0 {
				// logger.Infof("(%s, %s)balance=>%+v", balance.Asset, getParams.Symbol, balance)
				if strings.EqualFold(strings.ToUpper(balance.Asset), strings.ToUpper(getParams.Symbol)) {
					bals := make([]*Balance, 0)
					bal := FromBNBalance(getParams.BaseOrderParams, balance)
					if bal == nil {
						return nil, fmt.Errorf("%v FromBNBalance failed:%v", getParams, balance)
					}
					bals = append(bals, bal)
					return bals, nil
				}
				continue
			}
			bal := FromBNBalance(getParams.BaseOrderParams, balance)
			if bal == nil {
				continue
			}
			balances = append(balances, bal)

		}
		return balances, nil
	} else if getParams.Market == common.MarketFutureU || getParams.Market == common.MarketFutureB {
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		positionRsp, err := futureCli.NewGetBalanceService().Do(context.Background())
		if err != nil {
			return nil, err
		}
		balances := make([]*Balance, 0)
		for _, balance := range positionRsp {
			logger.Infof("%s %s balance=>%+v", getParams.Market, balance.Asset, balance)
			if utils.ToFloat64(balance.Balance) <= 0.0000001 {
				continue
			}

			if len(getParams.Symbol) > 0 {
				if strings.EqualFold(strings.ToUpper(balance.Asset), strings.ToUpper(getParams.Symbol)) {
					bals := make([]*Balance, 0)
					bal := FromBNBalance(getParams.BaseOrderParams, balance)
					if bal == nil {
						return nil, fmt.Errorf("NewFuturesClient FromBNBalance failed:%v", balance)
					}
					bals = append(bals, bal)
					return bals, nil
				}
			}
			bal := FromBNBalance(getParams.BaseOrderParams, balance)
			if bal == nil {
				continue
			}
			balances = append(balances, bal)
		}
		return balances, nil
	} else if getParams.Market == common.MarketFutureB {

		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		positionRsp, err := futureBCli.NewGetBalanceService().Do(context.Background())
		if err != nil {
			return nil, err
		}
		balances := make([]*Balance, 0)
		for _, balance := range positionRsp {
			if utils.ToFloat64(balance.Balance) <= 0.0000001 {
				continue
			}
			logger.Infof("%s %s balance=>%+v", getParams.Market, balance.Asset, balance)

			if len(getParams.Symbol) > 0 {
				if strings.EqualFold(strings.ToUpper(balance.Asset), strings.ToUpper(getParams.Symbol)) {
					bals := make([]*Balance, 0)
					bal := FromBNBalance(getParams.BaseOrderParams, balance)
					if bal == nil {
						return nil, fmt.Errorf(" NewDeliveryClient FromBNBalance failed:%v", balance)
					}
					bals = append(bals, bal)
					return bals, nil
				}
				return balances, nil
			}
			bal := FromBNBalance(getParams.BaseOrderParams, balance)
			if bal == nil {
				continue
			}
			balances = append(balances, bal)
		}
		return balances, nil
	}
	return nil, fmt.Errorf("not implemented")
}

func GetBinanceOrders(getParams GetOrderParams) ([]*Order, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[getParams.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return nil, fmt.Errorf("binance API key or secret is not set")
	}
	symbol := strings.Replace(getParams.Symbol, "-", "", -1)
	if getParams.Market == common.MarketSpot {
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		if getParams.OrderID != nil {
			orderRsp, err := spotCli.NewGetOrderService().
				Symbol(symbol).
				OrderID(*getParams.OrderID).Do(context.Background())
			if err != nil {
				return nil, err
			}
			return []*Order{FromBNOrder(getParams.BaseOrderParams, orderRsp)}, nil
		} else {
			if getParams.Limit == 0 {
				getParams.Limit = 1000
			}
			var orderRsp []*binance.Order
			var err error
			if getParams.Status != nil {
				orderRsp, err = spotCli.NewListOpenOrdersService().
					Symbol(symbol).Do(context.Background())
				if err != nil {
					return nil, err
				}

			} else {
				orderRsp, err = spotCli.NewListOrdersService().
					Symbol(symbol).Do(context.Background())
				if err != nil {
					return nil, err
				}
			}
			orders := make([]*Order, 0)
			for _, order := range orderRsp {
				logger.Infof("order(%s)=>%+v", *getParams.Status, order)

				if getParams.Status != nil {
					if strings.EqualFold(strings.ToUpper(string(order.Status)), strings.ToUpper(*getParams.Status)) {
						orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
					}
					continue
				}
				orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
			}
			return orders, nil
		}
	} else if getParams.Market == common.MarketFutureU {
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		if getParams.OrderID != nil { //only get one order
			orderRsp, err := futureCli.NewGetOrderService().
				Symbol(symbol).
				OrderID(*getParams.OrderID).Do(context.Background())
			if err != nil {
				return nil, err
			}
			return []*Order{FromBNOrder(getParams.BaseOrderParams, orderRsp)}, nil
		} else {
			if getParams.Limit == 0 {
				getParams.Limit = 1000
			}
			var orderRsp []*futures.Order
			var err error
			if getParams.Status != nil {
				orderRsp, err = futureCli.NewListOpenOrdersService().
					Symbol(symbol).Do(context.Background())
				if err != nil {
					return nil, err
				}

			} else {
				orderRsp, err = futureCli.NewListOrdersService().
					Symbol(symbol).Do(context.Background())
				if err != nil {
					return nil, err
				}
			}
			orders := make([]*Order, 0)
			for _, order := range orderRsp {
				logger.Infof("u order(%s)=>%+v", *getParams.Status, order)

				if getParams.Status != nil {
					if strings.EqualFold(strings.ToUpper(string(order.Status)), strings.ToUpper(*getParams.Status)) {
						orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
					}
					continue
				}
				orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
			}
			return orders, nil
		}

	} else if getParams.Market == common.MarketFutureB {
		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		if getParams.OrderID != nil { //only get one order
			orderRsp, err := futureBCli.NewGetOrderService().
				Symbol(symbol).
				OrderID(*getParams.OrderID).Do(context.Background())
			if err != nil {
				return nil, err
			}
			return []*Order{FromBNOrder(getParams.BaseOrderParams, orderRsp)}, nil
		} else {
			if getParams.Limit == 0 {
				getParams.Limit = 1000
			}
			var orderRsp []*delivery.Order
			var err error
			if getParams.Status != nil {
				orderRsp, err = futureBCli.NewListOpenOrdersService().
					Symbol(symbol).Do(context.Background())
				if err != nil {
					return nil, err
				}

			} else {
				orderRsp, err = futureBCli.NewListOrdersService().
					Symbol(symbol).Do(context.Background())
				if err != nil {
					return nil, err
				}
			}
			orders := make([]*Order, 0)
			for _, order := range orderRsp {
				logger.Infof("b order(%s)=>%+v", *getParams.Status, order)
				if getParams.Status != nil {
					if strings.EqualFold(strings.ToUpper(string(order.Status)), strings.ToUpper(*getParams.Status)) {
						orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
					}
					continue
				}
				orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
			}
			return orders, nil
		}
	}
	return nil, fmt.Errorf("unsupported market:%s", getParams.Market)
}

func CreateBinanceOrder(orderParams PlaceOrderParams) (*Order, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[orderParams.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return nil, fmt.Errorf("binance API key or secret is not set")
	}
	if orderParams.Symbol == "" {
		return nil, fmt.Errorf("%v Symbol is not set", orderParams)
	}
	logger.Infof("CreateBinanceOrder keys:%+v", tradeKeys)
	symbol := strings.Replace(orderParams.Symbol, "-", "", -1)
	quantity := fmt.Sprintf("%v", orderParams.Qty)
	price := fmt.Sprintf("%v", orderParams.Price)
	if orderParams.Market == common.MarketSpot {
		tradeType := binance.OrderType(strings.ToUpper(orderParams.Type))
		tradeSide := binance.SideTypeBuy
		if orderParams.Side == common.TradeSideShort || orderParams.Side == common.TradeSideCloseLong {
			tradeSide = binance.SideTypeSell
		}
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := spotCli.NewCreateOrderService().
			Symbol(symbol).
			Side(tradeSide).
			Type(tradeType).
			Quantity(quantity)
		if orderParams.StopPrice != nil && utils.ToFloat64(*orderParams.StopPrice) > 0.00000001 {
			orderServ.StopPrice(*orderParams.StopPrice)
		}
		if tradeType == binance.OrderTypeLimit {
			orderServ.Price(price).TimeInForce(binance.TimeInForceTypeGTC)
		}
		orderRsp, err := orderServ.Do(context.Background())
		if err != nil {
			return nil, err
		}
		return FromBNCreateOrder(orderParams.BaseOrderParams, orderRsp), nil

	} else if orderParams.Market == common.MarketFutureU {
		tradeType := futures.OrderType(strings.ToUpper(orderParams.Type))
		tradeSide := futures.SideTypeBuy
		poiSide := futures.PositionSideTypeLong
		if orderParams.Side == common.TradeSideShort || orderParams.Side == common.TradeSideCloseLong {
			tradeSide = futures.SideTypeSell
			poiSide = futures.PositionSideTypeShort
		}
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := futureCli.NewCreateOrderService().
			Symbol(symbol).
			PositionSide(poiSide).
			Side(tradeSide).
			Type(tradeType).
			TimeInForce(futures.TimeInForceTypeGTC).
			Quantity(quantity).
			Price(price)
		if orderParams.StopPrice != nil && utils.ToFloat64(*orderParams.StopPrice) > 0.00000001 {
			orderServ.StopPrice(*orderParams.StopPrice)
		}
		orderRsp, err := orderServ.ClosePosition(orderParams.IsClose != nil).Do(context.Background())
		if err != nil {
			return nil, err
		}
		return FromBNCreateOrder(orderParams.BaseOrderParams, orderRsp), nil
	} else if orderParams.Market == common.MarketFutureB {
		tradeType := delivery.OrderType(strings.ToUpper(orderParams.Type))
		tradeSide := delivery.SideTypeBuy
		if orderParams.Side == common.TradeSideShort || orderParams.Side == common.TradeSideCloseLong {
			tradeSide = delivery.SideTypeSell
		}

		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := futureBCli.NewCreateOrderService().
			Symbol(symbol).
			Side(tradeSide).
			Type(tradeType).
			TimeInForce(delivery.TimeInForceTypeGTC).
			Quantity(quantity).
			Price(price)
		if orderParams.StopPrice != nil && utils.ToFloat64(*orderParams.StopPrice) > 0.00000001 {
			orderServ.StopPrice(*orderParams.StopPrice)
		}
		orderRsp, err := orderServ.ClosePosition(orderParams.IsClose != nil).Do(context.Background())
		if err != nil {
			return nil, err
		}
		return FromBNCreateOrder(orderParams.BaseOrderParams, orderRsp), nil
	}
	return nil, fmt.Errorf("unsupported market:%s", orderParams.Market)
}

func CancelBinanceOrders(cancelParams CancelOrderParams) error {
	tradeKeys := config.GetGlobalCfg().TradeKeys[cancelParams.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return fmt.Errorf("binance API key or secret is not set")
	}
	symbol := strings.Replace(cancelParams.Symbol, "-", "", -1)
	orderId := cancelParams.OrderID
	if cancelParams.Market == common.MarketSpot {
		// if orderParams.IsTest {
		// 	binance.UseTestNet = true
		// }
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)

		if orderId != nil && *orderId > 0 {
			_, err := spotCli.NewCancelOrderService().
				Symbol(symbol).OrderID(*orderId).Do(context.Background())
			if err != nil {
				return err
			}
			return nil
		} else {
			_, err := spotCli.NewCancelOpenOrdersService().Symbol(symbol).Do(context.Background())
			if err != nil {
				return err
			}
			return nil
		}
	} else if cancelParams.Market == common.MarketFutureU {
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		if orderId != nil && *orderId > 0 {
			_, err := futureCli.NewCancelOrderService().
				Symbol(symbol).OrderID(*orderId).Do(context.Background())
			if err != nil {
				return err
			}
			return nil
		} else {
			err := futureCli.NewCancelAllOpenOrdersService().
				Symbol(symbol).Do(context.Background())
			if err != nil {
				return err
			}
			return nil
		}
	} else if cancelParams.Market == common.MarketFutureB {
		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)

		if orderId != nil && *orderId > 0 {
			_, err := futureBCli.NewCancelOrderService().
				Symbol(symbol).OrderID(*orderId).Do(context.Background())
			if err != nil {
				return err
			}
			return nil
		} else {
			err := futureBCli.NewCancelAllOpenOrdersService().
				Symbol(symbol).Do(context.Background())
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("unsupported market:%s", cancelParams.Market)
}

// CloseBinanceOrders close position 关闭相反方向的仓位
func CloseBinanceOrders(closeParams CloseOrderParams) error {
	if closeParams.Qty == nil || utils.ToFloat64(*closeParams.Qty) <= 0.0000001 {
		return fmt.Errorf("qty is nil")
	}
	tradeKeys := config.GetGlobalCfg().TradeKeys[closeParams.Exchange]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return fmt.Errorf("binance API key or secret is not set")
	}
	if closeParams.Symbol == "" {
		return fmt.Errorf("%v Symbol is not set", closeParams)
	}
	logger.Infof("CreateBinanceOrder keys:%+v", tradeKeys)
	symbol := strings.Replace(closeParams.Symbol, "-", "", -1)
	if closeParams.Market == common.MarketSpot {

		qty := "0"
		if closeParams.Qty == nil {
			closeParams.Symbol = strings.Split(closeParams.Symbol, "-")[0]
			logger.Info("CloseCancelBinanceOrders spot symbol:", closeParams.Symbol)
			bals, err := GetBinanceBalances(GetBalanceParams{
				BaseOrderParams: closeParams.BaseOrderParams,
			})
			if err != nil || len(bals) == 0 {
				return err
			}
			qty = bals[0].Available
			if utils.ToFloat64(qty) < common.MinFloatValue {
				return fmt.Errorf("%v 平仓余额(%v)不足", closeParams.Symbol, qty)
			}
		} else {
			qty = *(closeParams.Qty)
		}
		tradeT := binance.OrderTypeMarket
		if closeParams.TradeType == common.OrderTypeLimit {
			tradeT = binance.OrderTypeLimit
		}
		logger.Infof("现货%v 平仓 交易类型 %v 交易数量 %v",closeParams.Symbol,tradeT,qty)
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := spotCli.NewCreateOrderService().
			Symbol(symbol).
			Side(binance.SideTypeSell).
			Type(tradeT).
			Quantity(qty)
		tradeType := binance.OrderType(closeParams.TradeType)
		if tradeType == binance.OrderTypeLimit && closeParams.StopPrice != nil {
			orderServ.Price(*(closeParams.StopPrice)).TimeInForce(binance.TimeInForceTypeGTC)
		}
		orderRsp, err := orderServ.Do(context.Background())
		if err != nil {
			return err
		}
		logger.Info("CloseCancelBinanceOrders spot orderRsp:", orderRsp)
		return nil

	} else if closeParams.Market == common.MarketFutureU {
		side := futures.SideTypeSell
		pSide := futures.PositionSideTypeLong
		if closeParams.TradeSide == common.TradeSideLong || closeParams.TradeSide == common.TradeSideCloseShort {
			side = futures.SideTypeBuy
			pSide = futures.PositionSideTypeShort
		}
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := futureCli.NewCreateOrderService().PositionSide(pSide).
			Symbol(symbol).Side(side).Type(futures.OrderTypeMarket).Quantity(*closeParams.Qty)
		orderRsp, err := orderServ.ClosePosition(false).Do(context.Background())
		if err != nil {
			return fmt.Errorf("u  Close position error:%v", err)
		}
		logger.Infof("CloseCancelBinanceOrders u orderRsp: %v", orderRsp)
		return nil

	} else if closeParams.Market == common.MarketFutureB {
		side := delivery.SideTypeSell
		pSide := delivery.PositionSideTypeLong
		if closeParams.TradeSide == common.TradeSideLong || closeParams.TradeSide == common.TradeSideCloseShort {
			side = delivery.SideTypeBuy
			pSide = delivery.PositionSideTypeShort
		}
		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := futureBCli.NewCreateOrderService().PositionSide(pSide).
			Symbol(symbol).Side(side).Type(delivery.OrderTypeMarket).Quantity(*(closeParams.Qty))

		orderRsp, err := orderServ.ClosePosition(true).Do(context.Background())
		if err != nil {
			return fmt.Errorf("b  Close position error:%v", err)
		}
		logger.Info("%v B CloseCancelBinanceOrders b orderRsp:", closeParams, orderRsp)
		return nil
	}
	return fmt.Errorf("unsupported market:%s", closeParams.Market)
}

func getBinanceKLines(kLineParams KLineParams) ([]common.KLine, error) {
	syms := strings.Split(kLineParams.Symbol, "-")
	sLen := len(syms)
	if sLen == 2 && syms[sLen-1] == "USD" {
		syms[sLen-1] = "USDT"
	}
	kLineParams.Symbol = strings.Trim(strings.Join(syms, ""), " ")
	if kLineParams.Market == common.MarketFutureU {
		fCli := binance.NewFuturesClient("", "")
		klines, err := fCli.NewKlinesService().
			Symbol(kLineParams.Symbol).
			Interval(ToBNPeroid(kLineParams.Period)).
			Limit(kLineParams.Limit).
			Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("%v FUTURE-U failed to get klines: %s", kLineParams, err)
		}
		return ToKLines(klines), nil
	} else if kLineParams.Market == common.MarketFutureB {
		fCliB := binance.NewFuturesClient("", "") //todo:: 币本位合约获取K线失败:code=-1121, msg=Invalid symbol
		klines, err := fCliB.NewKlinesService().
			Symbol(kLineParams.Symbol).
			Interval(ToBNPeroid(kLineParams.Period)).
			Limit(kLineParams.Limit).
			Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("%v FUTURE-B failed to get klines: %s", kLineParams, err)
		}
		return ToKLines(klines), nil
	} else if kLineParams.Market == common.MarketSpot {
		cli := binance.NewClient("", "")
		klines, err := cli.NewKlinesService().
			Symbol(kLineParams.Symbol).
			Interval(ToBNPeroid(kLineParams.Period)).
			Limit(kLineParams.Limit).
			Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get klines: %s", err)
		}
		return ToKLines(klines), nil
	}
	return nil, fmt.Errorf("unsupported market: %s", kLineParams.Market)
}
