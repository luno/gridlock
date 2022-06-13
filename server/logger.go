package main

import (
	"encoding/json"
	jlog "github.com/luno/jettison/log"
	"log"
	"os"
)

type JSONLogger struct {
	*log.Logger
}

func (l *JSONLogger) Log(log jlog.Log) string {
	res, err := json.Marshal(log)
	if err != nil {
		l.Logger.Printf("jlogger: failed to marshal log: %v", err)
		l.Logger.Print(log.Message) // best-effort
		return log.Message
	}
	l.Logger.Print(string(res))
	return string(res)
}

func InitLogging() {
	jlog.SetLogger(&JSONLogger{Logger: log.New(os.Stdout, "", 0)})
}
