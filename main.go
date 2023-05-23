package main

import "os"

const (
	name   = "go-lab"
	region = "ap-northeast-1"
)

func main() {
	out, err := getAwsInstanceAsync(region)
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	printJson(out)

	o2, err := getAwsInstanceAsync2(region)
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	printJson(o2)

	o3, err := getAwsInstanceSync(region)
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	printJson(o3)
}
