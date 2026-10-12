package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestTreeView(t *testing.T) {
	app, screen := newTestApp(t, 100, 24)
	// Make a small family: 1 is the parent of 2 and 3, and 3 of 4.
	app.snap.Rows[1].PPID = 1
	app.snap.Rows[2].PPID = 1
	app.snap.Rows[3].PPID = 3
	// Process 4 is the busiest: sorted by CPU it is the first line.
	app.snap.Rows[3].CPUPercent = 50
	app.updateView()
	press(app, tcell.KeyHome) // select PID 4
	if app.selectedPID != 4 {
		t.Fatalf("selected PID = %d, want 4", app.selectedPID)
	}

	typeText(app, "t")
	app.draw()
	lines := screenLines(screen)
	wantEnds := []string{"proc-1", "├─ proc-2", "└─ proc-3", "   └─ proc-4"}
	for i, want := range wantEnds {
		// Two spaces separate the TIME column from the command.
		line := lines[firstRowLine+i]
		if !strings.Contains(line, ".00  "+want+" --with") {
			t.Errorf("tree line %d = %q, want the command to be %q", i, line, want)
		}
	}
	// The selection stays on the same process, now on the fourth line.
	if app.selectedPID != 4 || app.cursor != 3 {
		t.Errorf("selected PID %d at line %d, want PID 4 at line 3", app.selectedPID, app.cursor)
	}
	if !strings.Contains(lines[23], "sort[tree]") {
		t.Errorf("help line = %q, want it to say the order is the tree", lines[23])
	}

	// "t" again goes back to the previous order, without the tree lines.
	typeText(app, "t")
	if app.tree || app.rows[0].PID != 4 || app.rows[0].TreePrefix != "" {
		t.Errorf("after second t: tree %v, first row PID %d, prefix %q",
			app.tree, app.rows[0].PID, app.rows[0].TreePrefix)
	}

	// Choosing a sort column also leaves the tree view.
	typeText(app, "tp")
	if app.tree || app.rows[0].PID != 1 {
		t.Errorf("after t p: tree %v, first row PID %d", app.tree, app.rows[0].PID)
	}
}

func TestTreeWithFilter(t *testing.T) {
	app, _ := newTestApp(t, 100, 24)
	app.snap.Rows[1].PPID = 1  // 2 is a child of 1
	app.snap.Rows[19].PPID = 2 // 20 is a child of 2
	app.updateView()

	// The filter hides PID 1, so PID 2 becomes the root of what is left.
	typeText(app, "t/proc-2")
	press(app, tcell.KeyEnter)
	if len(app.rows) != 11 || app.rows[0].PID != 2 || app.rows[0].TreePrefix != "" {
		t.Fatalf("rows %d, first PID %d, prefix %q", len(app.rows), app.rows[0].PID, app.rows[0].TreePrefix)
	}
	if app.rows[1].PID != 20 || app.rows[1].TreePrefix != "└─ " {
		t.Errorf("second row = PID %d with prefix %q, want PID 20 under PID 2",
			app.rows[1].PID, app.rows[1].TreePrefix)
	}
}
