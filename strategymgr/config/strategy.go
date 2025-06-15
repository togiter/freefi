package config

import (
	"fmt"
	"freefi/strategymgr/strategy"
	"os"

	"gopkg.in/yaml.v3"
)

// Conf 全局配置对象
var StrateCfg *StrategyCfg

type StrategyCfg struct {
	Strategies []*strategy.StrategyParams `yaml:"strategies"`
}

func GetStrategyCfg(path *string) (*StrategyCfg, error) {
	if StrateCfg != nil {
		return StrateCfg, nil
	}
	cfgPath := ""
	if path != nil {
		cfgPath = *path
	} else {
		wd, _ := os.Getwd()
		cfgPath = wd + "/config/strategy.yaml"
	}

	flieContent, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(flieContent, &StrateCfg)
	fmt.Printf("strategy config: %+v\n", StrateCfg)
	return StrateCfg, err
}
