package accmgr

import (
	"fmt"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"freefi/trademgr/pkg/utils"
	"math"
	"reflect"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
)

func fromBNPosition(bnOrder interface{}) *Position {
	switch ord := bnOrder.(type) {
	case *futures.PositionRiskV2:
		amount := math.Abs(utils.ToFloat64(ord.PositionAmt))
		return &Position{
			Side:             string(ord.PositionSide),
			LeverRate:        utils.ToFloat64(ord.Leverage),
			Qty:              fmt.Sprintf("%v", amount),
			EntryPrice:       ord.EntryPrice,
			MarkPrice:        ord.MarkPrice,
			UnRealizedProfit: ord.UnRealizedProfit,
			updateTime:       0,
		}
	case *futures.PositionRisk:
		amount := math.Abs(utils.ToFloat64(ord.PositionAmt))
		return &Position{
			Side:             string(ord.PositionSide),
			LeverRate:        1,
			Qty:              fmt.Sprintf("%v", amount),
			EntryPrice:       ord.EntryPrice,
			MarkPrice:        ord.MarkPrice,
			UnRealizedProfit: ord.UnRealizedProfit,
			updateTime:       ord.UpdateTime,
		}
	case *delivery.PositionRisk:
		amount := math.Abs(utils.ToFloat64(ord.PositionAmt))
		return &Position{
			Side:             string(ord.PositionSide),
			LeverRate:        utils.ToFloat64(ord.Leverage),
			Qty:              fmt.Sprintf("%v", amount),
			EntryPrice:       ord.EntryPrice,
			MarkPrice:        ord.MarkPrice,
			UnRealizedProfit: ord.UnRealizedProfit,
			updateTime:       0,
		}
	case *binance.FuturesUserPosition:
		side := common.TradeSideShort
		profit := utils.ToFloat64(ord.UnRealizedProfit)
		if profit >= 0 && utils.ToFloat64(ord.MarkPrice) > utils.ToFloat64(ord.EntryPrice) {
			side = common.TradeSideLong
		}
		amount := math.Abs(utils.ToFloat64(ord.PositionAmt))
		return &Position{
			Side:             side,
			Qty:              fmt.Sprintf("%v", amount),
			EntryPrice:       ord.EntryPrice,
			MarkPrice:        ord.MarkPrice,
			UnRealizedProfit: ord.UnRealizedProfit,
			updateTime:       0,
			LeverRate:        1,
		}
	}
	return &Position{}
}

/*
	type Order struct {
	    Symbol                   string          `json:"symbol"`
	    OrderID                  int64           `json:"orderId"`
	    OrderListId              int64           `json:"orderListId"`
	    ClientOrderID            string          `json:"clientOrderId"`
	    Price                    string          `json:"price"`
	    OrigQuantity             string          `json:"origQty"`
	    ExecutedQuantity         string          `json:"executedQty"`
	    CummulativeQuoteQuantity string          `json:"cummulativeQuoteQty"`
	    Status                   OrderStatusType `json:"status"`
	    TimeInForce              TimeInForceType `json:"timeInForce"`
		Type                     OrderType       `json:"type"`
	    Side                     SideType        `json:"side"`
	    StopPrice                string          `json:"stopPrice"`
	    IcebergQuantity          string          `json:"icebergQty"`
	    Time                     int64           `json:"time"`
	    UpdateTime               int64           `json:"updateTime"`
	    IsWorking                bool            `json:"isWorking"`
	    IsIsolated               bool            `json:"isIsolated"`
	    OrigQuoteOrderQuantity   string          `json:"origQuoteOrderQty"`
	}
*/

func turnSide(side string) string {
	switch side {
	case "BUY":
		return common.TradeSideLong
	case "SELL":
		return common.TradeSideShort
	}
	return side
}
func FromBNCreateOrder(base BaseOrderParams, bnOrder interface{}) *Order {
	switch ord := bnOrder.(type) {
	case *binance.CreateOrderResponse:
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            turnSide(string(ord.Side)),
			Type:            string(ord.Type),
			Price:           ord.Price,
			Qty:             ord.OrigQuantity,
			OriQty:          ord.OrigQuantity,
			ExecutedQty:     ord.ExecutedQuantity,
			Status:          string(ord.Status),
			Time:            ord.TransactTime,
			UpdateTime:      ord.TransactTime,
		}

	case *futures.CreateOrderResponse:
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            turnSide(string(ord.Side)),
			Type:            string(ord.Type),
			Price:           ord.Price,
			Qty:             ord.OrigQuantity,
			OriQty:          ord.OrigQuantity,
			ExecutedQty:     ord.ExecutedQuantity,
			Status:          string(ord.Status),
			Time:            ord.UpdateTime,
			UpdateTime:      ord.UpdateTime,
		}
	case *delivery.CreateOrderResponse:
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            turnSide(string(ord.Side)),
			Type:            string(ord.Type),
			Price:           ord.Price,
			Qty:             ord.OrigQuantity,
			OriQty:          ord.OrigQuantity,
			ExecutedQty:     ord.ExecutedQuantity,
			Status:          string(ord.Status),
			Time:            ord.UpdateTime,
			UpdateTime:      ord.UpdateTime,
		}
	default:
		return nil
	}
}

func FromBNOrder(base BaseOrderParams, bnOrder interface{}) *Order {
	switch ord := bnOrder.(type) {
	case *binance.Order:
		// ord := bnOrder.(*binance.Order)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            turnSide(string(ord.Side)),
			Type:            string(ord.Type),
			Price:           ord.Price,
			Qty:             ord.OrigQuantity,
			OriQty:          ord.OrigQuantity,
			ExecutedQty:     ord.ExecutedQuantity,
			Status:          string(ord.Status),
			Time:            ord.Time,
			UpdateTime:      ord.UpdateTime,
		}
	case *futures.Order:
		// ord := bnOrder.(*futures.Order)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            turnSide(string(ord.Side)),
			Type:            string(ord.Type),
			Price:           ord.Price,
			Qty:             ord.OrigQuantity,
			OriQty:          ord.OrigQuantity,
			ExecutedQty:     ord.ExecutedQuantity,
			Status:          string(ord.Status),
			Time:            ord.Time,
			UpdateTime:      ord.UpdateTime,
		}
	case *delivery.Order:
		// ord := bnOrder.(*delivery.Order)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            turnSide(string(ord.Side)),
			Type:            string(ord.Type),
			Price:           ord.Price,
			Qty:             ord.OrigQuantity,
			OriQty:          ord.OrigQuantity,
			ExecutedQty:     ord.ExecutedQuantity,
			Status:          string(ord.Status),
			Time:            ord.Time,
			UpdateTime:      ord.UpdateTime,
		}
	default:
		return nil
	}
}

func FromBNBalance(base BaseOrderParams, bnBalance interface{}) *Balance {
	switch pos := bnBalance.(type) {
	case binance.Balance:
		return &Balance{
			BaseOrderParams: base,
			Balance:         pos.Asset,
			Frozen:          pos.Locked,
			Available:       pos.Free,
		}
	case *futures.Balance:
		return &Balance{
			BaseOrderParams: base,
			Balance:         pos.Balance,
			Frozen:          fmt.Sprintf("%.8f", utils.ToFloat64(pos.Balance)-utils.ToFloat64(pos.AvailableBalance)),
			Available:       pos.AvailableBalance,
		}
	case *delivery.Balance:
		return &Balance{
			BaseOrderParams: base,
			Balance:         pos.Balance,
			Frozen:          fmt.Sprintf("%.8f", utils.ToFloat64(pos.Balance)-utils.ToFloat64(pos.AvailableBalance)),
			Available:       pos.AvailableBalance,
		}
	default:
		t := reflect.TypeOf(bnBalance)
		logger.Errorf("FromBNBalance: unknown type:%v,%v", bnBalance, t)
		return nil
	}

}
