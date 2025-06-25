package model


type OrderType int
const (
	OrderType_Limit OrderType = iota
	OrderType_Market
)

type OrderSide int
const (
	OrderSide_None OrderSide = iota
	OrderSide_Open_Buy
	OrderSide_Open_Sell
	OrderSide_Close_Buy //平仓买单
	OrderSide_Close_Sell //平仓卖单
)

type OrderStatus int
const (
	OrderStatus_New OrderStatus = iota
	OrderStatus_PartiallyFilled
	OrderStatus_Filled
	OrderStatus_Canceled
	OrderStatus_Rejected
)