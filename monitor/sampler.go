// Package monitor turns the raw counters read by package proc into what the
// monitor shows: %CPU computed from the difference between two refreshes,
// and the system summary.
package monitor

import (
	"errors"

	"process-monitor/proc"
)

// Row is one line of the process table.
type Row struct {
	proc.Process
	CPUPercent float64
	MemPercent float64
	TreePrefix string // lines that draw the hierarchy, only in the tree view
}

// Summary is the header of the monitor.
type Summary struct {
	Total    int // processes shown (kernel threads excluded)
	Running  int // state R
	Sleeping int // states S, D, I
	Stopped  int // states T, t
	Zombie   int // state Z: finished but not yet collected by the parent

	CPUPercent float64 // usage of the whole machine
	Load       proc.LoadAvg
	Mem        proc.MemInfo
}

// Snapshot is everything gathered in one refresh.
type Snapshot struct {
	Rows    []Row
	Summary Summary
	Skipped int // processes that ended while being read, or unreadable data
	Denied  int // processes that exist but whose data we may not read
}

// procKey identifies a process across refreshes. The PID alone is not
// enough: when a process ends, the kernel can give its PID to a new one.
// The pair (PID, start time) is unique.
type procKey struct {
	pid   int
	start uint64
}

// Sampler remembers the counters of the previous refresh, which are needed
// to compute %CPU.
type Sampler struct {
	prevCPU   proc.CPUTimes
	prevTimes map[procKey]uint64    // utime+stime of each process, in ticks
	history   map[procKey][]float64 // last %CPU values of each process, oldest first
}

// Sample reads the whole system once. The first call has nothing to compare
// with, so every %CPU is 0 until the second call.
func (s *Sampler) Sample() (Snapshot, error) {
	cpu, err := proc.ReadCPUTimes()
	if err != nil {
		return Snapshot{}, err
	}
	mem, err := proc.ReadMemInfo()
	if err != nil {
		return Snapshot{}, err
	}
	load, err := proc.ReadLoadAvg()
	if err != nil {
		return Snapshot{}, err
	}
	pids, err := proc.ListPIDs()
	if err != nil {
		return Snapshot{}, err
	}

	first := s.prevTimes == nil
	deltaTotal := cpu.Total - s.prevCPU.Total // ticks that passed on all CPUs

	snap := Snapshot{Summary: Summary{Load: load, Mem: mem}}
	if !first && deltaTotal > 0 {
		deltaIdle := cpu.Idle - s.prevCPU.Idle
		snap.Summary.CPUPercent = percent(deltaTotal-deltaIdle, deltaTotal)
	}

	times := make(map[procKey]uint64, len(pids))
	history := make(map[procKey][]float64, len(pids))
	for _, pid := range pids {
		p, err := proc.Read(pid)
		if err != nil {
			// Skip it, but count why: a process that ended is normal, a
			// process we may not read is worth telling the user about.
			if errors.Is(err, proc.ErrNoAccess) {
				snap.Denied++
			} else {
				snap.Skipped++
			}
			continue
		}
		if p.IsKernelThread() {
			continue
		}

		key := procKey{p.PID, p.StartTime}
		used := p.UTime + p.STime
		times[key] = used

		row := Row{Process: p, MemPercent: p.MemPercent(mem.Total)}
		if !first && deltaTotal > 0 {
			// A process we did not see before started during this interval,
			// so all its CPU time was used inside it (prev stays 0).
			prev := s.prevTimes[key]
			if used >= prev {
				row.CPUPercent = percent(used-prev, deltaTotal)
			}
		}
		// Every live process gets an entry, even before its first value,
		// so that History can tell "no values yet" from "it ended".
		values := s.history[key]
		if !first {
			values = appendCapped(values, row.CPUPercent, HistoryLen)
		}
		history[key] = values
		snap.Rows = append(snap.Rows, row)
		countState(&snap.Summary, p.State)
	}

	// Keep only this refresh: processes that ended disappear from the maps.
	s.prevCPU = cpu
	s.prevTimes = times
	s.history = history
	return snap, nil
}

// percent returns part/total as a percentage.
func percent(part, total uint64) float64 {
	return float64(part) / float64(total) * 100
}

// countState adds one process to the summary counters.
func countState(sum *Summary, state string) {
	sum.Total++
	switch state {
	case "R":
		sum.Running++
	case "S", "D", "I":
		sum.Sleeping++
	case "T", "t":
		sum.Stopped++
	case "Z":
		sum.Zombie++
	}
}
