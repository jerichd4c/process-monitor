package main

import (
	"fmt"
	"os"

	"process-monitor/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "process-monitor:", err)
		os.Exit(1)
	}
}
