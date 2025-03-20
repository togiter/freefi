package micro_strategy

import common "freefi/strategymgr/common"

type MicroStrategyParams struct {
	Name string `json:"name"`
	//是否关闭微调模式
	Legacy bool                   `json:"legacy"`
	Params map[string]interface{} `json:"params"`
	Ext    *ExtParams             `json:"ext"`
}
type ExtParams struct {
	IsRequire      bool `json:"isRequire"`      //  0 非必需，1 必须
	IsStopPoi      bool `json:"isStopPoi"`      //  是否指定为止盈损指标 0 否，1 是，满足即可止盈止损
	DelayX         int  `json:"delayX"`         //  延时确认时间倍数(kPeriods * DelayX)
	IsDictatorship bool `json:"isDictatorship"` //  是否独裁
	FomoLevel      int  `json:"fomoLevel"`
}

type MicroStrategyRet struct {
	Params       *MicroStrategyParams   `json:"params"`
	TradeSuggest common.TradeSuggest    `json:"tradeSuggest"`
	Opts         map[string]interface{} `json:"opts"`
}

func (msRet *MicroStrategyRet) IsRequireChecked() bool {
	if msRet.Params.Ext == nil || !msRet.Params.Ext.IsRequire {
		return true
	}
	return msRet.TradeSuggest.StrictFomoLevel(msRet.Params.Ext.FomoLevel) != common.TradeSideNone
}

// DiactorShipSuggest 是否是独裁，策略建议
func (msRet *MicroStrategyRet) DiactorShipSuggest() (flag bool, sg common.TradeSuggest) {
	flag = msRet.Params.Ext != nil && msRet.Params.Ext.IsDictatorship
	sg = msRet.TradeSuggest
	return
}

func (msRet *MicroStrategyRet) MakeFinalTrade() {

}
