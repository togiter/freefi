package ordermgr

type StrategyMsg struct {
	DataSource  DataSource
	StrategyRet StrategyRet
}
type StrategyRet struct {
	Combine           int                         `json:"combine"` //联合模式
	TradeSuggest      TradeSuggest                `json:"tradeSuggest"`
	GroupStrategyRets map[int64]*GroupStrategyRet `json:"groupStrategyRets"`
}

type GroupStrategyRet struct {
	TradeSuggest      TradeSuggest                 `json:"tradeSuggest"`      // 交易建议
	MicroStrategyRets map[string]*MicroStrategyRet `json:"microStrategyRets"` // name->微策略结果
	Opts              map[string]interface{}       `json:"opts"`
}
type MicroStrategyRet struct {
	TradeSuggest TradeSuggest           `json:"tradeSuggest"`
	Opts         map[string]interface{} `json:"opts"`
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

type BaseParams struct {
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
	Market   string `json:"market"` // 市场类型 spot/future_u/future_b
}

type TimeParams struct {
	KPeriod                    float64 `json:"kPeriod" yaml:"kPeriod"`                                       // K线周期(秒)
	TimeoutCancelPeriodX       float64 `json:"timeoutCancelPeriodX" yaml:"timeoutCancelPeriodX"`             // 超时取消时间(X分钟)
	ClosedOrderNoOpWaitPeriodX float64 `json:"closedOrderNoOpWaitPeriodX" yaml:"closedOrderNoOpWaitPeriodX"` // 关闭订单无操作等待时间(X分钟)
	OpenOrderNoOpWaitPeriodX   float64 `json:"openOrderNoOpWaitPeriodX" yaml:"openOrderNoOpWaitPeriodX"`     // 开仓订单无操作等待时间(X分钟)
	OrderStatusCheckTicker     int     `json:"orderStatusCheckTicker" yaml:"orderStatusCheckTicker"`         // 订单状态检查间隔(秒)
}

type TradeParams struct {
	BaseParams          `json:"baseParams" yaml:"baseParams"`
	TimeParams          `json:"timeParams" yaml:"timeParams"`
	ClosePositionParams *ClosePositionParams `json:"closePositionParams" yaml:"closePositionParams"` //平仓策略
	PositionUseRate     float64              `json:"positionUseRate" yaml:"positionUseRate"`         // 持仓使用率
	LeverRate           float64              `json:"leverRate" yaml:"leverRate"`                     // 杠杆倍数
	TradeType           string               `json:"tradeType" yaml:"tradeType"`                     // 交易类型 market/limit
	OrdersCount         int                  `json:"ordersCount" yaml:"ordersCount"`                 // 订单数量
	QtyIncr             float64              `json:"qtyIncr" yaml:"qtyIncr"`                         // 订单数量增量
	PriceIncr           float64              `json:"priceIncr" yaml:"priceIncr"`                     // 价格增量
	InitPricePer        float64              `json:"initPricePer" yaml:"initPricePer"`               // 初始价格

	QtyPrecision   int     `json:"qtyPrecision" yaml:"qtyPrecision"`     // 数量精度
	PricePrecision int     `json:"pricePrecision" yaml:"pricePrecision"` // 价格精度
	MinUsdtQty     float64 `json:"minUsdtQty" yaml:"minUsdtQty"`         //最小USDT数量
	MinTokenQty    float64 `json:"minTokenQty" yaml:"minTokenQty"`       //最小币种数量
	StrategyType   string  `json:"strategyType" yaml:"strategyType"`     // 策略名称

}

type ClosePositionParams struct {
	CloseType       int              `json:"closeType" yaml:"closeType"`
	WinRate         float64          `json:"winRate" yaml:"winRate"`
	LossRate        float64          `json:"lossRate" yaml:"lossRate"`
	TradeType       string           `json:"tradeType" yaml:"tradeType"`             // 交易类型 market/limit
	Delays          *[]Delay         `json:"delays" yaml:"delays"`                   //指标策略名称
	Specifieds      *[]Specified     `json:"specifieds" yaml:"specifieds"`           //平仓延迟时间(X分钟)
	QuickVolidities *[]QuickVolidity `json:"quickVolidities" yaml:"quickVolidities"` //快速波动幅度
}

type Delay struct {
	// Specified
	NodeKPeriod int64    `json:"nodeKPeriod" yaml:"nodeKPeriod"`
	Leaves      []string `json:"leaves" yaml:"leaves"`
	TimeX       float64  `json:"timeX" yaml:"timeX"`
}

type Specified struct {
	NodeKPeriod int64    `json:"nodeKPeriod" yaml:"nodeKPeriod"`
	Leaves      []string `json:"leaves" yaml:"leaves"`
}

type QuickVolidity struct {
	NodeKPeriod  int64   `json:"nodeKPeriod" yaml:"nodeKPeriod"`
	LimitRateX   float64 `json:"limitRateX" yaml:"limitRateX"`     //生效倍数(相对平均波幅avgVolidity)
	ReserveOrder bool    `json:"reserveOrder" yaml:"reserveOrder"` //是否反向下单
	TradeType    string  `json:"tradeType" yaml:"tradeType"`       // 交易类型 market/limit

}

type SilentType int

const (
	SilentType_None SilentType = 0
	//下单时静默
	SilentType_NewOrder SilentType = 1
	//止盈止损静默
	SilentType_CloseOrder SilentType = 2
	//超时取消
	SilentType_TimeoutCancelOrder SilentType = 3
)

func (s SilentType) String() string {
	switch s {
	case SilentType_None:
		return "None"
	case SilentType_NewOrder:
		return "NewOrder"
	case SilentType_CloseOrder:
		return "CloseOrder"
	case SilentType_TimeoutCancelOrder:
		return "TimeoutCancelOrder"
	default:
		return "Unknown"
	}
}
