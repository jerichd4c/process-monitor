package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/monitor"
	"process-monitor/proc"
)

// newTestApp builds an App that draws on an in-memory screen of the given
// size, with 50 fake processes: PIDs 1 to 50, named "proc-1" to "proc-50".
func newTestApp(t *testing.T, width, height int) (*App, tcell.SimulationScreen) {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)

	var rows []monitor.Row
	for pid := 1; pid <= 50; pid++ {
		name := fmt.Sprintf("proc-%d", pid)
		rows = append(rows, monitor.Row{Process: proc.Process{
			PID: pid, State: "S", Comm: name, Cmdline: name + " --with --many --arguments",
		}})
	}
	app := &App{
		screen:   screen,
		users:    proc.Users{0: "root"},
		interval: time.Second,
		history:  func(proc.Process) []float64 { return nil },
		snap:     monitor.Snapshot{Rows: rows, Summary: monitor.Summary{Total: 50, Sleeping: 50}},
	}
	app.updateView()
	return app, screen
}

// screenLines returns what is on the simulated screen, one string per line.
func screenLines(screen tcell.SimulationScreen) []string {
	cells, width, height := screen.GetContents()
	lines := make([]string, height)
	for y := 0; y < height; y++ {
		var line strings.Builder
		for x := 0; x < width; x++ {
			line.WriteString(string(cells[y*width+x].Runes))
		}
		lines[y] = line.String()
	}
	return lines
}

// press sends keys to the app as if the user had typed them.
func press(app *App, keys ...tcell.Key) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for _, key := range keys {
		app.handleEvent(tcell.NewEventKey(key, 0, tcell.ModNone), ticker)
	}
}

// typeText sends each character of text as a key press.
func typeText(app *App, text string) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for _, r := range text {
		app.handleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), ticker)
	}
}

func TestDrawLayout(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)
	app.draw()
	lines := screenLines(screen)

	if !strings.HasPrefix(lines[0], "Tasks: 50 total") {
		t.Errorf("line 0 = %q, want the summary", lines[0])
	}
	if !strings.Contains(lines[headerLine], "PID") || !strings.Contains(lines[headerLine], "COMMAND") {
		t.Errorf("header line = %q", lines[headerLine])
	}
	if !strings.Contains(lines[23], "q:quit") {
		t.Errorf("last line = %q, want the help", lines[23])
	}
	// 24 lines - 3 summary - 1 header - 1 help = 19 processes fit.
	if !strings.HasPrefix(strings.TrimSpace(lines[22]), "19 ") {
		t.Errorf("last table line = %q, want process 19", lines[22])
	}
}

func TestDrawTinyScreens(t *testing.T) {
	// It must never panic, however small the window is.
	for _, size := range [][2]int{{1, 1}, {5, 2}, {10, 4}, {200, 5}, {30, 60}} {
		app, _ := newTestApp(t, size[0], size[1])
		press(app, tcell.KeyDown, tcell.KeyPgDn, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyHome)
		app.draw()
	}
}

func TestCursorAndScroll(t *testing.T) {
	app, screen := newTestApp(t, 100, 24) // 19 table lines

	press(app, tcell.KeyDown, tcell.KeyDown)
	if app.selectedPID != 3 {
		t.Errorf("after 2 downs selected PID = %d, want 3", app.selectedPID)
	}
	press(app, tcell.KeyUp, tcell.KeyUp, tcell.KeyUp) // cannot go above the first
	if app.selectedPID != 1 {
		t.Errorf("at the top selected PID = %d, want 1", app.selectedPID)
	}

	// Going to the end scrolls the table: the last line shows process 50.
	press(app, tcell.KeyEnd)
	app.draw()
	lines := screenLines(screen)
	if app.selectedPID != 50 || !strings.HasPrefix(strings.TrimSpace(lines[22]), "50 ") {
		t.Errorf("after End: selected %d, last line %q", app.selectedPID, lines[22])
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[firstRowLine]), "32 ") {
		t.Errorf("first table line = %q, want process 32", lines[firstRowLine])
	}
}

func TestSelectionFollowsProcess(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	press(app, tcell.KeyDown, tcell.KeyDown, tcell.KeyDown, tcell.KeyDown) // PID 5

	// Process 40 starts using CPU, so it moves to the first line. The
	// selection must stay on PID 5, which is now one line lower.
	app.snap.Rows[39].CPUPercent = 50
	app.updateView()
	if app.rows[0].PID != 40 {
		t.Fatalf("first row = PID %d, want 40", app.rows[0].PID)
	}
	if app.selectedPID != 5 || app.rows[app.cursor].PID != 5 {
		t.Errorf("selected PID = %d at row with PID %d, want 5",
			app.selectedPID, app.rows[app.cursor].PID)
	}

	// Sorting by PID puts it back in fifth place.
	typeText(app, "p")
	if app.cursor != 4 {
		t.Errorf("cursor = %d, want 4", app.cursor)
	}
}

func TestFilterTyping(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)

	typeText(app, "/proc-4") // proc-4 and proc-40 .. proc-49
	if app.mode != modeFilter || len(app.rows) != 11 {
		t.Fatalf("mode = %v, rows = %d; want filter mode, 11", app.mode, len(app.rows))
	}
	// While typing, "q" is text and must not quit the program.
	typeText(app, "q")
	if len(app.rows) != 0 {
		t.Errorf("rows = %d, want 0 for filter %q", len(app.rows), app.filter)
	}
	app.draw()
	if lines := screenLines(screen); !strings.Contains(lines[23], "Filter: proc-4q") {
		t.Errorf("bottom line = %q, want the filter being typed", lines[23])
	}

	press(app, tcell.KeyBackspace2, tcell.KeyEnter)
	if app.mode != modeNormal || app.filter != "proc-4" || len(app.rows) != 11 {
		t.Errorf("after Enter: mode %v, filter %q, rows %d", app.mode, app.filter, len(app.rows))
	}

	// The first Esc removes the filter instead of quitting.
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	quit := app.handleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), ticker)
	if quit || app.filter != "" || len(app.rows) != 50 {
		t.Errorf("after Esc: quit %v, filter %q, rows %d", quit, app.filter, len(app.rows))
	}
}

func TestFilterSelectsFirstMatch(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	press(app, tcell.KeyDown, tcell.KeyDown, tcell.KeyDown) // PID 4

	// PID 4 does not match "proc-3": the selection goes to the first row
	// that does, not to whatever ends up on the fourth line.
	typeText(app, "/proc-3")
	if app.selectedPID != 3 || app.cursor != 0 {
		t.Errorf("selected PID %d at line %d, want PID 3 at line 0", app.selectedPID, app.cursor)
	}

	// A process that still matches stays selected while the filter changes.
	press(app, tcell.KeyEnter, tcell.KeyDown, tcell.KeyDown) // PID 31
	typeText(app, "/")
	press(app, tcell.KeyBackspace2) // filter is now "proc-"
	if app.selectedPID != 31 {
		t.Errorf("selected PID = %d, want 31", app.selectedPID)
	}
}

func TestRowStyle(t *testing.T) {
	normal := rowStyle("S", false)
	if rowStyle("Z", false) == normal || rowStyle("D", false) == normal {
		t.Error("zombie and I/O-blocked rows must not look like normal rows")
	}
	if rowStyle("Z", false) == rowStyle("D", false) {
		t.Error("zombie and I/O-blocked rows must look different from each other")
	}
	if rowStyle("S", true) == normal {
		t.Error("the selected row must look different")
	}
}

func TestSetIntervalLimits(t *testing.T) {
	app, _ := newTestApp(t, 80, 24)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	app.setInterval(0, ticker)
	if app.interval != minInterval {
		t.Errorf("interval = %v, want %v", app.interval, minInterval)
	}
	app.setInterval(time.Hour, ticker)
	if app.interval != maxInterval {
		t.Errorf("interval = %v, want %v", app.interval, maxInterval)
	}
}

func TestFormatKB(t *testing.T) {
	cases := map[uint64]string{0: "0K", 512: "512K", 15436: "15.1M", 7017052: "6.7G"}
	for kb, want := range cases {
		if got := formatKB(kb); got != want {
			t.Errorf("formatKB(%d) = %q, want %q", kb, got, want)
		}
	}
}
