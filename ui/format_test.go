package ui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"process-monitor/monitor"
	"process-monitor/proc"
)

func TestFormatCPUTime(t *testing.T) {
	cases := map[uint64]string{
		0:             "0:00.00",
		147:           "0:01.47",
		6000:          "1:00.00",
		5999999:       "999:59.99",  // the longest in minutes: 9 characters
		6000000:       "16:40:00",   // 1000 minutes: switch to hours
		360000 * 5000: "5000:00:00", // 5000 hours of CPU
	}
	for ticks, want := range cases {
		got := formatCPUTime(ticks)
		if got != want {
			t.Errorf("formatCPUTime(%d) = %q, want %q", ticks, got, want)
		}
		if ticks <= 6000000 && len(got) > 9 {
			t.Errorf("formatCPUTime(%d) = %q does not fit in the 9-character column", ticks, got)
		}
	}
}

func TestFormatPriority(t *testing.T) {
	cases := map[int]string{20: "20", 39: "39", 0: "0", -51: "-51", -100: "rt"}
	for priority, want := range cases {
		if got := formatPriority(priority); got != want || len(got) > 3 {
			t.Errorf("formatPriority(%d) = %q, want %q (3 characters at most)", priority, got, want)
		}
	}
}

func TestExtremeValuesKeepColumnsAligned(t *testing.T) {
	app, screen := newTestApp(t, 120, 24)
	// The biggest values every column can hold.
	app.snap.Rows[0] = monitor.Row{
		Process: proc.Process{
			PID: 4194304, PPID: 4194303, State: "R", UID: 0,
			NumThreads: 9999, Priority: -100, Nice: -20,
			UTime: 5999999, RSSKB: 1023 * 1024 * 1024,
			Comm: "big", Cmdline: "big-process",
		},
		CPUPercent: 100, MemPercent: 100,
	}
	app.updateView()
	app.draw()
	lines := screenLines(screen)

	// Sorted by CPU, the extreme process is the first row and an ordinary
	// one (PID 2) is the second.
	header := lines[headerLine]
	extreme := lines[firstRowLine]
	normal := lines[firstRowLine+1]

	want := "4194304 4194303 root     R 100.0 100.0 1023.0G 9999  rt -20 999:59.99  big-process"
	if strings.TrimRight(extreme, " ") != want {
		t.Errorf("extreme row:\n got %q\nwant %q", strings.TrimRight(extreme, " "), want)
	}
	// The command starts in the same column in the header and in both rows.
	col := strings.Index(header, "COMMAND")
	if strings.Index(extreme, "big-process") != col || strings.Index(normal, "proc-") != col {
		t.Errorf("COMMAND column: header %d, extreme row %d, normal row %d",
			col, strings.Index(extreme, "big-process"), strings.Index(normal, "proc-"))
	}
}

func TestWideCharacters(t *testing.T) {
	app, screen := newTestApp(t, 40, 6)
	// Each of these characters takes two cells on screen.
	app.drawLine(0, styleNormal, "日本語|ab")
	app.drawLine(1, styleNormal, "ascii|ab")
	app.drawLine(2, styleBar, strings.Repeat("語", 100)) // far longer than the line
	app.screen.Show()

	cells, width, _ := screen.GetContents()
	cellAt := func(x, y int) string { return string(cells[y*width+x].Runes) }
	// Three wide characters fill cells 0 to 5, so "|" lands on cell 6;
	// in the plain line it is on cell 5.
	if cellAt(6, 0) != "|" || cellAt(7, 0) != "a" {
		t.Errorf("after 3 wide characters: cell 6 = %q, cell 7 = %q; want | and a", cellAt(6, 0), cellAt(7, 0))
	}
	if cellAt(5, 1) != "|" {
		t.Errorf("plain line: cell 5 = %q, want |", cellAt(5, 1))
	}
	// The rest of the line is filled with spaces.
	if cellAt(39, 0) != " " || cellAt(39, 1) != " " {
		t.Errorf("end of line not padded: %q, %q", cellAt(39, 0), cellAt(39, 1))
	}
	if !utf8.ValidString(cellAt(38, 2)) {
		t.Error("a wide character was cut in half at the end of the line")
	}
}

func TestSkippedProcessesAreShown(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)
	app.draw()
	if line := screenLines(screen)[0]; strings.Contains(line, "could not be read") {
		t.Errorf("summary = %q, want no note when nothing was skipped", line)
	}

	app.snap.Skipped = 3
	app.draw()
	if line := screenLines(screen)[0]; !strings.Contains(line, "(3 could not be read)") {
		t.Errorf("summary = %q, want it to say 3 processes were skipped", line)
	}
}

func TestEscNeverQuits(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	esc := func() bool {
		return app.handleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), ticker)
	}

	// In the table, with or without a filter, and after closing each view.
	for i := 0; i < 3; i++ {
		if esc() {
			t.Fatal("Esc in the table quit the monitor")
		}
	}
	for _, keys := range []string{"/abc", "?", "k", "K", "n", "t"} {
		typeText(app, keys)
		for i := 0; i < 3; i++ {
			if esc() {
				t.Fatalf("Esc after %q quit the monitor", keys)
			}
		}
		if app.mode != modeNormal || app.filter != "" {
			t.Errorf("after %q and Esc: mode %v, filter %q; want the plain table", keys, app.mode, app.filter)
		}
	}

	// q and Ctrl+C are the ways out.
	if !app.handleEvent(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone), ticker) {
		t.Error("q did not quit")
	}
	if !app.handleEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone), ticker) {
		t.Error("Ctrl+C did not quit")
	}
}

func TestHistoryStatsCountAllSamples(t *testing.T) {
	// 300 samples are kept, but only 72 fit in an 80-column window.
	samples := make([]float64, 300)
	samples[0] = 40 // an old peak, no longer on screen
	app, screen := newTestApp(t, 80, 24)
	fakeHistory(app, 1, samples)
	typeText(app, "g")
	app.draw()

	text := strings.Join(screenLines(screen), "\n")
	if !strings.Contains(text, "maximum 40.0%") || !strings.Contains(text, "last 300 samples") {
		t.Errorf("the numbers are not about all the samples:\n%s", text)
	}
	// The scale follows what is on screen (all zeros), not the old peak.
	if !strings.Contains(text, "   1.0%┤") {
		t.Errorf("scale label not found:\n%s", text)
	}
}

func TestDeniedProcessesAreShown(t *testing.T) {
	app, screen := newTestApp(t, 140, 24)
	app.snap.Skipped = 1
	app.snap.Denied = 2
	app.draw()
	line := screenLines(screen)[0]
	if !strings.Contains(line, "(1 could not be read)") ||
		!strings.Contains(line, "(2 hidden: no permission to read them)") {
		t.Errorf("summary = %q, want both notes", line)
	}
}
