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
	CloseOrders(closeParams CloseOrderParams) error
	GetBalances(getParams GetBalanceParams) ([]*Balance, error)
	GetPrice(params BaseOrderParams) (float64, error)
	GetKLines(params KLineParams) ([]common.KLine, error)
	GetPositions(params PositionParams) ([]*Position, error)
}

type AccMgr struct {
}

func (om *AccMgr) GetKLines(params KLineParams) ([]common.KLine, error) {
	ex := strings.ToUpper(params.Exchange)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return getBinanceKLines(params)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", params.Exchange)
}

func (om *AccMgr) GetPositions(params PositionParams) ([]*Position, error) {
	ex := strings.ToUpper(params.Exchange)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return GetBinancePositions(params)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", params.Exchange)
}

// GetPositions implements IAccMgr.
func (om *AccMgr) GetBalances(getParams GetBalanceParams) ([]*Balance, error) {
	ex := strings.ToUpper(getParams.Exchange)
	logger.Infof("GetBalances for exchange:%+v", getParams)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return GetBinanceBalances(getParams)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", getParams.Exchange)
}

func (om *AccMgr) GetPrice(params BaseOrderParams) (float64, error) {
	ex := strings.ToUpper(params.Exchange)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
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
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return GetBinanceOrders(getParams)
	}
	return nil, fmt.Errorf("Unsupported exchange:%s", getParams.Exchange)
}

func NewAccMgr() IAccMgr {
	return &AccMgr{}
}

func (om *AccMgr) CancelOrders(cancelParams CancelOrderParams) error {
	ex := strings.ToUpper(cancelParams.Exchange)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return CancelBinanceOrders(cancelParams)
	}
	return nil
}

func (om *AccMgr) CreateOrder(orderParams PlaceOrderParams) (*Order, error) {
	ex := strings.ToUpper(orderParams.BaseOrderParams.Exchange)
	logger.Infof("CreateOrder for exchange:%+v", orderParams)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return CreateBinanceOrder(orderParams)
	}

	return nil, fmt.Errorf("unsupported exchange:%s", orderParams.Exchange)

}

func (om *AccMgr) CloseOrders(closeParams CloseOrderParams) error {
	ex := strings.ToUpper(closeParams.BaseOrderParams.Exchange)
	logger.Info("closeParams for exchange:%+v", closeParams)
	if strings.HasPrefix(ex, strings.ToUpper(common.Binance)) {
		return CloseBinanceOrders(closeParams)
	}

	return fmt.Errorf("Unsupported exchange:%s", closeParams.Exchange)

}
