package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
)

func printInfo(baseTime time.Time) {
	pc, _, _, _ := runtime.Caller(1)

	fmt.Println("")
	fmt.Println(color.GreenString("FunctionName: %v", runtime.FuncForPC(pc).Name()))
	fmt.Println(color.GreenString("CpuCoreNumber: %v", runtime.NumCPU()))
	fmt.Println(color.GreenString("GoroutineNumber: %v", runtime.NumGoroutine()))
	fmt.Println(color.GreenString("ElapsedTime: %v", time.Since(baseTime)))
	fmt.Println("")
}

func printError(err error) {
	if name == "" {
		fmt.Println(color.RedString("%v\n", err))
		return
	}
	fmt.Println(color.RedString("[%v] %v\n", name, err))
}

func printJson(obj any) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		printError(fmt.Errorf("`json.MarshalIndent()` failed: %w", err))
	}
	os.Stdout.Write(b)
	fmt.Println("")
}
