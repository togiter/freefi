package ordermgr

import (
	"freefi/trademgr/accmgr"
	"freefi/trademgr/common"
	"freefi/trademgr/pkg/logger"
	"math"
	"time"
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

func closeBySpecifieds(curSide string, nodes map[int64]*GroupStrategyRet, specifieds []Specified) (close bool, tradeSide string) {
	if len(specifieds) == 0 || curSide == common.TradeSideNone || len(specifieds) == 0 {
		//基本前提判断
		return
	}
	//如果node满足，返回，否则判断node.leaves
	nodePass := 0
	nodeLen := len(specifieds)
	for _, sp := range specifieds {
		node := nodes[sp.NodeKPeriod]
		if node == nil {
			//node不存在
			logger.Warnf("暂没有该时段(kPeriod = %v)的节点决策", sp.NodeKPeriod)
			return
		}
		nodeTradeSide := node.TradeSuggest.TradeSide
		if nodeTradeSide != curSide { //满足反向，且不可能为None，上面有判断
			nodePass++
			continue
		}
		//以下是node == none
		if len(sp.Leaves) == 0 {
			//nodePass ！= nodeLen
			continue
		}
		for _, lef := range sp.Leaves {
			leaf := node.MicroStrategyRets[lef]
			if leaf == nil {
				return
			}
			leafTradeSide := leaf.TradeSuggest.TradeSide
			if curSide == leafTradeSide { //leaf同向
				logger.Warnf("暂没有该时段(kPeriod = %v)的节点(%v)决策(%v)不满足要求", sp.NodeKPeriod, lef, leafTradeSide)
				return
			}
		}
		nodePass++
	}
	if nodePass == nodeLen {
		close = true
		tradeSide = common.TradeSideLong
		if curSide == common.TradeSideLong { //反向平仓
			tradeSide = common.TradeSideShort
		}
	}
	return
}

func closeByDelays(curSide string, curSideTimestamp int64, nodes map[int64]*GroupStrategyRet, delays []Delay) (close bool, tradeSide string) {
	if curSide == common.TradeSideNone || curSideTimestamp == 0 || len(delays) == 0 {
		//基本前提
		return
	}
	nodePass := 0
	delayLen := len(delays)
	delayTime := time.Now().Unix()
	for _, d := range delays {
		if curSideTimestamp+int64(float64(d.NodeKPeriod)*60.0*d.TimeX) < delayTime {
			//延时时间未到
			return
		}
		node := nodes[d.NodeKPeriod]
		if node == nil {
			//不存在该Node
			return
		}
		if len(d.Leaves) == 0 {
			//没有延时策略
			return
		}
		for _, lef := range d.Leaves {
			leaf := node.MicroStrategyRets[lef]
			if leaf == nil {
				return
			}
			leafTradeSide := leaf.TradeSuggest.TradeSide
			if leafTradeSide == curSide {
				return
			}
		}
		nodePass++
	}

	if nodePass == delayLen {
		close = true
		tradeSide = common.TradeSideLong
		if curSide == common.TradeSideLong { //反向平仓
			tradeSide = common.TradeSideShort
		}
	}
	return
}
