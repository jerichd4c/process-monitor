package monitor

import (
	"cmp"
	"slices"
)

// Tree returns the rows ordered as a process tree: each process is followed
// by its children, built from the parent PID (PPID) of every process. The
// TreePrefix of each row holds the lines that draw the hierarchy, like
//
//	systemd
//	├─ cron
//	└─ bash
//	   └─ sleep
//
// A process whose parent is not in rows (PID 1, whose parent is the kernel,
// or a process whose parent was hidden by the filter) starts its own tree.
func Tree(rows []Row) []Row {
	present := make(map[int]bool, len(rows))
	for _, r := range rows {
		present[r.PID] = true
	}

	// Group the processes by parent.
	children := make(map[int][]Row)
	var roots []Row
	for _, r := range rows {
		if present[r.PPID] && r.PPID != r.PID {
			children[r.PPID] = append(children[r.PPID], r)
		} else {
			roots = append(roots, r)
		}
	}
	byPID := func(a, b Row) int { return cmp.Compare(a.PID, b.PID) }

	out := make([]Row, 0, len(rows))
	visited := make(map[int]bool, len(rows))

	// walk adds a process and then, recursively, its children (depth-first).
	// indent is what the lines above it left pending for this level.
	var walk func(r Row, indent string, isRoot, isLast bool)
	walk = func(r Row, indent string, isRoot, isLast bool) {
		if visited[r.PID] {
			return
		}
		visited[r.PID] = true

		childIndent := indent
		switch {
		case isRoot:
			r.TreePrefix = ""
		case isLast:
			r.TreePrefix = indent + "└─ "
			childIndent += "   " // nothing else hangs from the parent
		default:
			r.TreePrefix = indent + "├─ "
			childIndent += "│  " // the parent still has children below
		}
		out = append(out, r)

		kids := children[r.PID]
		slices.SortFunc(kids, byPID)
		for i, kid := range kids {
			walk(kid, childIndent, false, i == len(kids)-1)
		}
	}

	slices.SortFunc(roots, byPID)
	for _, r := range roots {
		walk(r, "", true, false)
	}
	// Safety net: if inconsistent data ever formed a loop of parents, those
	// processes have no root. Show them anyway instead of losing them.
	for _, r := range rows {
		walk(r, "", true, false)
	}
	return out
}
