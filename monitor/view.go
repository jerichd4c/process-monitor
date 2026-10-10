package monitor

import (
	"cmp"
	"slices"
	"strings"
)

// SortKey says which column orders the process table.
type SortKey int

const (
	SortByCPU SortKey = iota // highest %CPU first
	SortByMem                // highest memory first
	SortByPID                // lowest PID first
)

// String returns the name shown in the interface.
func (k SortKey) String() string {
	switch k {
	case SortByCPU:
		return "CPU"
	case SortByMem:
		return "MEM"
	default:
		return "PID"
	}
}

// Sort orders rows in place. Processes that tie (for example, all the ones
// at 0.0 %CPU) are ordered by PID so they do not jump around between
// refreshes.
func Sort(rows []Row, key SortKey) {
	slices.SortFunc(rows, func(a, b Row) int {
		var order int
		switch key {
		case SortByCPU:
			order = cmp.Compare(b.CPUPercent, a.CPUPercent) // b first: descending
		case SortByMem:
			order = cmp.Compare(b.RSSKB, a.RSSKB)
		}
		if order != 0 {
			return order
		}
		return cmp.Compare(a.PID, b.PID)
	})
}

// Filter returns a new slice with the rows whose name or command line
// contains text, ignoring upper and lower case. An empty text keeps all.
func Filter(rows []Row, text string) []Row {
	text = strings.ToLower(text)
	var kept []Row
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.Comm), text) ||
			strings.Contains(strings.ToLower(r.Cmdline), text) {
			kept = append(kept, r)
		}
	}
	return kept
}
