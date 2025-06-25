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

func ToBNPeroid(kPeriod int) string {
	switch kPeriod {
	case 1:
		return "1m"
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 120:
		return "2h"
	case 240:
		return "4h"
	case 360:
		return "6h"
	case 480:
		return "8h"
	case 720:
		return "12h"
	case 1440:
		return "1d"
	case 1440 * 7:
		return "1w"
	default:
		logger.Warnf("ToBNPeroid unknown kPeriod: %d", kPeriod)
		return "1d"
	}
}

func ToKLines(klines interface{}) []common.KLine {
	switch v := klines.(type) {
	case []*futures.Kline:
		ks := make([]common.KLine, len(v))
		for i, k := range v {
			ks[i] = ToKLine(k)
		}		
		return ks
	case []*delivery.Kline:				
		ks := make([]common.KLine, len(v))												
		for i, k := range v {															
			ks[i] = ToKLine(k)															
		}													
		return ks											
	case []*binance.Kline:									
		ks := make([]common.KLine, len(v))												
		for i, k := range v {															
			ks[i] = ToKLine(k)															
		}													
		return ks											
	}
	return []common.KLine{}
}

func ToKLine(kline interface{}) common.KLine {
	switch kline.(type) {
	case *futures.Kline:
		k := kline.(*futures.Kline)
		return common.KLine{
			Open:                  utils.ToFloat64(k.Open),
			High:                  utils.ToFloat64(k.High),
			Low:                   utils.ToFloat64(k.Low),
			Close:                 utils.ToFloat64(k.Close),
			Volume:                utils.ToFloat64(k.Volume),
			CloseTime:             k.CloseTime,
			OpenTime:              k.OpenTime,
			TradeNum:              utils.ToInt64(k.TradeNum),
			TakerBuyBaseAssetVol:  utils.ToFloat64(k.TakerBuyBaseAssetVolume),
			TakerBuyQuoteAssetVol: utils.ToFloat64(k.TakerBuyQuoteAssetVolume),
			QuoteAssetVol:         utils.ToFloat64(k.QuoteAssetVolume),
		}
	case *delivery.Kline:
		k := kline.(*delivery.Kline)
		return common.KLine{
			Open:                  utils.ToFloat64(k.Open),
			High:                  utils.ToFloat64(k.High),
			Low:                   utils.ToFloat64(k.Low),
			Close:                 utils.ToFloat64(k.Close),
			Volume:                utils.ToFloat64(k.Volume),
			CloseTime:             k.CloseTime,
			OpenTime:              k.OpenTime,
			TradeNum:              utils.ToInt64(k.TradeNum),
			TakerBuyBaseAssetVol:  utils.ToFloat64(k.TakerBuyBaseAssetVolume),
			TakerBuyQuoteAssetVol: utils.ToFloat64(k.TakerBuyQuoteAssetVolume),
			QuoteAssetVol:         utils.ToFloat64(k.QuoteAssetVolume),
		}
	case *binance.Kline:
		k := kline.(*binance.Kline)
		return common.KLine{
			Open:                  utils.ToFloat64(k.Open),
			High:                  utils.ToFloat64(k.High),
			Low:                   utils.ToFloat64(k.Low),
			Close:                 utils.ToFloat64(k.Close),
			Volume:                utils.ToFloat64(k.Volume),
			CloseTime:             k.CloseTime,
			OpenTime:              k.OpenTime,
			TradeNum:              utils.ToInt64(k.TradeNum),
			TakerBuyBaseAssetVol:  utils.ToFloat64(k.TakerBuyBaseAssetVolume),
			TakerBuyQuoteAssetVol: utils.ToFloat64(k.TakerBuyQuoteAssetVolume),
			QuoteAssetVol:         utils.ToFloat64(k.QuoteAssetVolume),
		}
	}
	logger.Warnf("ToKLine unknown kline type: %v", kline)
	return common.KLine{}
}
