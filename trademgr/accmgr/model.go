package accmgr

type PlaceOrderParams struct {
	BaseOrderParams
	Type      string //limit, market
	Side      string
	Price     float64
	Qty       float64
	LeverRate float64
	StopPrice float64
	IsTest    bool
}

type BaseOrderParams struct {
	Market   string
	Exchange string
	Symbol   string
}

type CancelOrderParams struct {
	BaseOrderParams
	OrderID int64
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
