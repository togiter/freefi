package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Conf 全局配置对象
var Conf *Config

type RedisCfg struct {
	IP          string `json:"ip" yaml:"ip"`
	Port        int    `json:"port" yaml:"port"`
	Password    string `json:"password" yaml:"password"`
	DBIndex     int    `json:"dbIndex" yaml:"dbIndex"`
	MaxIdle     int    `json:"maxIdle" yaml:"maxIdle"`
	MaxActive   int    `json:"maxActive" yaml:"maxActive"`
	IdleTimeout int    `json:"idleTimeout" yaml:"idleTimeout"`
}

// Logs 日记
type LogCfg struct {
	Path    string `yaml:"path"`
	Level   string `yaml:"level"`
	Type    string `yaml:"type"`
	CutTime int    `yaml:"cutTime"`
}

type Config struct {
	Redis RedisCfg `json:"redis" yaml:"redis"`
	Log   LogCfg   `json:"log" yaml:"log"`
}

func GetGlobalCfg() Config {
	if Conf == nil {
		err := InitConf(nil)
		if err != nil {
			panic(err)
		}
	}
	return *Conf
}

// InitConf 初始化配置文件
func InitConf(path *string) error {
	cfgPath := ""
	if path != nil {
		cfgPath = *path
	} else {
		wd, _ := os.Getwd()
		cfgPath = wd + "/config/config.yaml"
	}

	flieContent, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(flieContent, &Conf)
	fmt.Printf("config: %+v\n", Conf)
	return err
}
