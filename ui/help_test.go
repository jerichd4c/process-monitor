package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestHelpScreen(t *testing.T) {
	// 40 lines: the whole help fits.
	app, screen := newTestApp(t, 90, 40)

	typeText(app, "?")
	if app.mode != modeHelp {
		t.Fatalf("mode = %v, want the help screen", app.mode)
	}
	app.draw()
	text := strings.Join(screenLines(screen), "\n")

	// Every key the program reacts to must be explained.
	for _, want := range []string{
		"Up/Down", "PgUp/PgDn", "Home/End",
		"c  by CPU", "m  by memory", "p  by PID", "t  process tree",
		"/  type part of a name",
		"h  threads", "g  CPU history",
		"k  terminate (SIGTERM)", "K  force kill (SIGKILL)",
		"s  suspend (SIGSTOP)", "r  resume (SIGCONT)", "n  change the priority",
		"+  slower", "-  faster",
		"q, Esc or Ctrl+C",
		"Z zombie", "D blocked", "T stopped",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the help does not mention %q", want)
		}
	}
	if !strings.Contains(text, "any other key:back") {
		t.Error("the bottom line does not say how to leave the help")
	}
}

func TestHelpKeys(t *testing.T) {
	app, screen := newTestApp(t, 90, 12) // only 7 lines for the help
	calls := fakeApply(app, nil)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	typeText(app, "?")
	app.draw()
	before := screenLines(screen)[firstRowLine+1]

	// The arrows scroll the text instead of closing the help.
	press(app, tcell.KeyDown, tcell.KeyDown, tcell.KeyDown)
	app.draw()
	after := screenLines(screen)[firstRowLine+1]
	if app.mode != modeHelp || before == after {
		t.Errorf("mode %v; line before %q, after %q: want it to scroll", app.mode, before, after)
	}

	// Scrolling far past both ends must not break anything.
	press(app, tcell.KeyPgDn, tcell.KeyPgDn, tcell.KeyPgDn, tcell.KeyEnd, tcell.KeyDown)
	app.draw()
	if last := screenLines(screen)[10]; !strings.Contains(last, "T stopped") {
		t.Errorf("last help line after End = %q, want the end of the help", last)
	}
	press(app, tcell.KeyPgUp, tcell.KeyPgUp, tcell.KeyPgUp, tcell.KeyHome, tcell.KeyUp)
	app.draw()
	if app.helpTop != 0 {
		t.Errorf("helpTop = %d after going to the start, want 0", app.helpTop)
	}

	// "q" only closes the help: it must not quit, and "k" must not start
	// an action while the help is on screen.
	quit := app.handleEvent(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone), ticker)
	if quit || app.mode != modeNormal {
		t.Errorf("q in the help: quit %v, mode %v; want it to go back", quit, app.mode)
	}
	typeText(app, "?k")
	if app.mode != modeNormal || len(*calls) != 0 {
		t.Errorf("k in the help: mode %v, calls %v", app.mode, *calls)
	}
}

func TestHelpTinyScreens(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {5, 2}, {10, 4}, {200, 5}, {30, 60}} {
		app, _ := newTestApp(t, size[0], size[1])
		typeText(app, "?")
		press(app, tcell.KeyDown, tcell.KeyPgDn, tcell.KeyEnd)
		app.draw()
	}
}

func TestBottomLineFitsIn80Columns(t *testing.T) {
	app, screen := newTestApp(t, 80, 24)
	typeText(app, "t/systemd-resolved")
	press(app, tcell.KeyEnter)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	app.setInterval(10*time.Second, ticker)
	app.draw()

	last := strings.TrimRight(screenLines(screen)[23], " ")
	for _, want := range []string{"?:help", "q:quit", "sort[tree]", "refresh[10.0s]", "filter[systemd-resolved]"} {
		if !strings.Contains(last, want) {
			t.Errorf("bottom line %q does not show %q", last, want)
		}
	}
	if len([]rune(last)) >= 80 {
		t.Errorf("bottom line is %d characters long: it does not fit", len([]rune(last)))
	}
}
