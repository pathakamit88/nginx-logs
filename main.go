package main

import (
	"context"
	"fmt"
	"nginx-log/cmd"
	"nginx-log/pkg/config"
	"nginx-log/pkg/ossearch"
	"os"
	"time"
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

	start := time.Now()
	stats := ossearch.GetResponse(context.Background(), cfg, intervalStr)
	elapsed := time.Since(start)

	fmt.Printf("Total time taken: %v\n", elapsed)

	printOnConsole(stats)
}

func printOnConsole(stats []ossearch.ResponseStat) {
	fmt.Printf("%-10s %-10s %-10s %-10s %-s\n", "min", "avg", "max", "Count", "Path")
	for _, stat := range stats {
		fmt.Printf("%-10.2f %-10.2f %-10.2f %-10d %-s\n", stat.Min, stat.Avg, stat.Max, stat.Count, stat.Request)
	}
}
