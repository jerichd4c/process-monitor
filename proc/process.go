// Package proc reads the state of processes and of the system directly from
// the files the Linux kernel exposes in /proc. It only reads raw data;
// everything derived from it (percentages, totals) is computed in our code.
package proc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ClockTicks is how many ticks per second the kernel uses for the time
// fields in /proc (USER_HZ). It is 100 on mainstream Linux; Go cannot call
// sysconf(_SC_CLK_TCK) without cgo, so we keep it as a constant.
const ClockTicks = 100

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

// ListPIDs returns the PID of every process, sorted: each entry of /proc
// whose name is a number is a process.
func ListPIDs() ([]int, error) {
	return numericEntries("/proc")
}

// ListThreadIDs returns the IDs of the threads of a process, sorted: the
// entries of /proc/[pid]/task. The main thread has the same ID as the
// process.
func ListThreadIDs(pid int) ([]int, error) {
	return numericEntries(fmt.Sprintf("/proc/%d/task", pid))
}

// numericEntries returns, sorted, the entries of dir whose name is a number.
func numericEntries(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, classify(err)
	}
	var ids []int
	for _, e := range entries {
		id, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a process: "meminfo", "self", "sys"...
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

// Read gathers everything we show about one process.
func Read(pid int) (Process, error) {
	data, err := readFile(pid, "stat")
	if err != nil {
		return Process{}, err
	}
	p, err := parseStat(pid, data)
	if err != nil {
		return Process{}, err
	}

	data, err = readFile(pid, "status")
	if err != nil {
		return Process{}, err
	}
	p.UID, p.RSSKB, err = parseStatus(data)
	if err != nil {
		return Process{}, fmt.Errorf("/proc/%d/status: %w", pid, err)
	}

	// Zombies have an empty cmdline; like top and ps, we fall back to the
	// short name between brackets.
	data, err = readFile(pid, "cmdline")
	p.Cmdline = parseCmdline(data)
	if err != nil || p.Cmdline == "" {
		p.Cmdline = "[" + p.Comm + "]"
	}
	return p, nil
}

// IsKernelThread reports whether the process is a kernel thread.
func (p Process) IsKernelThread() bool {
	return p.Flags&pfKthread != 0
}

// MemPercent returns the share of the system RAM the process uses.
func (p Process) MemPercent(memTotalKB uint64) float64 {
	return float64(p.RSSKB) / float64(memTotalKB) * 100
}

// readFile returns the content of /proc/[pid]/[name] as a string.
func readFile(pid int, name string) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/%s", pid, name))
	return string(data), classify(err)
}

// parseStat parses the line of /proc/[pid]/stat.
// Field numbers follow "man 5 proc".
func parseStat(pid int, line string) (Process, error) {
	// The name (field 2) is between parentheses and may contain spaces or
	// ")" itself, so we look for the FIRST "(" and the LAST ")".
	openParen := strings.IndexByte(line, '(')
	closeParen := strings.LastIndexByte(line, ')')
	if openParen < 0 || closeParen < openParen {
		return Process{}, fmt.Errorf("/proc/%d/stat: malformed line", pid)
	}

	// Everything after ")" is safe to split by spaces.
	// rest[0] is field 3, so field n is rest[n-3].
	rest := strings.Fields(line[closeParen+1:])
	if len(rest) < 20 {
		return Process{}, fmt.Errorf("/proc/%d/stat: too few fields (%d)", pid, len(rest)+2)
	}
	field := func(n int) string { return rest[n-3] }

	// num parses field n as a number and remembers the first error,
	// so we do not need an "if err != nil" after every field.
	var parseErr error
	num := func(n int) int64 {
		v, err := strconv.ParseInt(field(n), 10, 64)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("/proc/%d/stat: field %d: %w", pid, n, err)
		}
		return v
	}

	p := Process{
		PID:        pid,
		Comm:       printable(line[openParen+1 : closeParen]),
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

// parseStatus reads two lines of /proc/[pid]/status:
//   - "Uid:" has four IDs: real, effective, saved and filesystem; like top,
//     we keep the effective one.
//   - "VmRSS:" is the resident memory in kB. Zombies and kernel threads
//     have no memory, so the line is missing and rssKB stays 0.
func parseStatus(data string) (uid int, rssKB uint64, err error) {
	uid = -1
	for _, line := range strings.Split(data, "\n") {
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
			return 0, 0, fmt.Errorf("%s %w", fields[0], err)
		}
	}
	if uid < 0 {
		return 0, 0, fmt.Errorf("no Uid line")
	}
	return uid, rssKB, nil
}

// parseCmdline turns the content of /proc/[pid]/cmdline into a readable
// command. In the file the arguments are separated by NUL bytes ("\x00").
func parseCmdline(data string) string {
	cmd := strings.TrimRight(data, "\x00")
	return printable(strings.ReplaceAll(cmd, "\x00", " "))
}

// printable replaces with "?" every character that cannot be shown as is:
// control characters (a new line, a tab, an escape sequence...) and bytes
// that are not valid text. A process chooses its own name and arguments, so
// they must never be trusted to be harmless when written to a terminal.
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if r == utf8.RuneError || !unicode.IsPrint(r) {
			return '?'
		}
		return r
	}, s)
}

// ErrNoAccess means the kernel did not let us read the data of a process:
// the process exists, but /proc hides its files from this user (as happens
// when /proc is mounted with the hidepid option). It is a different case
// from a process that has ended, and it gets a different message.
var ErrNoAccess = errors.New("no permission to read its data in /proc")

// classify turns a "permission denied" error into ErrNoAccess and returns
// every other error unchanged.
func classify(err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return ErrNoAccess
	}
	return err
}
