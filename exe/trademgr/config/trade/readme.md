    "btc-usdt":{ //symbol
        "exchange":"BINANCE",
        "market": "SPOT", 
        "strategy": "normal", //normal, grid,/etc.
        "positionUseRate": 0.5, //仓位利用率
        "leverRate": 10, //杠杆倍数
        "tradeType": "MARKET", //MARKET, LIMIT
        "stopLossRate": 0.05, //止损比率
        "stopWinRate": 0.05, //止盈比率
        "ordersCount": 3, //最大订单数(单次策略多笔订单)
        "qtyIncr": 0.3,  //每笔订单增加的金额
        "priceIncr": 0.03,  //每笔订单增加的价格
        "timeoutCancelPeriodX": 2,  //超时取消订单时间(X * Period)
        "closedOrderNoOpWaitPeriodX": 2,//订单完成后不响应其他操作等待时间(X * Period)
        "openOrderNoOpWaitPeriodX":2, //订单开放后不响应其他操作等待时间(X * Period)
        "orderStatusCheckTicker": 600 //订单状态检查时间间隔(秒)
    }