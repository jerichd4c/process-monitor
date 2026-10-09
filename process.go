package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// clockTicks is how many ticks per second the kernel uses for the time
// fields in /proc (USER_HZ). It is 100 on mainstream Linux; Go cannot call
// sysconf(_SC_CLK_TCK) without cgo, so we keep it as a constant.
const clockTicks = 100

// Process holds the data of one process read from /proc/[pid]/stat.
type Process struct {
	PID        int
	Comm       string // executable name (the kernel truncates it to 15 chars)
	State      string // R, S, D, Z, T, t, I...
	PPID       int
	UTime      uint64 // CPU time spent in user mode, in ticks
	STime      uint64 // CPU time spent in kernel mode, in ticks
	Priority   int
	Nice       int
	NumThreads int
	StartTime  uint64 // when the process started, in ticks since boot
	RSS        uint64 // resident memory, in pages
}

// readStat reads /proc/[pid]/stat and parses the fields we need.
// Field numbers follow "man 5 proc".
func readStat(pid int) (Process, error) {
	path := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return Process{}, err
	}
	line := string(data)

	// The name (field 2) is between parentheses and may contain spaces or
	// ")" itself, so we look for the FIRST "(" and the LAST ")".
	openParen := strings.IndexByte(line, '(')
	closeParen := strings.LastIndexByte(line, ')')
	if openParen < 0 || closeParen < openParen {
		return Process{}, fmt.Errorf("%s: malformed line", path)
	}

	// Everything after ")" is safe to split by spaces.
	// rest[0] is field 3, so field n is rest[n-3].
	rest := strings.Fields(line[closeParen+1:])
	if len(rest) < 22 {
		return Process{}, fmt.Errorf("%s: too few fields (%d)", path, len(rest)+2)
	}
	field := func(n int) string { return rest[n-3] }

	// num parses field n as a number and remembers the first error,
	// so we do not need an "if err != nil" after every field.
	var parseErr error
	num := func(n int) int64 {
		v, err := strconv.ParseInt(field(n), 10, 64)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("%s: field %d: %w", path, n, err)
		}
		return v
	}

	p := Process{
		PID:        pid,
		Comm:       line[openParen+1 : closeParen],
		State:      field(3),
		PPID:       int(num(4)),
		UTime:      uint64(num(14)),
		STime:      uint64(num(15)),
		Priority:   int(num(18)),
		Nice:       int(num(19)),
		NumThreads: int(num(20)),
		StartTime:  uint64(num(22)),
		RSS:        uint64(num(24)),
	}
	if parseErr != nil {
		return Process{}, parseErr
	}
	return p, nil
}

// formatCPUTime turns ticks into "minutes:seconds.hundredths", like top.
func formatCPUTime(ticks uint64) string {
	hundredths := ticks * 100 / clockTicks
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths%100)
}
