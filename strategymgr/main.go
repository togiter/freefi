package main

import (
	"flag"
	"fmt"
	"freefi/strategymgr/config"
	"freefi/strategymgr/pkg/logger"
	"freefi/strategymgr/pkg/redis"
	"runtime"
)

var (
	configPath = flag.String("conf", "./config/config.yaml", "configPath")
)

// Init 初始化
func Init() error {
	// 把用户传递的命令行参数解析为对应变量的值
	flag.Parse()
	err := config.InitConf(configPath)
	if err != nil {
		return err
	}
	logCfg := config.GetGlobalCfg().Log
	err = logger.InitLog(logger.LogCfg{
		Level:   logCfg.Level,
		Path:    logCfg.Path,
		Type:    logCfg.Type,
		CutTime: int64(logCfg.CutTime), //hour
	})
	if err != nil {
		return err
	}
	rdsCfg := config.GetGlobalCfg().Redis
	err = redis.Init(redis.RedisCfg{
		IP:          rdsCfg.IP,
		Port:        rdsCfg.Port,
		Password:    rdsCfg.Password,
		DBIndex:     rdsCfg.DBIndex,
		MaxIdle:     rdsCfg.MaxIdle,
		MaxActive:   rdsCfg.MaxActive,
		IdleTimeout: rdsCfg.IdleTimeout,
	})
	// db.Init()
	return err
}
func main() {
	if err := Init(); err != nil {
		panic(err)
	}
	defer func() {
		if err := recover(); err != nil {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			tmpStr := fmt.Sprintf("err: %v, panic==> %s\n", err, string(buf[:n]))
			logger.Errorf(tmpStr)
			//fmt.Println(tmpStr)
		}
	}()
	startStrate()
	// 阻塞主
	<-make(chan struct{})
}

func startStrate() {
	err := StartStrategyMgr()
	if err != nil {
		panic(err)
	}
}
