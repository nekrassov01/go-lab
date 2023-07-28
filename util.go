package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
)

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
		printError(err)
	}
	os.Stdout.Write(b)
	fmt.Println("")
}

func benchmark(f func()) {
	baseTime := time.Now()

	f()

	fmt.Println("")
	fmt.Println(color.GreenString("CpuCoreNumber: %v", runtime.NumCPU()))
	fmt.Println(color.GreenString("GoroutineNumber: %v", runtime.NumGoroutine()))
	fmt.Println(color.GreenString("ElapsedTime: %v", time.Since(baseTime)))
	fmt.Println("")
}
