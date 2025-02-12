package ordermgr

type StrategyMsg struct {
	DataSource   DataSource
	TradeSuggest TradeSuggest
}

type TradeSuggest struct {
	//多空决策
	TradeSide  string  `json:"tradeSide"`
	FomoLevel  int     `json:"fomoLevel"`
	Mark       string  `json:"mark"`
	Price      float64 `json:"price"`
	CreateTime int64   `json:"createTime"`
}

type DataSource struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Market   string `json:"market"`
	Limit    int    `json:"limit"`
	Ticker   int    `json:"ticker"`
}

type TradeParams struct {
	KPeriod                    int64   `json:"period"` // mins
	Symbol                     string  `json:"symbol"`
	Exchange                   string  `json:"exchange"`
	Market                     string  `json:"market"`                     // 市场类型 spot/future_u/future_b
	StrategyType               string  `json:"strategyType"`               // 策略名称
	PositionUseRate            float64 `json:"positionUseRate"`            // 持仓使用率
	LeverRate                  float64 `json:"leverRate"`                  // 杠杆倍数
	TradeType                  string  `json:"tradeType"`                  // 交易类型 market/limit
	StopLossRate               float64 `json:"stopLossRate"`               // 止损比例
	StopWinRate                float64 `json:"stopWinRate"`                // 止盈比例
	OrdersCount                int     `json:"ordersCount"`                // 订单数量
	QtyIncr                    float64 `json:"qtyIncr"`                    // 订单数量增量
	PriceIncr                  float64 `json:"priceIncr"`                  // 价格增量
	InitPricePer               float64 `json:"initPricePer"`               // 初始价格
	TimeoutCancelPeriodX       float64 `json:"timeoutCancelPeriodX"`       // 超时取消时间(X分钟)
	ClosedOrderNoOpWaitPeriodX float64 `json:"closedOrderNoOpWaitPeriodX"` // 关闭订单无操作等待时间(X分钟)
	OpenOrderNoOpWaitPeriodX   float64 `json:"openOrderNoOpWaitPeriodX"`   // 开仓订单无操作等待时间(X分钟)
	OrderStatusCheckTicker     int     `json:"orderStatusCheckTicker"`     // 订单状态检查间隔(秒)
	QtyPrecision               int     `json:"qtyPrecision"`               // 数量精度
	PricePrecision             int     `json:"pricePrecision"`             // 价格精度
	MinUsdtQty                 float64 `json:"minUsdtQty"`                 // 最小USDT数量
}
