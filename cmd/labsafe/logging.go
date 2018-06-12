package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

func printLog(level string, attr color.Attribute, format string, a ...interface{}) {
	ts := time.Now().Format("15:04:05")
	prefix := color.New(attr).Sprintf("%s [%s]", ts, level)
	fmt.Fprintln(color.Output, prefix, fmt.Sprintf(format, a...))
}

func Info(format string, a ...interface{}) {
	printLog("INFO", color.FgHiCyan, format, a...)
}

func Debug(format string, a ...interface{}) {
	printLog("DEBU", color.FgHiYellow, format, a...)
}

func Error(format string, a ...interface{}) {
	printLog("ERRO", color.FgHiRed, format, a...)
}

func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}
