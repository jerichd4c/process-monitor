package ui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/control"
)

// startNice asks for a new nice value for the selected process.
func (a *App) startNice() {
	if len(a.rows) == 0 {
		a.setMessage(true, "No process selected")
		return
	}
	// As with the signals, remember the process itself, not the cursor.
	a.pending = pendingAction{target: a.rows[a.cursor].Process}
	a.mode = modeNice
}

// handleNiceKey handles a key while the user types the new nice value.
func (a *App) handleNiceKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		a.mode = modeNormal
		a.setMessage(false, "Cancelled: the priority of %s was not changed", describe(a.pending.target))
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		a.pending.typed = dropLastRune(a.pending.typed)
	case tcell.KeyRune:
		// Only what can be part of a number: digits and a leading minus.
		r := ev.Rune()
		if (r >= '0' && r <= '9') || (r == '-' && a.pending.typed == "") {
			a.pending.typed += string(r)
		}
	case tcell.KeyEnter:
		a.runNice()
	}
}

// runNice applies the typed nice value and reports the result.
func (a *App) runNice() {
	a.mode = modeNormal
	target := a.pending.target

	nice, err := strconv.Atoi(a.pending.typed)
	if err != nil || nice < control.MinNice || nice > control.MaxNice {
		a.setMessage(true, "Invalid nice %q: it must be a number from %d to %d",
			a.pending.typed, control.MinNice, control.MaxNice)
		return
	}
	if err := a.setNice(target, nice); err != nil {
		a.setMessage(true, "Cannot change nice of %s to %d: %v", describe(target), nice, err)
		return
	}
	a.setMessage(false, "Nice of %s changed from %d to %d", describe(target), target.Nice, nice)
	a.soon = time.After(150 * time.Millisecond) // show the new value right away
}

// niceText is the question shown while the user types the nice value.
func (a *App) niceText() string {
	return fmt.Sprintf(" New nice for %s, now %d (%d = highest priority, %d = lowest): %s_",
		describe(a.pending.target), a.pending.target.Nice,
		control.MinNice, control.MaxNice, a.pending.typed)
}
