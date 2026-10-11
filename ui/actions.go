package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/control"
	"process-monitor/proc"
)

// forceWord is what the user must type to confirm a force kill. It is a
// separate, stronger confirmation than the single key of a normal
// terminate, because SIGKILL gives the process no chance to save its work.
const forceWord = "yes"

// pendingAction is an action waiting for the user's confirmation.
type pendingAction struct {
	action control.Action
	target proc.Process // the process as it was when the key was pressed
	typed  string       // what the user has typed so far (force kill only)
}

// startAction begins an action on the selected process. Suspend and resume
// can be undone, so they are applied at once; terminate and force kill ask
// for confirmation first.
func (a *App) startAction(action control.Action) {
	if len(a.rows) == 0 {
		a.setMessage(true, "No process selected")
		return
	}
	// Remember the process itself, not the cursor: the table may be
	// reordered while the question is on screen.
	a.pending = pendingAction{action: action, target: a.rows[a.cursor].Process}

	switch action {
	case control.Terminate, control.Kill:
		a.mode = modeConfirm
	default:
		a.runPending()
	}
}

// handleConfirmKey handles a key while a confirmation is on screen.
func (a *App) handleConfirmKey(ev *tcell.EventKey) {
	if a.pending.action == control.Kill {
		// Force kill: the user must type the whole word and press Enter.
		switch ev.Key() {
		case tcell.KeyEscape:
			a.cancelPending()
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			a.pending.typed = dropLastRune(a.pending.typed)
		case tcell.KeyRune:
			a.pending.typed += string(ev.Rune())
		case tcell.KeyEnter:
			if a.pending.typed == forceWord {
				a.runPending()
			} else {
				a.cancelPending()
			}
		}
		return
	}

	// Terminate: "y" confirms, any other key cancels.
	if ev.Key() == tcell.KeyRune && (ev.Rune() == 'y' || ev.Rune() == 'Y') {
		a.runPending()
	} else {
		a.cancelPending()
	}
}

// cancelPending leaves the confirmation without doing anything.
func (a *App) cancelPending() {
	a.mode = modeNormal
	a.setMessage(false, "Cancelled: nothing was sent to %s", describe(a.pending.target))
}

// runPending applies the pending action and reports, in the bottom line,
// which action was applied to which process, or why it could not be.
func (a *App) runPending() {
	a.mode = modeNormal
	action, target := a.pending.action, a.pending.target

	if err := a.apply(target, action); err != nil {
		a.setMessage(true, "Cannot %v %s: %v", action, describe(target), err)
		return
	}
	a.setMessage(false, "Sent %s (%v) to %s", action.SignalName(), action, describe(target))

	// The signal is delivered asynchronously: read the system again in a
	// moment so the effect (gone, stopped, running) shows up right away.
	a.soon = time.After(150 * time.Millisecond)
}

// setMessage puts a message in the bottom line until the next key.
func (a *App) setMessage(isErr bool, format string, args ...any) {
	a.message = fmt.Sprintf(format, args...)
	a.messageIsErr = isErr
}

// describe identifies a process in the messages.
func describe(p proc.Process) string {
	return fmt.Sprintf("PID %d (%s)", p.PID, p.Comm)
}

// confirmText is the question shown while a confirmation is pending.
func (a *App) confirmText() string {
	target := describe(a.pending.target)
	if a.pending.action == control.Kill {
		return fmt.Sprintf(" FORCE KILL %s? It cannot save its work. Type %q + Enter (Esc cancels): %s_",
			target, forceWord, a.pending.typed)
	}
	return fmt.Sprintf(" Terminate %s? [y/N]", target)
}
