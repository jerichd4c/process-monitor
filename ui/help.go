package ui

import (
	"github.com/gdamore/tcell/v2"
)

// helpEntry is one line of the help screen.
type helpEntry struct {
	text  string
	style tcell.Style
}

// helpContent lists every key of the monitor. The lines about the states
// use the same colours as the rows of the table.
func helpContent() []helpEntry {
	plain := func(text string) helpEntry { return helpEntry{text, styleNormal} }
	return []helpEntry{
		plain(""),
		plain("  Move      Up/Down  PgUp/PgDn  Home/End      select a process"),
		plain(""),
		plain("  Order     c  by CPU usage     m  by memory     p  by PID"),
		plain("            t  process tree (parent and children), on/off"),
		plain("  Filter    /  type part of a name; Enter keeps it, Esc clears it"),
		plain(""),
		plain("  Views     h  threads of the selected process"),
		plain("            g  CPU history graph of the selected process"),
		plain(""),
		plain("  Actions   k  terminate (SIGTERM); asks y/N first"),
		plain("            K  force kill (SIGKILL); asks to type \"yes\""),
		plain("            s  suspend (SIGSTOP)        r  resume (SIGCONT)"),
		plain("            n  change the priority (nice, from -20 to 19)"),
		plain(""),
		plain("  Refresh   +  slower     -  faster     (from 0.5 s to 10 s)"),
		plain("  Quit      q or Ctrl+C     (Esc only goes back; it never quits)"),
		plain(""),
		plain("  Columns   S state   RES memory in RAM   THR threads   PR priority   NI nice"),
		plain("            %CPU and %MEM: 100 is the whole machine"),
		plain("  States    R running    S sleeping    I idle"),
		{"            Z zombie: it ended, but its parent has not collected it", rowStyle("Z", false)},
		{"            D blocked, waiting for input/output", rowStyle("D", false)},
		{"            T stopped (suspended)", rowStyle("T", false)},
	}
}

// openHelp shows the help screen.
func (a *App) openHelp() {
	a.mode = modeHelp
	a.helpTop = 0
}

// handleHelpKey handles a key in the help screen: the arrows scroll it (it
// may not fit in a small window) and any other key closes it.
func (a *App) handleHelpKey(ev *tcell.EventKey) {
	page := a.tableHeight()
	switch ev.Key() {
	case tcell.KeyUp:
		a.helpTop--
	case tcell.KeyDown:
		a.helpTop++
	case tcell.KeyPgUp:
		a.helpTop -= page
	case tcell.KeyPgDn:
		a.helpTop += page
	case tcell.KeyHome:
		a.helpTop = 0
	case tcell.KeyEnd:
		a.helpTop = len(helpContent())
	default:
		a.mode = modeNormal
	}
}

// drawHelp paints the help in the area of the process table.
func (a *App) drawHelp(helpLine int) {
	a.drawLine(headerLine, styleBar, " Keys of the process monitor")

	content := helpContent()
	// Keep the scroll position inside the text.
	a.helpTop = max(min(a.helpTop, len(content)-a.tableHeight()), 0)

	for i := a.helpTop; i < len(content); i++ {
		y := firstRowLine + i - a.helpTop
		if y >= helpLine {
			break
		}
		a.drawLine(y, content[i].style, content[i].text)
	}
}
