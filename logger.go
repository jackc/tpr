package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

var logger *SimpleLogger

func init() {
	logger = &SimpleLogger{dest: os.Stderr}
}

type SimpleLogger struct {
	dest io.Writer
}

func (l *SimpleLogger) Error(msg string) {
	l.Log("Error", msg)
}

func (l *SimpleLogger) Warning(msg string) {
	l.Log("Warning", msg)
}

func (l *SimpleLogger) Info(msg string) {
	l.Log("Info", msg)
}

func (l *SimpleLogger) Debug(msg string) {
	l.Log("Debug", msg)
}

func (l *SimpleLogger) Log(level, msg string) {
	t := time.Now()
	_, err := fmt.Fprintf(l.dest, "%v - %s - %s\n", t, level, msg)
	if err != nil {
		panic(fmt.Sprintf("Writing to log failed: %v\n", err))
	}
}
