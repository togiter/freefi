package common

const MinFloatValue = 0.00000001

const (
	MarketSpot    = "SPOT"
	MarketFutureU = "FUTURE_U" //U本位合约
	MarketFutureB = "FUTURE_B" //B本位合约
	// MarketType_Options // 期权
	// MarketType_Crypto // 加密货币
)
const (
	Okex    = "OKEX"
	Binance = "BINANCE"
	UniSwap = "UNISWAP"
)

const (
	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"
)

const (
	TradeSideNone       = ""
	TradeSideShort      = "SHORT"
	TradeSideLong       = "LONG"
	TradeSideCloseLong  = "CLOSE_LONG"  //平多
	TradeSideCloseShort = "CLOSE_SHORT" //平空
)

// ReserveSide 反转开平仓
func ReserveSide(tSide string) string {
	if tSide == TradeSideLong {
		return TradeSideShort
	} else if tSide == TradeSideShort {
		return TradeSideLong
	}
	return TradeSideNone
}

func SideToFuture(tSide string) string {
	if tSide == TradeSideLong {
		return "LONG"
	} else if tSide == TradeSideShort {
		return "SHORT"
	}
	return tSide
}

func SideToSpot(tSide string) string {
	if tSide == TradeSideCloseShort {
		return TradeSideLong
	} else if tSide == TradeSideCloseLong {
		return TradeSideShort
	}
	return tSide
}

const (
	StrategyNormal  = "NORMAL"
	StrategyGRID    = "GRID"
	StrategyGAMBLER = "GAMBLER"
)

const (
	OrderStatusTypeNew             = "NEW"
	OrderStatusTypePartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusTypeFilled          = "FILLED"
	OrderStatusTypeCanceled        = "CANCELED"
	OrderStatusTypePendingCancel   = "PENDING_CANCEL"
	OrderStatusTypeRejected        = "REJECTED"
	OrderStatusTypeExpired         = "EXPIRED"
	OrderStatusExpiredInMatch      = "EXPIRED_IN_MATCH" // STP Expired
)

type KLine struct {
	Open                  float64 `json:"open"`
	Close                 float64 `json:"close"`
	High                  float64 `json:"high"`
	Low                   float64 `json:"low"`
	Volume                float64 `json:"volume"`
	CloseTime             int64   `json:"closeTime"`
	OpenTime              int64   `json:"openTime"`
	TakerBuyBaseAssetVol  float64 `json:"takerBuyBaseVol"`  // 吃单基础(eg：USDT)成交量
	TakerBuyQuoteAssetVol float64 `json:"takerBuyQuoteVol"` // 吃单引用(eg:BTC)成交量
	QuoteAssetVol         float64 `json:"quoteAssetVol"`    // 引用资产成交量
	TradeNum              int64   `json:"tradeNum"`         // 交易次数
}
