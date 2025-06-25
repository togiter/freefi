package common

type TradeSide string

const (
	TradeSideNone  TradeSide = ""
	TradeSideLong  TradeSide = "LONG"
	TradeSideShort TradeSide = "SHORT"
)

const (
	FomoLevelNormal = 0
	FomoLevelWeak   = -1
	FomoLevelStrong = 1
)

type TradeSuggest struct {
	//多空决策
	TradeSide  TradeSide `json:"tradeSide"`
	FomoLevel  int       `json:"fomoLevel"`
	Mark       string    `json:"mark"`
	Price      float64   `json:"price"`
	CreateTime int64     `json:"createTime"`
}

func (ts *TradeSuggest) StrictFomoLevel(fomoLevel int) (p TradeSide) {
	defer func() {
		ts.TradeSide = p
	}()

	p = ts.TradeSide
	if ts.FomoLevel == fomoLevel || fomoLevel == FomoLevelWeak {
		return
	}
	if fomoLevel == FomoLevelNormal {
		if ts.FomoLevel == FomoLevelWeak {
			p = TradeSideNone
		}
	} else if fomoLevel == FomoLevelStrong {
		if ts.FomoLevel != FomoLevelStrong {
			p = TradeSideNone
		}
	}
	return
}

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

// AvgPrice K线平均价格
func (kl *KLine) AvgPrice() float64 {
	d := *kl
	return (d.High + d.Low + d.Close + d.Open) / 4.0
}

// MidPrice K线中间价格
func (kl *KLine) MidPrice() float64 {
	d := *kl
	return (d.High + d.Low) / 2.0
}

// C_O_Price close-open
func (kl *KLine) C_O_Price() float64 {
	d := *kl
	return d.Close - d.Open
}
