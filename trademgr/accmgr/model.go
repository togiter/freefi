package accmgr

type PlaceOrderParams struct {
	BaseOrderParams
	Type      string //limit, market
	Side      string
	Price     string
	Qty       string
	LeverRate float64
	StopPrice *string
	IsTest    bool
	IsClose   *bool //是否平仓
}

type PositionParams struct {
	BaseOrderParams
	Side *string
}

type Position struct {
	BaseOrderParams
	Side             string
	Qty              string
	EntryPrice       string
	MarkPrice        string //标记价格
	UnRealizedProfit string //持仓盈亏
	updateTime       int64
	LeverRate        float64
}

type CloseOrderParams struct {
	BaseOrderParams
	TradeType string
	TradeSide string
	StopPrice *string
	Qty       *string //for spot
}
type BaseOrderParams struct {
	Market   string
	Exchange string
	Symbol   string
}

type CancelOrderParams struct {
	BaseOrderParams
	OrderID *int64
}

type GetOrderParams struct {
	BaseOrderParams
	OrderID *int64  //为空则获取所有订单
	Status  *string //为空则获取所有状态
	Limit   int     //为空则获取所有订单
}

type Order struct {
	BaseOrderParams
	ID          int64
	Type        string //limit, market
	Side        string
	Price       string
	Qty         string
	OriQty      string
	ExecutedQty string
	Status      string // NEW, PARTIALLY_FILLED, FILLED, CANCELED, PENDING_CANCEL, REJECTED, EXPIRED, EXPIRED_IN_MATCH
	Time        int64
	UpdateTime  int64
	Ext         *OrderExt
}

type OrderExt struct {
	WinPrice  string //止盈止损价格，如果有
	LossPrice string //止盈止损价格，如果有
	LeverRate float64
	Timeout   int64 //订单超时时间，如果有
}

type GetBalanceParams struct {
	BaseOrderParams
}

type Balance struct {
	BaseOrderParams
	Balance   string
	Available string
	Frozen    string
}

type Base struct {
	Exchange string
	Market   string
	Symbol   string
	Limit    int
	Period   int
}

type KLineParams struct {
	Base
	// StartTime int64
	// EndTime   int64
}
