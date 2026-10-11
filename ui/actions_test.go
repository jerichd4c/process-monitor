package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"process-monitor/control"
	"process-monitor/proc"
)

// call records one use of the fake apply function.
type call struct {
	pid    int
	action control.Action
}

// fakeApply replaces control.Apply in the app, so the tests never send a
// real signal. It records the calls and returns the given error.
func fakeApply(app *App, result error) *[]call {
	var calls []call
	app.apply = func(p proc.Process, action control.Action) error {
		calls = append(calls, call{p.PID, action})
		return result
	}
	return &calls
}

func TestSuspendAndResumeAreImmediate(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	calls := fakeApply(app, nil)
	press(app, tcell.KeyDown, tcell.KeyDown) // PID 3

	typeText(app, "s")
	if len(*calls) != 1 || (*calls)[0] != (call{3, control.Suspend}) {
		t.Fatalf("calls after s = %v, want suspend of PID 3", *calls)
	}
	// The message says which action was applied to which process.
	for _, want := range []string{"SIGSTOP", "suspend", "PID 3", "proc-3"} {
		if !strings.Contains(app.message, want) {
			t.Errorf("message %q does not mention %q", app.message, want)
		}
	}

	typeText(app, "r")
	if len(*calls) != 2 || (*calls)[1] != (call{3, control.Resume}) {
		t.Errorf("calls after r = %v, want resume of PID 3", *calls)
	}
}

func TestTerminateAsksFirst(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)
	calls := fakeApply(app, nil)

	typeText(app, "k")
	if app.mode != modeConfirm || len(*calls) != 0 {
		t.Fatalf("after k: mode %v, calls %v; want a question and no signal", app.mode, *calls)
	}
	app.draw()
	if last := screenLines(screen)[23]; !strings.Contains(last, "Terminate PID 1 (proc-1)? [y/N]") {
		t.Errorf("bottom line = %q, want the question", last)
	}

	typeText(app, "n") // any key other than y cancels
	if app.mode != modeNormal || len(*calls) != 0 {
		t.Errorf("after n: mode %v, calls %v; want cancelled", app.mode, *calls)
	}

	typeText(app, "ky")
	if len(*calls) != 1 || (*calls)[0] != (call{1, control.Terminate}) {
		t.Errorf("calls after ky = %v, want terminate of PID 1", *calls)
	}
}

func TestForceKillNeedsTheWholeWord(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	calls := fakeApply(app, nil)

	// A single "y" is not enough for a force kill.
	typeText(app, "Ky")
	press(app, tcell.KeyEnter)
	if len(*calls) != 0 || app.mode != modeNormal {
		t.Fatalf("after K y Enter: calls %v, mode %v; want cancelled", *calls, app.mode)
	}

	typeText(app, "Kyes")
	if len(*calls) != 0 {
		t.Fatalf("signal sent before Enter: %v", *calls)
	}
	press(app, tcell.KeyEnter)
	if len(*calls) != 1 || (*calls)[0] != (call{1, control.Kill}) {
		t.Errorf("calls = %v, want force kill of PID 1", *calls)
	}

	typeText(app, "Kyes")
	press(app, tcell.KeyEscape)
	if len(*calls) != 1 {
		t.Errorf("Esc did not cancel: calls %v", *calls)
	}
}

func TestTargetIsFixedWhenTheQuestionAppears(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	calls := fakeApply(app, nil)
	press(app, tcell.KeyDown) // PID 2
	typeText(app, "k")

	// While the question is on screen a refresh reorders the table and
	// another process takes the line of PID 2.
	app.snap.Rows[29].CPUPercent = 90
	app.snap.Rows[30].CPUPercent = 80
	app.updateView()

	typeText(app, "y")
	if len(*calls) != 1 || (*calls)[0].pid != 2 {
		t.Errorf("calls = %v, want the signal for PID 2", *calls)
	}
}

func TestActionErrorsAreReported(t *testing.T) {
	for _, problem := range []error{control.ErrGone, control.ErrDenied, control.ErrSelf} {
		app, screen := newTestApp(t, 100, 24)
		fakeApply(app, problem)

		typeText(app, "s")
		if !app.messageIsErr || !strings.Contains(app.message, problem.Error()) ||
			!strings.Contains(app.message, "PID 1 (proc-1)") {
			t.Errorf("message = %q (error %v), want it to explain %q", app.message, app.messageIsErr, problem)
		}
		app.draw() // and it must not crash
		if last := screenLines(screen)[23]; !strings.Contains(last, "Cannot suspend") {
			t.Errorf("bottom line = %q", last)
		}
		// The message disappears with the next key.
		press(app, tcell.KeyDown)
		if app.message != "" {
			t.Errorf("message still there after a key: %q", app.message)
		}
	}
}

func TestActionWithEmptyTable(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	calls := fakeApply(app, nil)
	typeText(app, "/nothing-matches")
	press(app, tcell.KeyEnter)

	typeText(app, "s")
	typeText(app, "k")
	if len(*calls) != 0 || app.mode != modeNormal || !app.messageIsErr {
		t.Errorf("calls %v, mode %v, message %q; want no action", *calls, app.mode, app.message)
	}
}
