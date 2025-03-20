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
	TradeSideNone       = "NONE"
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
