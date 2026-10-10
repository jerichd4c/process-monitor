package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/monitor"
	"process-monitor/proc"
)

// newTestApp builds an App that draws on an in-memory screen of the given
// size, with a few fake processes.
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
		rows = append(rows, monitor.Row{Process: proc.Process{
			PID: pid, State: "S", Cmdline: "a-very-long-command --with --many --arguments",
		}})
	}
	app := &App{
		screen:   screen,
		users:    proc.Users{0: "root"},
		interval: time.Second,
		snap:     monitor.Snapshot{Rows: rows, Summary: monitor.Summary{Total: 50, Sleeping: 50}},
	}
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

func TestDrawLayout(t *testing.T) {
	app, screen := newTestApp(t, 80, 24)
	app.draw()
	lines := screenLines(screen)

	if !strings.HasPrefix(lines[0], "Tasks: 50 total") {
		t.Errorf("line 0 = %q, want the summary", lines[0])
	}
	if !strings.Contains(lines[headerLine], "PID") || !strings.Contains(lines[headerLine], "COMMAND") {
		t.Errorf("header line = %q", lines[headerLine])
	}
	if !strings.Contains(lines[23], "q: quit") {
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
		app.draw()
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
