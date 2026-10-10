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
	snap     monitor.Snapshot // what is on screen
	interval time.Duration    // time between refreshes
	status   string           // message for the bottom line
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
}

// handleEvent reacts to one event and reports whether the user asked to
// quit.
func (a *App) handleEvent(ev tcell.Event, ticker *time.Ticker) (quit bool) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		// The window changed size: repaint everything with the new size.
		a.screen.Sync()
	case *tcell.EventKey:
		switch {
		case ev.Key() == tcell.KeyCtrlC, ev.Key() == tcell.KeyEscape:
			return true
		case ev.Rune() == 'q', ev.Rune() == 'Q':
			return true
		case ev.Rune() == '+':
			a.setInterval(a.interval+intervalStep, ticker)
		case ev.Rune() == '-':
			a.setInterval(a.interval-intervalStep, ticker)
		}
	}
	return false
}

// setInterval changes the refresh interval, keeping it inside the limits.
func (a *App) setInterval(d time.Duration, ticker *time.Ticker) {
	d = min(max(d, minInterval), maxInterval)
	a.interval = d
	ticker.Reset(d)
}
