package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/monitor"
	"process-monitor/proc"
)

// threadView is the state of the screen that lists the threads of one
// process.
type threadView struct {
	process proc.Process // whose threads are shown
	sampler monitor.ThreadSampler
	rows    []monitor.ThreadRow
	top     int // index of the first thread on screen
}

// openThreads switches to the thread view of the selected process.
func (a *App) openThreads() {
	if len(a.rows) == 0 {
		a.setMessage(true, "No process selected")
		return
	}
	a.threads = threadView{process: a.rows[a.cursor].Process}
	a.mode = modeThreads
	a.refreshThreads()
	// %CPU needs two readings: take the second one soon instead of waiting
	// for a whole refresh interval.
	a.soon = time.After(300 * time.Millisecond)
}

// closeThreads goes back to the process table.
func (a *App) closeThreads() {
	a.mode = modeNormal
	a.threads = threadView{}
}

// refreshThreads reads the threads again. If the process ended, it says so
// and goes back to the table.
func (a *App) refreshThreads() {
	rows, err := a.threads.sampler.Sample(a.threads.process)
	if err != nil {
		gone := describe(a.threads.process)
		a.closeThreads()
		a.setMessage(true, "Cannot show the threads of %s: %v", gone, err)
		return
	}
	a.threads.rows = rows
}

// handleThreadsKey handles a key in the thread view.
func (a *App) handleThreadsKey(ev *tcell.EventKey, ticker *time.Ticker) (quit bool) {
	page := a.tableHeight()
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyBackspace, tcell.KeyBackspace2:
		a.closeThreads()
	case tcell.KeyUp:
		a.threads.top--
	case tcell.KeyDown:
		a.threads.top++
	case tcell.KeyPgUp:
		a.threads.top -= page
	case tcell.KeyPgDn:
		a.threads.top += page
	case tcell.KeyHome:
		a.threads.top = 0
	case tcell.KeyEnd:
		a.threads.top = len(a.threads.rows)
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return true
		case 'h':
			a.closeThreads()
		case '+':
			a.setInterval(a.interval+intervalStep, ticker)
		case '-':
			a.setInterval(a.interval-intervalStep, ticker)
		}
	}
	return false
}

// drawThreads paints the thread list in the area of the process table.
func (a *App) drawThreads(helpLine int) {
	a.drawLine(headerLine, styleBar, fmt.Sprintf(
		"%7s %s %5s %3s %3s %9s  %s", "TID", "S", "%CPU", "PR", "NI", "TIME", "THREAD NAME"))

	// Keep the scroll position inside the list.
	visible := a.tableHeight()
	a.threads.top = max(min(a.threads.top, len(a.threads.rows)-visible), 0)

	for i := a.threads.top; i < len(a.threads.rows); i++ {
		y := firstRowLine + i - a.threads.top
		if y >= helpLine {
			break
		}
		t := a.threads.rows[i]
		name := t.Name
		if t.TID == a.threads.process.PID {
			name += "  (main thread)"
		}
		a.drawLine(y, rowStyle(t.State, false), fmt.Sprintf(
			"%7d %s %5.1f %3s %3d %9s  %s",
			t.TID, t.State, t.CPUPercent, formatPriority(t.Priority), t.Nice,
			formatCPUTime(t.UTime+t.STime), name))
	}
}

// threadsHelp is the bottom line of the thread view.
func (a *App) threadsHelp() string {
	return fmt.Sprintf(" Threads of %s: %d   Esc/h:back  Arrows:scroll  +/-:[%.1fs]  q:quit",
		describe(a.threads.process), len(a.threads.rows), a.interval.Seconds())
}
