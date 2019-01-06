package util

import (
	log "pkg/seelog"
)

//seelog是go早期的日志框架，go.uber.org/zap性能更好

var (
	Log         log.LoggerInterface
	RequestLog  log.LoggerInterface
	ResponseLog log.LoggerInterface
)

func InitLogger(LogConfFile string) {
	var err error
	Log, err = log.LoggerFromConfigAsFile(LogConfFile)
	if err != nil {
		panic(err)
	}
}
