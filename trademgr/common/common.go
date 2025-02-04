package common

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
	TradeSideBuy  = "BUY"
	TradeSideSell = "SELL"
	TradeSideNone = "NONE"
)

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
