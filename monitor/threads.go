package monitor

import (
	"errors"

	"process-monitor/proc"
)

// ThreadRow is one line of the thread view.
type ThreadRow struct {
	proc.Thread
	CPUPercent float64
}

// ErrProcessGone means the process whose threads were being watched ended.
var ErrProcessGone = errors.New("the process no longer exists")

// ThreadSampler computes the %CPU of each thread of one process, the same
// way Sampler does for processes: from the difference between two readings.
type ThreadSampler struct {
	target    procKey        // the process being watched
	prevCPU   proc.CPUTimes  // machine counters in the previous reading
	prevTimes map[int]uint64 // utime+stime of each thread, by TID
}

// Sample reads the threads of target, sorted by TID. As with processes,
// the first reading of a process has nothing to compare with, so every
// %CPU is 0 until the second one.
func (s *ThreadSampler) Sample(target proc.Process) ([]ThreadRow, error) {
	key := procKey{target.PID, target.StartTime}
	if key != s.target {
		*s = ThreadSampler{target: key} // another process: forget the old counters
	}

	// Make sure the PID still belongs to the same process.
	current, err := proc.Read(target.PID)
	if err != nil || current.StartTime != target.StartTime {
		return nil, ErrProcessGone
	}
	cpu, err := proc.ReadCPUTimes()
	if err != nil {
		return nil, err
	}
	tids, err := proc.ListThreadIDs(target.PID)
	if err != nil {
		return nil, ErrProcessGone
	}

	first := s.prevTimes == nil
	deltaTotal := cpu.Total - s.prevCPU.Total

	times := make(map[int]uint64, len(tids))
	rows := make([]ThreadRow, 0, len(tids))
	for _, tid := range tids {
		t, err := proc.ReadThread(target.PID, tid)
		if err != nil {
			continue // the thread ended between listing and reading it
		}
		used := t.UTime + t.STime
		times[tid] = used

		row := ThreadRow{Thread: t}
		if !first && deltaTotal > 0 {
			// A thread we did not see before was created during this
			// interval, so all its CPU time belongs to it (prev stays 0).
			prev := s.prevTimes[tid]
			if used >= prev {
				row.CPUPercent = percent(used-prev, deltaTotal)
			}
		}
		rows = append(rows, row)
	}

	s.prevCPU = cpu
	s.prevTimes = times
	return rows, nil
}
