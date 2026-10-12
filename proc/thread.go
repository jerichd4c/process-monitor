package proc

import (
	"fmt"
	"os"
)

// Thread holds the data of one thread of a process. For the kernel a thread
// is a task like any other: it has its own ID, state, CPU counters and
// priority, but shares the memory of its process.
type Thread struct {
	TID      int    // thread ID; the main thread's TID is the PID
	Name     string // threads can have their own name
	State    string // R, S, D, T...
	UTime    uint64 // CPU time of this thread in user mode, in ticks
	STime    uint64 // CPU time of this thread in kernel mode, in ticks
	Priority int
	Nice     int
}

// ReadThread reads /proc/[pid]/task/[tid]/stat. The file has the same
// format as /proc/[pid]/stat, but its counters are those of one thread
// instead of the sum of all the threads of the process.
func ReadThread(pid, tid int) (Thread, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/stat", pid, tid))
	if err != nil {
		return Thread{}, err
	}
	p, err := parseStat(tid, string(data))
	if err != nil {
		return Thread{}, err
	}
	return Thread{
		TID:      tid,
		Name:     p.Comm,
		State:    p.State,
		UTime:    p.UTime,
		STime:    p.STime,
		Priority: p.Priority,
		Nice:     p.Nice,
	}, nil
}
