package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
)

func main() {
	// Without arguments we inspect ourselves; otherwise the PID given.
	pid := os.Getpid()
	if len(os.Args) > 1 {
		n, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Println("usage: process-monitor [pid]")
			os.Exit(1)
		}
		pid = n
	}

	p, err := readStat(pid)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			fmt.Printf("process %d does not exist\n", pid)
		case errors.Is(err, fs.ErrPermission):
			fmt.Printf("no permission to read process %d\n", pid)
		default:
			fmt.Println("error:", err)
		}
		os.Exit(1)
	}

	pageKB := uint64(os.Getpagesize()) / 1024
	fmt.Printf("PID:       %d\n", p.PID)
	fmt.Printf("Command:   %s\n", p.Comm)
	fmt.Printf("State:     %s\n", p.State)
	fmt.Printf("Parent:    %d\n", p.PPID)
	fmt.Printf("Priority:  %d (nice %d)\n", p.Priority, p.Nice)
	fmt.Printf("Threads:   %d\n", p.NumThreads)
	fmt.Printf("CPU time:  %s (user %d + kernel %d ticks)\n",
		formatCPUTime(p.UTime+p.STime), p.UTime, p.STime)
	fmt.Printf("Memory:    %d kB (%d pages)\n", p.RSS*pageKB, p.RSS)
}
