package main

import (
	"context"
	"fmt"
	"nginx-log/cmd"
	"nginx-log/pkg/config"
	"nginx-log/pkg/ossearch"
	"os"
)

func main() {
	cfgPath, version := cmd.ParseFlags()
	if version {
		cmd.PrintVersion()
		return
	}

	// Initalise config
	cfg, err := config.ReadConfig(cfgPath)
	if err != nil {
		fmt.Println("cannot initialize config", err)
	}

	intervalStr := "5m"
	if len(os.Args) > 1 {
		intervalStr = os.Args[1]
	}

	ossearch.GetResponse(context.Background(), cfg, intervalStr)
}
