package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

var logger *Logger

func init() {
	logger = &Logger{dest: os.Stderr}
}

type Logger struct {
	dest io.Writer
}

func (l *Logger) Log(pkg, level, msg string) {
	t := time.Now()
	_, err := fmt.Fprintf(l.dest, "%v %s %s - %s\n", t, pkg, level, msg)
	if err != nil {
		panic(fmt.Sprintf("Writing to log failed: %v\n", err))
	}
}

func (l *Logger) Error(pkg, msg string) {
	l.Log(pkg, "Error", msg)
}

func (l *Logger) Warning(pkg, msg string) {
	l.Log(pkg, "Warning", msg)
}

func (l *Logger) Info(pkg, msg string) {
	l.Log(pkg, "Info", msg)
}

func (l *Logger) Debug(pkg, msg string) {
	l.Log(pkg, "Debug", msg)
}

type PackageLogger struct {
	logger *Logger
	pkg    string
}

func (l *PackageLogger) Log(level, msg string) {
	l.logger.Log(l.pkg, level, msg)
}

func (l *PackageLogger) Error(msg string) {
	l.Log("Error", msg)
}

func (l *PackageLogger) Warning(msg string) {
	l.Log("Warning", msg)
}

func (l *PackageLogger) Info(msg string) {
	l.Log("Info", msg)
}

func (l *PackageLogger) Debug(msg string) {
	l.Log("Debug", msg)
}
