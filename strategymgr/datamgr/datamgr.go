package datamgr

import (
	"context"
	"fmt"
	"strings"

	"freefi/strategymgr/common"

	"github.com/adshao/go-binance/v2"
)

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

// DataMgr is the main struct for managing data
type DataMgr struct {
}

type IDataMgr interface {
	GetKLines(kLineParams KLineParams) ([]common.KLine, error)
}

// NewDataMgr creates a new DataMgr object
func NewDataMgr() IDataMgr {
	return &DataMgr{}
}

func (d *DataMgr) GetKLines(kLineParams KLineParams) ([]common.KLine, error) {
	switch kLineParams.Exchange {
	case Binance:
		return getBinanceKLines(kLineParams)
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", kLineParams.Exchange)
	}
}

func getBinanceKLines(kLineParams KLineParams) ([]common.KLine, error) {
	kLineParams.Symbol = strings.Trim(strings.Replace(kLineParams.Symbol, "-", "", -1)," ")
	if kLineParams.Market == MarketFutureU {
		fCli := binance.NewFuturesClient("", "")
		klines, err := fCli.NewKlinesService().
			Symbol(kLineParams.Symbol).
			Interval(ToBNPeroid(kLineParams.Period)).
			Limit(kLineParams.Limit).
			Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("%v FUTURE-U failed to get klines: %s",kLineParams, err)
		}
		return ToKLines(klines), nil
	}else if kLineParams.Market == MarketFutureB {
		fCliB := binance.NewFuturesClient("", "") //todo:: 币本位合约获取K线失败:code=-1121, msg=Invalid symbol 
		klines, err := fCliB.NewKlinesService().
			Symbol(kLineParams.Symbol).
			Interval(ToBNPeroid(kLineParams.Period)).
			Limit(kLineParams.Limit).
			Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("%v FUTURE-B failed to get klines: %s",kLineParams, err)
		}
		return ToKLines(klines), nil
	}else if kLineParams.Market == MarketSpot {
		cli := binance.NewClient("", "")
		klines, err := cli.NewKlinesService().
			Symbol(kLineParams.Symbol).
			Interval(ToBNPeroid(kLineParams.Period)).
			Limit(kLineParams.Limit).
			Do(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get klines: %s", err)
		}
		return ToKLines(klines), nil
	}
	return nil, fmt.Errorf("unsupported market: %s", kLineParams.Market)	
}
