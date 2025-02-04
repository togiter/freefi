package accmgr

import (
	"fmt"
	"freefi/trademgr/common"
	"strings"
)

type IAccMgr interface {
	CreateOrder(placeParams PlaceOrderParams) (*Order, error)
	CancelOrders(cancelParams CancelOrderParams) error
	GetOrders(getParams GetOrderParams) ([]*Order, error)
	CreateOrders(placeParams []PlaceOrderParams) ([]*Order, error)
	GetBalances(getParams GetBalanceParams) ([]*Balance, error)
}

type AccMgr struct {
}

// GetPositions implements IAccMgr.
func (om *AccMgr) GetBalances(getParams GetBalanceParams) ([]*Balance, error) {
	ex := strings.ToUpper(getParams.Exchange)
	if ex == common.Binance {
		return GetBinanceBalances(getParams)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", getParams.Exchange)
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
	ex := strings.ToUpper(orderParams.Exchange)
	if ex == common.Binance {
		return CreateBinanceOrder(orderParams)
	}

	return nil, fmt.Errorf("Unsupported exchange:%s", orderParams.Exchange)

}
