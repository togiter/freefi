package accmgr

import (
	"fmt"
	"freefi/trademgr/pkg/utils"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
)

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
func FromBNCreateOrder(base BaseOrderParams, bnOrder interface{}) *Order {
	switch bnOrder.(type) {
	case *binance.CreateOrderResponse:
		ord := bnOrder.(*binance.CreateOrderResponse)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            string(ord.Side),
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
		ord := bnOrder.(*futures.CreateOrderResponse)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            string(ord.Side),
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
		ord := bnOrder.(*delivery.CreateOrderResponse)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            string(ord.Side),
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
	switch bnOrder.(type) {
	case *binance.Order:
		ord := bnOrder.(*binance.Order)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            string(ord.Side),
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
		ord := bnOrder.(*futures.Order)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            string(ord.Side),
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
		ord := bnOrder.(*delivery.Order)
		return &Order{
			ID:              ord.OrderID,
			BaseOrderParams: base,
			Side:            string(ord.Side),
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
	switch bnBalance.(type) {
	case *binance.Balance:
		pos := bnBalance.(*binance.Balance)
		return &Balance{
			BaseOrderParams: base,
			Balance:         pos.Asset,
			Frozen:          pos.Locked,
			Available:       pos.Free,
		}
	case *futures.Balance:
		pos := bnBalance.(*futures.Balance)
		return &Balance{
			BaseOrderParams: base,
			Balance:         pos.Balance,
			Frozen:          fmt.Sprintf("%.8f", utils.ToFloat64(pos.Balance)-utils.ToFloat64(pos.AvailableBalance)),
			Available:       pos.AvailableBalance,
		}
	case *delivery.Balance:
		pos := bnBalance.(*delivery.Balance)
		return &Balance{
			BaseOrderParams: base,
			Balance:         pos.Balance,
			Frozen:          fmt.Sprintf("%.8f", utils.ToFloat64(pos.Balance)-utils.ToFloat64(pos.AvailableBalance)),
			Available:       pos.AvailableBalance,
		}
	default:
		return nil
	}

}
