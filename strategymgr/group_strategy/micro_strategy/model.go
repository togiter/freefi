package micro_strategy

import common "freefi/strategymgr/common"

type MicroStrategyParams struct {
	Name string `json:"name" yaml:"name"`
	//是否关闭微调模式
	Legacy         bool                   `json:"legacy" yaml:"legacy"`
	Required       bool                   `json:"required" yaml:"required"`
	FomoLevel      int                    `json:"fomoLevel" yaml:"fomoLevel"`
	IsDictatorship bool                   `json:"isDictatorship" yaml:"isDictatorship"`
	Params         map[string]interface{} `json:"params" yaml:"params"`
}

type MicroStrategyRet struct {
	Params       *MicroStrategyParams   `json:"params"`
	TradeSuggest common.TradeSuggest    `json:"tradeSuggest"`
	Opts         map[string]interface{} `json:"opts"`
}

func (msRet *MicroStrategyRet) IsRequirePassed() bool {
	if !msRet.Params.Required {
		return true
	}
	return msRet.TradeSuggest.StrictFomoLevel(msRet.Params.FomoLevel) != common.TradeSideNone
}

// DiactorShipSuggest 是否是独裁，策略建议
func (msRet *MicroStrategyRet) DiactorShipSuggest() (flag bool, sg common.TradeSuggest) {
	flag = msRet.Params.IsDictatorship
	sg = msRet.TradeSuggest
	return
}

func (msRet *MicroStrategyRet) MakeFinalTrade() {

}
