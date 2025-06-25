{
        "name": "strategy1",
        "description": "This is a sample strategy",
        "combine": 0, //联合group策略，暂时不生效
        "status": 0, //0:启用，1:禁用
        "dataSource": {
            "exchange": "binance",
            "symbol": "BTC-USDT",
            "market": "future_u", //future_u:u本位合约，future_b:币本位合约 spot:现货
            "limit": 120,         //k线数据limit数量
            "ticker": 600        //ticker数据获取频率，单位：秒
        },
        "groupStrategies": {
            "15":{ //k线周期 分钟
                "name":"group1",
                "description":"This is a sample group",
                "kLimit": 100,
                "disable": 0,
                "type": 0, //0:普通，1:控盘策略
                "passRate": 0.8, //通过率，满足条件的信号数量/总信号数量
                "prePassRate":0.0, //同上，较低概率
                "fomoLevel": 0,    //FOMO水平，-1:低，0:一般，1:高
                "DelayMicroStrate":"", //延迟策略,一般是macd等
                "DelayX":0, //延迟 X * kPeriod 
                "MicroStrategies":{
                    "macd":{ //指标名称
                        "name":"macd", 
                        "isRequire": true, //该指标是否必须，如果true， 必须有buy/sell，否则整个策略不生效 none
                        "isStopPoi": false,
                        "delayX": 0,
                        "isDictatorship": false, //是否独断策略，独断策略不再考虑其他策略的信号，只做单一策略的信号
                        "fomoLevel": 0,
                        "params":{ //指标参数
                            "fastPeriod": 12,
                            "slowPeriod": 26,
                            "signalPeriod": 9,
                            "continueSize": 0,
                            "ContinueFomo": 0
                        }
                    },
                    "kdj":{
                        "name":"kdj",
                        "isRequire": true,
                        "isStopPoi": false,
                        "delayX": 0,
                        "isDictatorship": false,
                        "fomoLevel": 0,
                        "params":{
                            "kPeriod": 9,
                            "dPeriod": 3,
                            "jPeriod": 3,
                            "overBuyVal": 85,
                            "overSellVal": 20
                        }
                    },
                     "volatility":{
                        "name":"volatility",
                        "params":{
                            "effectKLineNum": 15,
                            "vFactor": 1.2,
                            "diffType": 0
                        }
                    },
                }
            }
        }
    },