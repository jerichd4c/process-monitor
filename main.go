package main

import (
	"fmt"
	"os"

	"process-monitor/proc"
)

func main() {
	pids, err := proc.ListPIDs()
	if err != nil {
		fmt.Println("cannot read /proc:", err)
		os.Exit(1)
	}
	mem, err := proc.ReadMemInfo()
	if err != nil {
		fmt.Println("cannot read memory info:", err)
		os.Exit(1)
	}
	users := proc.LoadUsers()

	fmt.Printf("Memory: %d MB total, %d MB used, %d MB available\n\n",
		mem.Total/1024, mem.Used()/1024, mem.Available/1024)
	fmt.Printf("%7s %7s %-10s %s %5s %9s %4s %3s %9s  %s\n",
		"PID", "PPID", "USER", "S", "%MEM", "RSS(kB)", "THR", "NI", "TIME", "COMMAND")

	shown, kernel, skipped := 0, 0, 0
	for _, pid := range pids {
		p, err := proc.Read(pid)
		if err != nil {
			// The process ended between listing /proc and reading it, or we
			// are not allowed to read it: skip it, never crash.
			skipped++
			continue
		}
		if p.IsKernelThread() {
			kernel++
			continue
		}
		shown++
		fmt.Printf("%7d %7d %-10.10s %s %5.1f %9d %4d %3d %9s  %.60s\n",
			p.PID, p.PPID, users.Name(p.UID), p.State,
			p.MemPercent(mem.Total), p.RSSKB, p.NumThreads, p.Nice,
			formatCPUTime(p.UTime+p.STime), p.Cmdline)
	}
	fmt.Printf("\n%d processes shown, %d kernel threads hidden, %d skipped\n",
		shown, kernel, skipped)
}

// formatCPUTime turns ticks into "minutes:seconds.hundredths", like top.
func formatCPUTime(ticks uint64) string {
	hundredths := ticks * 100 / proc.ClockTicks
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths%100)
}
