package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// clockTicks is how many ticks per second the kernel uses for the time
// fields in /proc (USER_HZ). It is 100 on mainstream Linux; Go cannot call
// sysconf(_SC_CLK_TCK) without cgo, so we keep it as a constant.
const clockTicks = 100

// pfKthread is the PF_KTHREAD bit of the "flags" field (field 9 of stat).
// The kernel sets it on its own threads (kworker, ksoftirqd...).
const pfKthread = 0x00200000

// Process holds the data we show about one process.
type Process struct {
	// From /proc/[pid]/stat
	PID        int
	Comm       string // executable name (the kernel truncates it to 15 chars)
	State      string // R, S, D, Z, T, t, I...
	PPID       int
	Flags      uint64 // kernel flags, see pfKthread
	UTime      uint64 // CPU time spent in user mode, in ticks
	STime      uint64 // CPU time spent in kernel mode, in ticks
	Priority   int
	Nice       int
	NumThreads int
	StartTime  uint64 // when the process started, in ticks since boot

	// From /proc/[pid]/status and /proc/[pid]/cmdline
	UID     int    // effective user ID
	RSSKB   uint64 // resident memory in kB
	Cmdline string // full command with its arguments
}

// listPIDs returns the PID of every process, sorted: each entry of /proc
// whose name is a number is a process.
func listPIDs() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a process: "meminfo", "self", "sys"...
		}
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	return pids, nil
}

// readProcess gathers everything we show about one process.
func readProcess(pid int) (Process, error) {
	p, err := readStat(pid)
	if err != nil {
		return Process{}, err
	}

	p.UID, p.RSSKB, err = readStatus(pid)
	if err != nil {
		return Process{}, err
	}

	// Zombies have an empty cmdline; like top and ps, we fall back to the
	// short name between brackets.
	cmd, err := readCmdline(pid)
	if err != nil || cmd == "" {
		cmd = "[" + p.Comm + "]"
	}
	p.Cmdline = cmd
	return p, nil
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
		Flags:      uint64(num(9)),
		UTime:      uint64(num(14)),
		STime:      uint64(num(15)),
		Priority:   int(num(18)),
		Nice:       int(num(19)),
		NumThreads: int(num(20)),
		StartTime:  uint64(num(22)),
	}
	if parseErr != nil {
		return Process{}, parseErr
	}
	return p, nil
}

// readStatus reads two lines of /proc/[pid]/status:
//   - "Uid:" has four IDs: real, effective, saved and filesystem; like top,
//     we keep the effective one.
//   - "VmRSS:" is the resident memory in kB. Zombies and kernel threads
//     have no memory, so the line is missing and rssKB stays 0.
func readStatus(pid int) (uid int, rssKB uint64, err error) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	uid = -1
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line) // e.g. ["VmRSS:", "7876", "kB"]
		if len(fields) < 3 {
			continue
		}
		switch fields[0] {
		case "Uid:":
			uid, err = strconv.Atoi(fields[2])
		case "VmRSS:":
			rssKB, err = strconv.ParseUint(fields[1], 10, 64)
		}
		if err != nil {
			return 0, 0, fmt.Errorf("%s: %s %w", path, fields[0], err)
		}
	}
	if uid < 0 {
		return 0, 0, fmt.Errorf("%s: no Uid line", path)
	}
	return uid, rssKB, nil
}

// readCmdline returns the full command line. In the file the arguments are
// separated by NUL bytes ("\x00"), so we turn them into spaces.
func readCmdline(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	cmd := strings.TrimRight(string(data), "\x00")
	return strings.ReplaceAll(cmd, "\x00", " "), nil
}

// IsKernelThread reports whether the process is a kernel thread.
func (p Process) IsKernelThread() bool {
	return p.Flags&pfKthread != 0
}

// MemPercent returns the share of the system RAM the process uses.
func (p Process) MemPercent(memTotalKB uint64) float64 {
	return float64(p.RSSKB) / float64(memTotalKB) * 100
}

// formatCPUTime turns ticks into "minutes:seconds.hundredths", like top.
func formatCPUTime(ticks uint64) string {
	hundredths := ticks * 100 / clockTicks
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths%100)
}
