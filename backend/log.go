package main

import (
	"github.com/jackc/pgx"
	log "gopkg.in/inconshreveable/log15.v2"
)

type log15Adapter struct {
	logger log.Logger
}

func (a *log15Adapter) Log(level int, msg string, ctx ...interface{}) {
	switch level {
	case pgx.LogLevelTrace, pgx.LogLevelDebug:
		a.logger.Debug(msg, ctx...)
	case pgx.LogLevelInfo:
		a.logger.Info(msg, ctx...)
	case pgx.LogLevelWarn:
		a.logger.Warn(msg, ctx...)
	case pgx.LogLevelError:
		a.logger.Error(msg, ctx...)
	default:
		panic("invalid log level")
	}
}
