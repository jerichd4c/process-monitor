// Package ui is the full-screen terminal interface of the monitor.
package ui

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/monitor"
	"process-monitor/proc"
)

// Limits of the refresh interval the user can choose with + and -.
const (
	minInterval  = 500 * time.Millisecond
	maxInterval  = 10 * time.Second
	intervalStep = 500 * time.Millisecond
)

// App holds the state of the interface. Only the main loop touches it, so
// it needs no locks.
type App struct {
	screen   tcell.Screen
	sampler  monitor.Sampler
	users    proc.Users
	snap     monitor.Snapshot // last reading of the system
	interval time.Duration    // time between refreshes
	status   string           // message for the bottom line

	// What the table shows: snap.Rows after filtering and sorting.
	rows    []monitor.Row
	sortBy  monitor.SortKey
	filter  string // text the process name must contain
	editing bool   // true while the user is typing the filter

	// The selection follows a process (by PID), not a position, so it does
	// not jump to another process when the table is reordered.
	selectedPID int
	cursor      int // index in rows of the selected process
	top         int // index in rows of the first line on screen
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
	}
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
	for {
		a.draw()
		select {
		case ev := <-events:
			if quit := a.handleEvent(ev, ticker); quit {
				return
			}
		case <-ticker.C:
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
		a.status = "cannot read /proc: " + err.Error()
		return
	}
	a.snap = snap
	a.status = ""
	a.updateView()
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
		if a.editing {
			a.handleFilterKey(ev)
			return false
		}
		return a.handleKey(ev, ticker)
	}
	return false
}

// handleKey handles a key in the normal mode, where each key is a command.
func (a *App) handleKey(ev *tcell.EventKey, ticker *time.Ticker) (quit bool) {
	switch ev.Key() {
	case tcell.KeyEscape:
		if a.filter == "" {
			return true
		}
		a.filter = "" // the first Esc only removes the filter
		a.updateView()
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
		case '/':
			a.editing = true
		}
	}
	return false
}

// handleFilterKey handles a key while the user types the filter: letters
// are text, not commands. The table is filtered as the text changes.
func (a *App) handleFilterKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEnter:
		a.editing = false // keep the filter
	case tcell.KeyEscape:
		a.editing = false
		a.filter = ""
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if runes := []rune(a.filter); len(runes) > 0 {
			a.filter = string(runes[:len(runes)-1])
		}
	case tcell.KeyRune:
		a.filter += string(ev.Rune())
	}
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
	a.updateView()
}
