package accmgr

import (
	"context"
	"fmt"
	"freefi/trademgr/common"
	"freefi/trademgr/config"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"strings"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
)

func GetBinanceBalances(getParams GetBalanceParams) ([]*Balance, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[common.Binance]
	if getParams.Market == common.MarketSpot {
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		positionRsp, err := spotCli.NewGetAccountService().Do(context.Background())
		if err != nil {
			return nil, err
		}
		balances := make([]*Balance, 0)
		for _, balance := range positionRsp.Balances {
			if utils.ToFloat64(balance.Free) <= 0.0000001 {
				continue
			}
			bal := FromBNBalance(getParams.BaseOrderParams, balance)
			if bal == nil {
				continue
			}
			logger.Infof("Binance spot balance: %s, free: %s, locked: %s", balance.Asset, balance.Free, balance.Locked)
			if strings.ToUpper(balance.Asset) == strings.ToUpper(getParams.Symbol) {
				bals := make([]*Balance, 0)
				bals = append(bals, bal)
				return bals, nil
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
			if utils.ToFloat64(balance.Balance) <= 0.0000001 {
				continue
			}
			bal := FromBNBalance(getParams.BaseOrderParams, balance)
			if bal == nil {
				continue
			}
			if len(getParams.Symbol) > 0 {
				if strings.EqualFold(strings.ToUpper(balance.Asset), strings.ToUpper(getParams.Symbol)) {
					bals := make([]*Balance, 0)
					bals = append(bals, bal)
					return bals, nil
				}
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
			bal := FromBNBalance(getParams.BaseOrderParams, balance)
			if bal == nil {
				continue
			}
			if len(getParams.Symbol) > 0 {
				if strings.EqualFold(strings.ToUpper(balance.Asset), strings.ToUpper(getParams.Symbol)) {
					bals := make([]*Balance, 0)
					bals = append(bals, bal)
					return bals, nil
				}
				return balances, nil
			}
			balances = append(balances, bal)
		}
		return balances, nil
	}
	return nil, fmt.Errorf("Not implemented")
}

func GetBinanceOrders(getParams GetOrderParams) ([]*Order, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[common.Binance]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return nil, fmt.Errorf("Binance API key or secret is not set")
	}
	symbol := strings.Replace(getParams.Symbol, "-", "", -1)
	if getParams.Market == common.MarketSpot {
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		if getParams.OrderID != nil { //only get one order
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
				orders = append(orders, FromBNOrder(getParams.BaseOrderParams, order))
			}
			return orders, nil
		}
	}
	return nil, fmt.Errorf("Unsupported market:%s", getParams.Market)
}

func CreateBinanceOrder(orderParams PlaceOrderParams) (*Order, error) {
	tradeKeys := config.GetGlobalCfg().TradeKeys[common.Binance]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return nil, fmt.Errorf("Binance API key or secret is not set")
	}
	symbol := strings.Replace(orderParams.Symbol, "-", "", -1)
	quantity := fmt.Sprintf("%v", orderParams.Qty)
	price := fmt.Sprintf("%v", orderParams.Price)
	if orderParams.Market == common.MarketSpot {
		tradeType := binance.OrderType(strings.ToUpper(orderParams.Type))
		tradeSide := binance.SideType(strings.ToUpper(orderParams.Side))
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := spotCli.NewCreateOrderService().
			Symbol(symbol).
			Side(tradeSide).
			Type(tradeType).
			TimeInForce(binance.TimeInForceTypeGTC).
			Quantity(quantity).
			Price(price)
		orderRsp, err := orderServ.Do(context.Background())
		if err != nil {
			return nil, err
		}
		return FromBNCreateOrder(orderParams.BaseOrderParams, orderRsp), nil

	} else if orderParams.Market == common.MarketFutureU {
		tradeType := futures.OrderType(strings.ToUpper(orderParams.Type))
		tradeSide := futures.SideType(strings.ToUpper(orderParams.Side))
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := futureCli.NewCreateOrderService().
			Symbol(symbol).
			Side(tradeSide).
			Type(tradeType).
			TimeInForce(futures.TimeInForceTypeGTC).
			Quantity(quantity).
			Price(price)
		orderRsp, err := orderServ.Do(context.Background())
		if err != nil {
			return nil, err
		}
		return FromBNCreateOrder(orderParams.BaseOrderParams, orderRsp), nil
	} else if orderParams.Market == common.MarketFutureB {
		tradeType := delivery.OrderType(strings.ToUpper(orderParams.Type))
		tradeSide := delivery.SideType(strings.ToUpper(orderParams.Side))
		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := futureBCli.NewCreateOrderService().
			Symbol(symbol).
			Side(tradeSide).
			Type(tradeType).
			TimeInForce(delivery.TimeInForceTypeGTC).
			Quantity(quantity).
			Price(price)
		orderRsp, err := orderServ.Do(context.Background())
		if err != nil {
			return nil, err
		}
		return FromBNCreateOrder(orderParams.BaseOrderParams, orderRsp), nil
	}
	return nil, fmt.Errorf("Unsupported market:%s", orderParams.Market)
}

func CancelBinanceOrders(cancelParams CancelOrderParams) error {
	tradeKeys := config.GetGlobalCfg().TradeKeys[common.Binance]
	if tradeKeys.APIKey == "" || tradeKeys.SecretKey == "" {
		return fmt.Errorf("Binance API key or secret is not set")
	}
	symbol := strings.Replace(cancelParams.Symbol, "-", "", -1)
	orderId := cancelParams.OrderID
	if cancelParams.Market == common.MarketSpot {
		// if orderParams.IsTest {
		// 	binance.UseTestNet = true
		// }
		spotCli := binance.NewClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		orderServ := spotCli.NewCancelOrderService().
			Symbol(symbol).
			OrderID(orderId)
		_, err := orderServ.Do(context.Background())
		if err != nil {
			return err
		}
		return nil
	} else if cancelParams.Market == common.MarketFutureU {
		futureCli := binance.NewFuturesClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		_, err := futureCli.NewCancelOrderService().
			Symbol(symbol).
			OrderID(orderId).Do(context.Background())
		if err != nil {
			return err
		}
		return nil
	} else if cancelParams.Market == common.MarketFutureB {
		futureBCli := binance.NewDeliveryClient(tradeKeys.APIKey, tradeKeys.SecretKey)
		_, err := futureBCli.NewCancelOrderService().
			Symbol(symbol).
			OrderID(orderId).Do(context.Background())
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("Unsupported market:%s", cancelParams.Market)
}
