package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"process-monitor/proc"
)

// fakeHistory makes the app believe that the process with the given PID
// has these %CPU values; every other process counts as ended.
func fakeHistory(app *App, pid int, values []float64) {
	app.history = func(p proc.Process) []float64 {
		if p.PID != pid {
			return nil
		}
		return values
	}
}

func TestGraphRows(t *testing.T) {
	// Two lines of 8 levels each: 16 possible bar sizes for a scale of 100.
	got := graphRows([]float64{0, 25, 50, 75, 100}, 5, 2, 100)
	want := []string{
		"   ▄█", // upper half: only 75 and 100 reach it
		" ▄███", // lower half: 25 fills half a cell, the rest fill it
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGraphRowsAlignment(t *testing.T) {
	// Fewer samples than columns: they go on the right, newest last.
	if got := graphRows([]float64{100}, 4, 1, 100); got[0] != "   █" {
		t.Errorf("got %q, want the bar in the last column", got[0])
	}
	// More samples than columns: only the newest ones are drawn.
	if got := graphRows([]float64{100, 100, 0, 50}, 2, 1, 100); got[0] != " ▄" {
		t.Errorf("got %q, want the last two samples", got[0])
	}
	// A tiny value is still visible, and a huge one does not overflow.
	if got := graphRows([]float64{0.001, 500}, 2, 1, 100); got[0] != "▁█" {
		t.Errorf("got %q, want the thinnest block and a full one", got[0])
	}
}

func TestNiceScale(t *testing.T) {
	cases := map[float64]float64{0: 1, 0.3: 1, 1: 1, 6.2: 10, 12.5: 25, 60: 100, 100: 100, 130: 200, 1570: 1600}
	for highest, want := range cases {
		if got := niceScale(highest); got != want {
			t.Errorf("niceScale(%v) = %v, want %v", highest, got, want)
		}
	}
}

func TestHistoryView(t *testing.T) {
	app, screen := newTestApp(t, 60, 12)
	press(app, tcell.KeyDown) // PID 2
	fakeHistory(app, 2, []float64{0, 6.2, 6.2, 12.5})

	typeText(app, "g")
	if app.mode != modeHistory {
		t.Fatalf("mode = %v, message %q; want the history view", app.mode, app.message)
	}
	app.draw()
	lines := screenLines(screen)

	if !strings.Contains(lines[headerLine], "CPU history of PID 2 (proc-2)") {
		t.Errorf("header = %q", lines[headerLine])
	}
	// 12 lines - 4 above - 1 help - 2 below the graph = 5 graph lines, with
	// the scale (25, the round number above 12.5) on the first one.
	if !strings.HasPrefix(lines[firstRowLine], "  25.0%┤") {
		t.Errorf("top graph line = %q, want the scale label", lines[firstRowLine])
	}
	if !strings.HasPrefix(lines[firstRowLine+4], "   0.0%┤") || !strings.HasSuffix(lines[firstRowLine+4], "███") {
		t.Errorf("bottom graph line = %q, want three bars at the right", lines[firstRowLine+4])
	}
	if !strings.Contains(lines[firstRowLine+6], "now 12.5%") ||
		!strings.Contains(lines[firstRowLine+6], "maximum 12.5%") {
		t.Errorf("stats line = %q", lines[firstRowLine+6])
	}
	if !strings.Contains(lines[11], "Esc/g:back") {
		t.Errorf("bottom line = %q", lines[11])
	}

	// Keys of the table do nothing here.
	calls := fakeApply(app, nil)
	typeText(app, "ksrKnh")
	if len(*calls) != 0 || app.mode != modeHistory {
		t.Errorf("calls %v, mode %v; want nothing to happen", *calls, app.mode)
	}

	press(app, tcell.KeyEscape)
	if app.mode != modeNormal || app.selectedPID != 2 {
		t.Errorf("after Esc: mode %v, selected PID %d", app.mode, app.selectedPID)
	}
	typeText(app, "gg") // "g" also closes it
	if app.mode != modeNormal {
		t.Errorf("after g g: mode %v", app.mode)
	}
}

func TestHistoryViewProcessEnds(t *testing.T) {
	app, _ := newTestApp(t, 80, 24)
	fakeHistory(app, 1, []float64{1, 2, 3})
	typeText(app, "g")
	if app.mode != modeHistory {
		t.Fatalf("mode = %v, want the history view", app.mode)
	}

	// On a later refresh the process is gone.
	fakeHistory(app, 999, nil)
	app.checkHistory()
	if app.mode != modeNormal || !app.messageIsErr ||
		!strings.Contains(app.message, "PID 1 (proc-1)") ||
		!strings.Contains(app.message, "no longer exists") {
		t.Errorf("mode %v, message %q", app.mode, app.message)
	}
}

func TestHistoryViewNoSamplesYet(t *testing.T) {
	app, screen := newTestApp(t, 80, 24)
	fakeHistory(app, 1, []float64{}) // alive, but not measured yet
	typeText(app, "g")
	app.draw()
	if app.mode != modeHistory {
		t.Fatalf("mode = %v, want the history view", app.mode)
	}
	found := false
	for _, line := range screenLines(screen) {
		found = found || strings.Contains(line, "waiting for the first measurement")
	}
	if !found {
		t.Error("the view does not say it is waiting for data")
	}
}

func TestHistoryViewTinyScreens(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {5, 2}, {8, 6}, {9, 7}, {10, 4}, {200, 5}, {30, 60}} {
		app, _ := newTestApp(t, size[0], size[1])
		fakeHistory(app, 1, []float64{0, 50, 100, 3})
		typeText(app, "g")
		app.draw()
	}
}
