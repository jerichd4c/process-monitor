package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"process-monitor/proc"
)

// selectSelf puts the test process itself, read from /proc, in the first
// row of the table and selects it.
func selectSelf(t *testing.T, app *App) proc.Process {
	t.Helper()
	self, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	app.snap.Rows[0].Process = self
	app.updateView()
	for i, r := range app.rows {
		if r.PID == self.PID {
			app.cursor = i
		}
	}
	app.moveCursor(0)
	return self
}

func TestThreadView(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)
	self := selectSelf(t, app)

	typeText(app, "h")
	if app.mode != modeThreads {
		t.Fatalf("mode = %v, message %q; want the thread view", app.mode, app.message)
	}
	// A Go program always has several threads; one of them is the main
	// thread, whose TID is the PID.
	var hasMain bool
	for _, row := range app.threads.rows {
		if row.TID == self.PID {
			hasMain = true
		}
		if row.State == "" {
			t.Errorf("thread %d has no state", row.TID)
		}
	}
	if len(app.threads.rows) < 2 || !hasMain {
		t.Fatalf("threads = %d, main thread found = %v", len(app.threads.rows), hasMain)
	}

	app.draw()
	lines := screenLines(screen)
	if !strings.Contains(lines[headerLine], "TID") || !strings.Contains(lines[headerLine], "THREAD NAME") {
		t.Errorf("header = %q", lines[headerLine])
	}
	if !strings.Contains(lines[firstRowLine], "(main thread)") {
		t.Errorf("first thread line = %q, want the main thread", lines[firstRowLine])
	}
	if !strings.Contains(lines[23], "Threads of PID") || !strings.Contains(lines[23], "back") {
		t.Errorf("bottom line = %q", lines[23])
	}

	// Scrolling far past both ends must not break the view.
	press(app, tcell.KeyPgDn, tcell.KeyPgDn, tcell.KeyEnd, tcell.KeyDown)
	app.draw()
	press(app, tcell.KeyPgUp, tcell.KeyPgUp, tcell.KeyHome, tcell.KeyUp, tcell.KeyUp)
	app.draw()
	if app.threads.top != 0 {
		t.Errorf("top = %d after scrolling to the start, want 0", app.threads.top)
	}

	// Keys of the table do nothing here: no signal can be sent by mistake.
	calls := fakeApply(app, nil)
	typeText(app, "ksrKn")
	if len(*calls) != 0 || app.mode != modeThreads {
		t.Errorf("calls %v, mode %v; want nothing to happen", *calls, app.mode)
	}

	press(app, tcell.KeyEscape)
	if app.mode != modeNormal || app.selectedPID != self.PID {
		t.Errorf("after Esc: mode %v, selected PID %d", app.mode, app.selectedPID)
	}

	// "h" also closes the view.
	typeText(app, "hh")
	if app.mode != modeNormal {
		t.Errorf("after h h: mode %v", app.mode)
	}
}

func TestThreadViewProcessGone(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	// The fake process of the first row (PID 1 with start time 0) does not
	// match any real process, as if it had ended.
	typeText(app, "h")
	if app.mode != modeNormal || !app.messageIsErr ||
		!strings.Contains(app.message, "PID 1 (proc-1)") ||
		!strings.Contains(app.message, "no longer exists") {
		t.Errorf("mode %v, message %q", app.mode, app.message)
	}
}

func TestThreadViewClosesWhenProcessEnds(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	selectSelf(t, app)
	typeText(app, "h")

	// The watched process "ends": the next refresh finds another start time.
	app.threads.process.StartTime++
	app.refreshThreads()
	if app.mode != modeNormal || !strings.Contains(app.message, "no longer exists") {
		t.Errorf("mode %v, message %q", app.mode, app.message)
	}
}

func TestThreadViewTinyScreens(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {5, 2}, {10, 4}, {200, 5}} {
		app, _ := newTestApp(t, size[0], size[1])
		selectSelf(t, app)
		typeText(app, "h")
		press(app, tcell.KeyDown, tcell.KeyPgDn, tcell.KeyEnd)
		app.draw()
	}
}
