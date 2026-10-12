package monitor

import "process-monitor/proc"

// HistoryLen is how many %CPU values are remembered for each process: one
// per refresh, so with the default interval it is the last five minutes.
const HistoryLen = 300

// History returns the %CPU of a process in the last refreshes, oldest
// first. It is nil if the process is not known (it ended), and empty if it
// is alive but has no values yet.
func (s *Sampler) History(p proc.Process) []float64 {
	values, known := s.history[procKey{p.PID, p.StartTime}]
	if !known {
		return nil
	}
	// A copy, so the caller cannot change what the sampler remembers.
	return append([]float64{}, values...)
}

// appendCapped adds value at the end of values and, if there are more than
// limit, drops the oldest ones.
func appendCapped(values []float64, value float64, limit int) []float64 {
	values = append(values, value)
	if len(values) > limit {
		values = values[len(values)-limit:]
	}
	return values
}
