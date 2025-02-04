package model

type Order struct {
	ID        int64
	Symbol    string
	Side      string
	Type      string
	Price     float64
	Quantity  float64
	Timestamp int64
}

type OrderBook struct {
	Symbol string
	Bids   []Order
	Asks   []Order
}


//策略结果数据
type StrategySource struct {

}

//策略参数
type StrategyGroup struct {
	Name      string
	Stragegies []Strategy
}

type Strategy struct {
	Name      string
	Params    map[string]interface{} 
}

//策略建议
type StrategySugguestion struct {
	TradeType OrderSide
	Symbol    string
	Side      string
}

type StrategyRet struct {
	DataSource StrategySource
	StragegyGroups []StrategyGroup

}