package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"process-monitor/control"
	"process-monitor/proc"
)

// niceCall records one use of the fake setNice function.
type niceCall struct {
	pid  int
	nice int
}

// fakeSetNice replaces control.SetNice in the app, so the tests never
// change a real priority.
func fakeSetNice(app *App, result error) *[]niceCall {
	var calls []niceCall
	app.setNice = func(p proc.Process, nice int) error {
		calls = append(calls, niceCall{p.PID, nice})
		return result
	}
	return &calls
}

func TestNicePrompt(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)
	calls := fakeSetNice(app, nil)
	press(app, tcell.KeyDown) // PID 2

	typeText(app, "n")
	if app.mode != modeNice {
		t.Fatalf("mode = %v, want the nice prompt", app.mode)
	}
	// Letters are ignored, and a minus only counts at the start.
	typeText(app, "1x0-q")
	app.draw()
	if last := screenLines(screen)[23]; !strings.Contains(last, "New nice for PID 2 (proc-2), now 0") ||
		!strings.Contains(last, ": 10_") {
		t.Errorf("bottom line = %q", last)
	}

	press(app, tcell.KeyEnter)
	if len(*calls) != 1 || (*calls)[0] != (niceCall{2, 10}) {
		t.Fatalf("calls = %v, want nice 10 for PID 2", *calls)
	}
	if app.mode != modeNormal || app.messageIsErr ||
		!strings.Contains(app.message, "PID 2 (proc-2) changed from 0 to 10") {
		t.Errorf("mode %v, message %q", app.mode, app.message)
	}
}

func TestNiceNegativeAndCancel(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	calls := fakeSetNice(app, nil)

	typeText(app, "n-5")
	press(app, tcell.KeyEnter)
	if len(*calls) != 1 || (*calls)[0].nice != -5 {
		t.Errorf("calls = %v, want nice -5", *calls)
	}

	typeText(app, "n15")
	press(app, tcell.KeyEscape)
	if len(*calls) != 1 || app.mode != modeNormal {
		t.Errorf("Esc did not cancel: calls %v, mode %v", *calls, app.mode)
	}
}

func TestNiceInvalidValues(t *testing.T) {
	for _, typed := range []string{"", "-", "20", "-21", "999"} {
		app, _ := newTestApp(t, 100, 24)
		calls := fakeSetNice(app, nil)

		typeText(app, "n"+typed)
		press(app, tcell.KeyEnter)
		if len(*calls) != 0 || !app.messageIsErr || !strings.Contains(app.message, "Invalid nice") {
			t.Errorf("typed %q: calls %v, message %q", typed, *calls, app.message)
		}
	}
}

func TestNiceErrorsAreReported(t *testing.T) {
	for _, problem := range []error{control.ErrNeedRoot, control.ErrDenied, control.ErrGone} {
		app, _ := newTestApp(t, 100, 24)
		fakeSetNice(app, problem)

		typeText(app, "n-10")
		press(app, tcell.KeyEnter)
		if !app.messageIsErr || !strings.Contains(app.message, problem.Error()) ||
			!strings.Contains(app.message, "PID 1 (proc-1) to -10") {
			t.Errorf("message = %q, want it to explain %q", app.message, problem)
		}
	}
}
