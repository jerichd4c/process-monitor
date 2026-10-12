package ui

import "process-monitor/monitor"

// updateView rebuilds the rows of the table from the last snapshot,
// applying the filter and the sort order (or the tree order), and puts the
// cursor back on the process that was selected.
func (a *App) updateView() {
	a.rows = monitor.Filter(a.snap.Rows, a.filter)
	if a.tree {
		a.rows = monitor.Tree(a.rows)
	} else {
		monitor.Sort(a.rows, a.sortBy)
	}

	// Look for the selected process in its new position. If it is gone
	// (it ended or the filter hides it), the cursor stays on the same line.
	for i, r := range a.rows {
		if r.PID == a.selectedPID {
			a.cursor = i
			break
		}
	}
	a.moveCursor(0)
}

// moveCursor moves the selection delta lines down (or up if negative),
// without leaving the table, and remembers which process is selected.
func (a *App) moveCursor(delta int) {
	if len(a.rows) == 0 {
		a.cursor, a.selectedPID = 0, 0
		return
	}
	a.cursor = min(max(a.cursor+delta, 0), len(a.rows)-1)
	a.selectedPID = a.rows[a.cursor].PID
}

// tableHeight is how many process lines fit on the screen.
func (a *App) tableHeight() int {
	_, height := a.screen.Size()
	return max(height-firstRowLine-1, 0) // -1 for the help line
}

// scrollToCursor moves the visible window of the table just enough for the
// selected line to be inside it.
func (a *App) scrollToCursor() {
	visible := a.tableHeight()
	if a.cursor < a.top {
		a.top = a.cursor
	}
	if a.cursor >= a.top+visible {
		a.top = a.cursor - visible + 1
	}
	// Do not leave empty lines at the bottom if there are rows above.
	a.top = min(a.top, max(len(a.rows)-visible, 0))
	a.top = max(a.top, 0)
}
