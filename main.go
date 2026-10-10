package main

import (
	"fmt"
	"os"
)

func main() {
	pids, err := listPIDs()
	if err != nil {
		fmt.Println("cannot read /proc:", err)
		os.Exit(1)
	}
	mem, err := readMemInfo()
	if err != nil {
		fmt.Println("cannot read memory info:", err)
		os.Exit(1)
	}
	users := loadUsers()

	fmt.Printf("Memory: %d MB total, %d MB used, %d MB available\n\n",
		mem.Total/1024, mem.Used()/1024, mem.Available/1024)
	fmt.Printf("%7s %7s %-10s %s %5s %9s %4s %3s %9s  %s\n",
		"PID", "PPID", "USER", "S", "%MEM", "RSS(kB)", "THR", "NI", "TIME", "COMMAND")

	shown, kernel, skipped := 0, 0, 0
	for _, pid := range pids {
		p, err := readProcess(pid)
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
			p.PID, p.PPID, userName(users, p.UID), p.State,
			p.MemPercent(mem.Total), p.RSSKB, p.NumThreads, p.Nice,
			formatCPUTime(p.UTime+p.STime), p.Cmdline)
	}
	fmt.Printf("\n%d processes shown, %d kernel threads hidden, %d skipped\n",
		shown, kernel, skipped)
}
