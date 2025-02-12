package accmgr

import (
	"fmt"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"strings"
)

type IAccMgr interface {
	CreateOrder(placeParams PlaceOrderParams) (*Order, error)
	CancelOrders(cancelParams CancelOrderParams) error
	GetOrders(getParams GetOrderParams) ([]*Order, error)
	CreateOrders(placeParams []PlaceOrderParams) ([]*Order, error)
	GetBalances(getParams GetBalanceParams) ([]*Balance, error)
	GetPrice(params BaseOrderParams) (float64, error)
}

type AccMgr struct {
}

// GetPositions implements IAccMgr.
func (om *AccMgr) GetBalances(getParams GetBalanceParams) ([]*Balance, error) {
	ex := strings.ToUpper(getParams.Exchange)
	logger.Infof("GetBalances for exchange:%+v", getParams)
	if ex == common.Binance {
		return GetBinanceBalances(getParams)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", getParams.Exchange)
}

func (om *AccMgr) GetPrice(params BaseOrderParams) (float64, error) {
	ex := strings.ToUpper(params.Exchange)
	if ex == common.Binance {
		return GetBinancePrice(params)
	}
	return 0.0, fmt.Errorf("Unsupported exchange:%s", params.Exchange)
}

// CreateOrders implements IAccMgr.
func (om *AccMgr) CreateOrders(placeParams []PlaceOrderParams) ([]*Order, error) {
	// ex := strings.ToUpper(placeParams[0].Exchange)
	// if ex == common.Binance {
	// 	return GetBinanceBalances(getParams)
	// }
	return nil, fmt.Errorf("Unsupported exchange")
}

// GetOrders implements IAccMgr.
func (om *AccMgr) GetOrders(getParams GetOrderParams) ([]*Order, error) {
	ex := strings.ToUpper(getParams.Exchange)
	if ex == common.Binance {
		return GetBinanceOrders(getParams)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", getParams.Exchange)
}

func NewAccMgr() IAccMgr {
	return &AccMgr{}
}

func (om *AccMgr) CancelOrders(cancelParams CancelOrderParams) error {
	ex := strings.ToUpper(cancelParams.Exchange)
	if ex == common.Binance {
		return CancelBinanceOrders(cancelParams)
	}
	return nil
}

func (om *AccMgr) CreateOrder(orderParams PlaceOrderParams) (*Order, error) {
	ex := strings.ToUpper(orderParams.BaseOrderParams.Exchange)
	logger.Info("CreateOrder for exchange:%+v", orderParams)
	if ex == common.Binance {
		return CreateBinanceOrder(orderParams)
	}

	return nil, fmt.Errorf("Unsupported exchange:%s", orderParams.Exchange)

}
