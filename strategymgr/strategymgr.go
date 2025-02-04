package main

import (
	"encoding/json"
	"fmt"
	"freefi/strategymgr/pkg/logger"
	"os"

	"github.com/fsnotify/fsnotify"
)

const (
	CONFIG_FILE_PATH = "./config/strategies/strategy.json"
)

type IStrategyMgr interface {
	CfgPath() string
	AddStrategy(strategy IStrategy) error
	RemoveStrategy(strategyName string) error
	Init() error
	Run() error
	Update()
}

type StrategyMgr struct {
	strategies map[string]IStrategy
	cfgPath    string
}

func NewStrategyMgr() IStrategyMgr {
	return &StrategyMgr{
		strategies: make(map[string]IStrategy),
		cfgPath:    CONFIG_FILE_PATH,
	}
}

func (s *StrategyMgr) Init() error {
	sParams, err := getCfg(nil)
	if err != nil {
		return err
	}
	for _, sParam := range sParams {
		if err = checkParams(sParam); err != nil {
			return fmt.Errorf("%s Invalid params %v", sParam.Name, err)
		}
		if sParam.Status == 1 {
			continue
		}
		strategy := NewStrategy(sParam)
		s.AddStrategy(strategy)
	}
	go s.watchFile()
	return nil
}

func (s *StrategyMgr) RemoveStrategy(strategyName string) error {
	if s.strategies[strategyName] == nil {
		return fmt.Errorf("strategy %s not exist", strategyName)
	}
	s.strategies[strategyName].Stop()
	delete(s.strategies, strategyName)
	logger.Infof("remove %s \n", strategyName)
	return nil
}

func (s *StrategyMgr) Update() {

	sParams, err := getCfg(&s.cfgPath)
	if err != nil {
		logger.Infof("Update config failed: %v\n", err)
		return
	}
	for _, sParam := range sParams {
		if sParam.Status == 1 {
			if s.strategies[sParam.Name] != nil {
				s.RemoveStrategy(sParam.Name)
			}
			continue
		}
		if s.strategies[sParam.Name] == nil {
			strategy := NewStrategy(sParam)
			s.AddStrategy(strategy)
			go func(s IStrategy) {
				err := s.Work()
				if err != nil {
					logger.Warnf("Start strategy %s failed: %v\n", s.Name(), err)
				}
			}(strategy)
			continue
		}
		logger.Infof("Update %s params\n", sParam.Name)
		s.strategies[sParam.Name].UpdateParams(sParam)
	}
}

func (s *StrategyMgr) CfgPath() string {
	return s.cfgPath
}

func (s *StrategyMgr) AddStrategy(iStrategy IStrategy) error {
	s.strategies[iStrategy.Name()] = iStrategy
	logger.Infof("Add Strategy %s \n", iStrategy.Name())
	return nil
}

func (s *StrategyMgr) Run() error {
	for _, strategy := range s.strategies {
		go func(s IStrategy) {
			err := s.Work()
			if err != nil {
				logger.Warnf("Start %s failed: %v\n", s.Name(), err)
			}
		}(strategy)
	}
	return nil
}

func getCfg(path *string) ([]StrategyParams, error) {
	cfgPath := "./config/strategies/strategy.json"
	if path != nil {
		cfgPath = *path
	}
	flie, err := os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer flie.Close()
	decoder := json.NewDecoder(flie)
	cfg := []StrategyParams{}
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *StrategyMgr) watchFile() {
	// 创建一个新的文件系统监视器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()
	logger.Infof("watchFile: %v", s.CfgPath())
	// 添加需要监视的文件或目录
	err = watcher.Add(s.CfgPath())
	if err != nil {
		panic(err)
	}
	// 启动一个 goroutine 处理事件
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				logger.Infof("watchFile event: %v", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					logger.Infof("watchFile modified file: %v", event.Name)
					go s.Update()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Warnf("watchFile error: %v", err)
			}
		}
	}()

	// 阻塞主 goroutine
	<-make(chan struct{})
}
