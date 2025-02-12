package ordermgr

import (
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"math"
)

func getPrice(accmgr.BaseOrderParams) (float64, error) {
	//todo: get price from exchange api
	return 0, nil
}

// calAmountsByTotal 计算每次订单的数量,
func calAmountsByTotal(total float64, aIncr float64, maxCount int, stratType string) []float64 {

	amounts := []float64{}
	if maxCount <= 1 {
		amounts = append(amounts, total)
		return amounts
	}

	amountIncr := aIncr
	if stratType == common.StrategyGRID { //1
		//网格策略, 使用原始绝对增量，如果为0/1，则平均每个节点的数量
		// x + x(1+amountIncr) + x(1+amountIncr)= x + sum(x*amountIncr*i)
		//x(1 + n*(1+amountIncr))
		base := 1.0
		base += (float64(maxCount-1) * (1.0 + amountIncr))

		varAmount := total / base
		for i := 1; i < maxCount; i++ { //第0个已经在外面赋值，从1开始
			//fmt.Printf("\n下单数量(%d/%d):(%.2f/%.2f)",i-1,maxCount,varAmount,total)
			varAmount = varAmount * (1 + amountIncr)
			amounts = append(amounts, varAmount)

		}
	} else if stratType == common.StrategyGAMBLER { //2
		//马丁(赌徒)策略, 每隔一个价格低洼翻倍投入
		//x + (2x) + 4x + 8x + 16x... = total; x(1+...+2^(n-1)) = total, n>= 2;
		//初始数量
		//varAmount := total / sum(1.0 + math.Pow(2,maxCount))
		base := 1.0
		for j := 1; j < maxCount; j++ { //第0个已经在外面赋值，从1开始
			base += math.Pow(2, float64(j))
		}
		varAmount := total / base

		amounts = append(amounts, varAmount)
		for i := 1; i < maxCount; i++ { //第0个已经在外面赋值，从1开始
			varAmount *= 2
			amounts = append(amounts, varAmount)
		}
	} else {
		//默认下单策略
		//相对增量,设 a = amountIncr
		//amount + amount*(1 + a) + amount*(1 + a)*(1 + a).... = total
		//amount(1+(1+a)^(n-1)) = total ,n >= 2
		base := 1.0
		for j := 1; j < maxCount; j++ { //第0个已经在外面赋值，从1开始
			base += math.Pow((1.0 + amountIncr), float64(j))
		}
		//初始数量
		varAmount := total / base
		remain := total - varAmount
		amounts = append(amounts, varAmount)
		for i := 1; i < maxCount; i++ { //第0个已经在外面赋值，从1开始
			if maxCount-1 == i {
				amounts = append(amounts, remain)
				return amounts
			}
			varAmount *= (1.0 + amountIncr)
			remain -= varAmount
			amounts = append(amounts, varAmount)
		}
	}
	return amounts
}
