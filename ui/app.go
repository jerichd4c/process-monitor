// Package ui is the full-screen terminal interface of the monitor.
package ui

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/control"
	"process-monitor/monitor"
	"process-monitor/proc"
)

// Limits of the refresh interval the user can choose with + and -.
const (
	minInterval  = 500 * time.Millisecond
	maxInterval  = 10 * time.Second
	intervalStep = 500 * time.Millisecond
)

// mode says how the keyboard is interpreted.
type mode int

const (
	modeNormal  mode = iota // each key is a command
	modeFilter              // the user is typing the filter
	modeConfirm             // the user is answering "are you sure?"
	modeNice                // the user is typing a new nice value
	modeThreads             // the thread view of one process is on screen
	modeHistory             // the CPU history of one process is on screen
	modeHelp                // the help screen is on screen
)

// App holds the state of the interface. Only the main loop touches it, so
// it needs no locks.
type App struct {
	screen   tcell.Screen
	sampler  monitor.Sampler
	users    proc.Users
	snap     monitor.Snapshot // last reading of the system
	interval time.Duration    // time between refreshes
	mode     mode

	// Bottom line: the result of the last action (until the next key) or a
	// problem reading /proc.
	message      string
	messageIsErr bool
	readError    string

	// What the table shows: snap.Rows after filtering and sorting.
	rows   []monitor.Row
	sortBy monitor.SortKey
	tree   bool   // show the parent/child hierarchy instead of sorting
	filter string // text the process name must contain

	// The selection follows a process (by PID), not a position, so it does
	// not jump to another process when the table is reordered.
	selectedPID int
	cursor      int // index in rows of the selected process
	top         int // index in rows of the first line on screen

	threads threadView // thread view, see threads.go

	helpTop int // first line of the help on screen, see help.go

	// CPU history view, see history.go.
	historyOf proc.Process                 // whose history is shown
	history   func(proc.Process) []float64 // sampler.History, replaced in tests

	// Action waiting for the user's confirmation, see actions.go.
	pending pendingAction
	apply   func(proc.Process, control.Action) error // control.Apply, replaced in tests
	setNice func(proc.Process, int) error            // control.SetNice, replaced in tests
	soon    <-chan time.Time                         // extra refresh shortly after an action
}

// Run starts the interface and returns when the user quits.
func Run() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	app := &App{
		screen:   screen,
		users:    proc.LoadUsers(),
		interval: time.Second,
		apply:    control.Apply,
		setNice:  control.SetNice,
	}
	app.history = app.sampler.History
	app.loop()
	return nil
}

// loop is the main loop: draw, then wait for a key, a tick of the refresh
// timer or a signal, whichever comes first.
func (a *App) loop() {
	// Restore the terminal (echo, cursor, normal screen) however we leave,
	// even if the program panics.
	defer func() {
		p := recover()
		a.screen.Fini()
		if p != nil {
			panic(p)
		}
	}()
	a.screen.HideCursor()

	// PollEvent blocks until something happens, so it runs in its own
	// goroutine and hands the events to the main loop through a channel.
	events := make(chan tcell.Event)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return // the screen was closed
			}
			events <- ev
		}
	}()

	// Signals sent from outside (kill, closing the terminal) also end the
	// program through the normal path, so the terminal is restored.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	a.refresh()
	// %CPU needs two readings: take the second one soon, so the first
	// screen does not show zeros for a whole interval.
	a.soon = time.After(300 * time.Millisecond)
	for {
		a.draw()
		select {
		case ev := <-events:
			if quit := a.handleEvent(ev, ticker); quit {
				return
			}
		case <-ticker.C:
			a.refresh()
		case <-a.soon:
			// An action was just applied: show its effect without waiting
			// for the next tick. (A nil channel never fires.)
			a.soon = nil
			a.refresh()
		case <-signals:
			return
		}
	}
}

// refresh reads the system again. On error the old data stays on screen
// and the problem is reported in the bottom line.
func (a *App) refresh() {
	snap, err := a.sampler.Sample()
	if err != nil {
		a.readError = "cannot read /proc: " + err.Error()
		return
	}
	a.snap = snap
	a.readError = ""
	a.updateView()
	switch a.mode {
	case modeThreads:
		a.refreshThreads()
	case modeHistory:
		a.checkHistory()
	}
}

// handleEvent reacts to one event and reports whether the user asked to
// quit.
func (a *App) handleEvent(ev tcell.Event, ticker *time.Ticker) (quit bool) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		// The window changed size: repaint everything with the new size.
		a.screen.Sync()
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyCtrlC {
			return true
		}
		a.message = "" // the last result stays until the next key
		switch a.mode {
		case modeFilter:
			a.handleFilterKey(ev)
		case modeNice:
			a.handleNiceKey(ev)
		case modeHelp:
			a.handleHelpKey(ev)
		case modeHistory:
			return a.handleHistoryKey(ev, ticker)
		case modeThreads:
			return a.handleThreadsKey(ev, ticker)
		case modeConfirm:
			a.handleConfirmKey(ev)
		default:
			return a.handleKey(ev, ticker)
		}
	}
	return false
}

// handleKey handles a key in the normal mode, where each key is a command.
func (a *App) handleKey(ev *tcell.EventKey, ticker *time.Ticker) (quit bool) {
	switch ev.Key() {
	case tcell.KeyEscape:
		// Esc always means "go back": here it removes the filter. It never
		// quits, so pressing it once too often cannot close the monitor.
		if a.filter != "" {
			a.filter = ""
			a.updateView()
		}
	case tcell.KeyUp:
		a.moveCursor(-1)
	case tcell.KeyDown:
		a.moveCursor(1)
	case tcell.KeyPgUp:
		a.moveCursor(-a.tableHeight())
	case tcell.KeyPgDn:
		a.moveCursor(a.tableHeight())
	case tcell.KeyHome:
		a.moveCursor(-len(a.rows))
	case tcell.KeyEnd:
		a.moveCursor(len(a.rows))
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return true
		case '+':
			a.setInterval(a.interval+intervalStep, ticker)
		case '-':
			a.setInterval(a.interval-intervalStep, ticker)
		case 'c':
			a.setSort(monitor.SortByCPU)
		case 'm':
			a.setSort(monitor.SortByMem)
		case 'p':
			a.setSort(monitor.SortByPID)
		case 't':
			a.tree = !a.tree
			a.updateView()
		case '/':
			a.mode = modeFilter
		case 'k':
			a.startAction(control.Terminate)
		case 'K':
			a.startAction(control.Kill)
		case 's':
			a.startAction(control.Suspend)
		case 'r':
			a.startAction(control.Resume)
		case '?':
			a.openHelp()
		case 'g':
			a.openHistory()
		case 'h':
			a.openThreads()
		case 'n':
			a.startNice()
		}
	}
	return false
}

// handleFilterKey handles a key while the user types the filter: letters
// are text, not commands. The table is filtered as the text changes.
func (a *App) handleFilterKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEnter:
		a.mode = modeNormal // keep the filter
	case tcell.KeyEscape:
		a.mode = modeNormal
		a.filter = ""
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		a.filter = dropLastRune(a.filter)
	case tcell.KeyRune:
		a.filter += string(ev.Rune())
	}
	// If the new filter hides the selected process, select the first match
	// (updateView moves the cursor back if the process is still visible).
	a.cursor = 0
	a.updateView()
}

// setInterval changes the refresh interval, keeping it inside the limits.
func (a *App) setInterval(d time.Duration, ticker *time.Ticker) {
	d = min(max(d, minInterval), maxInterval)
	a.interval = d
	ticker.Reset(d)
}

// setSort changes the column that orders the table.
func (a *App) setSort(key monitor.SortKey) {
	a.sortBy = key
	a.tree = false // choosing a column leaves the tree view
	a.updateView()
}

// dropLastRune removes the last character of s, if it has one.
func dropLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	return string(runes[:len(runes)-1])
}
