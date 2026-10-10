package main

import (
	"fmt"
	"os"
	"time"

	"process-monitor/monitor"
	"process-monitor/proc"
)

func main() {
	users := proc.LoadUsers()

	// %CPU needs two readings: the first one only stores the counters.
	var sampler monitor.Sampler
	if _, err := sampler.Sample(); err != nil {
		fmt.Println("cannot read /proc:", err)
		os.Exit(1)
	}
	time.Sleep(time.Second)
	snap, err := sampler.Sample()
	if err != nil {
		fmt.Println("cannot read /proc:", err)
		os.Exit(1)
	}

	sum := snap.Summary
	fmt.Printf("Tasks: %d total, %d running, %d sleeping, %d stopped, %d zombie\n",
		sum.Total, sum.Running, sum.Sleeping, sum.Stopped, sum.Zombie)
	fmt.Printf("CPU: %.1f%%   Load average: %.2f %.2f %.2f\n",
		sum.CPUPercent, sum.Load.One, sum.Load.Five, sum.Load.Fifteen)
	fmt.Printf("Memory: %d MB total, %d MB used, %d MB available\n\n",
		sum.Mem.Total/1024, sum.Mem.Used()/1024, sum.Mem.Available/1024)

	fmt.Printf("%7s %7s %-10s %s %5s %5s %9s %4s %3s %9s  %s\n",
		"PID", "PPID", "USER", "S", "%CPU", "%MEM", "RSS(kB)", "THR", "NI", "TIME", "COMMAND")
	for _, r := range snap.Rows {
		fmt.Printf("%7d %7d %-10.10s %s %5.1f %5.1f %9d %4d %3d %9s  %.60s\n",
			r.PID, r.PPID, users.Name(r.UID), r.State, r.CPUPercent, r.MemPercent,
			r.RSSKB, r.NumThreads, r.Nice, formatCPUTime(r.UTime+r.STime), r.Cmdline)
	}
	fmt.Printf("\n%d processes shown, %d skipped\n", len(snap.Rows), snap.Skipped)
}

// formatCPUTime turns ticks into "minutes:seconds.hundredths", like top.
func formatCPUTime(ticks uint64) string {
	hundredths := ticks * 100 / proc.ClockTicks
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths%100)
}
