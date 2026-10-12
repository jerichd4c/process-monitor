package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"process-monitor/proc"
)

// Screen layout: the summary on top, the table in the middle and one line
// of help at the bottom.
const (
	summaryLines = 3
	headerLine   = summaryLines     // line with the column titles
	firstRowLine = summaryLines + 1 // first line of the process table
)

var (
	styleNormal = tcell.StyleDefault
	styleBar    = tcell.StyleDefault.Reverse(true) // column titles and help

	// Bottom line when it carries a message instead of the help.
	styleOK    = tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack)
	styleError = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite).Bold(true)
	styleWarn  = tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack)
)

// draw repaints the whole screen. tcell keeps the frame in memory and, on
// Show, sends the terminal only the cells that changed: that is why there
// is no flicker and nothing is left over from the previous frame.
func (a *App) draw() {
	a.screen.Clear()
	_, height := a.screen.Size()

	sum := a.snap.Summary
	a.drawLine(0, styleNormal, fmt.Sprintf(
		"Tasks: %d total, %d running, %d sleeping, %d stopped, %d zombie",
		sum.Total, sum.Running, sum.Sleeping, sum.Stopped, sum.Zombie))
	a.drawLine(1, styleNormal, fmt.Sprintf(
		"CPU: %5.1f%%   Load average: %.2f %.2f %.2f",
		sum.CPUPercent, sum.Load.One, sum.Load.Five, sum.Load.Fifteen))
	a.drawLine(2, styleNormal, fmt.Sprintf(
		"Mem: %s total, %s used, %s available",
		formatKB(sum.Mem.Total), formatKB(sum.Mem.Used()), formatKB(sum.Mem.Available)))

	// The middle of the screen is the process table or, in the thread
	// view, the threads of one process.
	helpLine := height - 1
	if a.mode == modeThreads {
		a.drawThreads(helpLine)
	} else {
		a.drawTable(helpLine)
	}

	text, style := a.bottomLine()
	a.drawLine(helpLine, style, text)
	a.screen.Show()
}

// drawTable paints the process table between the summary and the help line.
func (a *App) drawTable(helpLine int) {
	a.drawLine(headerLine, styleBar, fmt.Sprintf(
		"%7s %7s %-8s %s %5s %5s %7s %3s %3s %3s %9s  %s",
		"PID", "PPID", "USER", "S", "%CPU", "%MEM", "RES", "THR", "PR", "NI", "TIME", "COMMAND"))

	// If there are more processes than lines, the table shows a window of
	// them that starts at a.top and always contains the selected one.
	a.scrollToCursor()
	for i := a.top; i < len(a.rows); i++ {
		y := firstRowLine + i - a.top
		if y >= helpLine {
			break
		}
		r := a.rows[i]
		a.drawLine(y, rowStyle(r.State, i == a.cursor), fmt.Sprintf(
			"%7d %7d %-8.8s %s %5.1f %5.1f %7s %3d %3d %3d %9s  %s",
			r.PID, r.PPID, a.users.Name(r.UID), r.State, r.CPUPercent, r.MemPercent,
			formatKB(r.RSSKB), r.NumThreads, r.Priority, r.Nice,
			formatCPUTime(r.UTime+r.STime), r.TreePrefix+r.Cmdline))
	}
	if len(a.rows) == 0 && a.filter != "" {
		a.drawLine(firstRowLine, styleNormal, "  no process matches the filter")
	}
}

// bottomLine decides what the last line shows, in order of importance: a
// question waiting for an answer, the filter being typed, the result of the
// last action, a problem reading /proc, or the list of keys.
func (a *App) bottomLine() (string, tcell.Style) {
	switch {
	case a.mode == modeConfirm:
		return a.confirmText(), styleWarn
	case a.mode == modeNice:
		return a.niceText(), styleWarn
	case a.mode == modeFilter:
		return fmt.Sprintf(" Filter: %s_   (Enter: keep, Esc: clear)", a.filter), styleBar
	case a.message != "" && a.messageIsErr:
		return " " + a.message, styleError
	case a.message != "":
		return " " + a.message, styleOK
	case a.readError != "":
		return " " + a.readError, styleError
	case a.mode == modeThreads:
		return a.threadsHelp(), styleBar
	}
	order := a.sortBy.String()
	if a.tree {
		order = "tree"
	}
	help := fmt.Sprintf(" q:quit c/m/p:sort[%s] t:tree /:filter", order)
	if a.filter != "" {
		help += "[" + a.filter + "]"
	}
	help += fmt.Sprintf(" h:threads k:term K:kill s:stop r:cont n:nice +/-:[%.1fs]", a.interval.Seconds())
	return help, styleBar
}

// rowStyle chooses the colours of a table line. Zombies (finished but not
// collected by their parent) and processes blocked waiting for I/O stand
// out; the selected line is shown in reverse video.
func rowStyle(state string, selected bool) tcell.Style {
	style := styleNormal
	switch state {
	case "Z":
		style = style.Foreground(tcell.ColorRed).Bold(true)
	case "D":
		style = style.Foreground(tcell.ColorYellow).Bold(true)
	case "T", "t":
		style = style.Foreground(tcell.ColorTeal)
	}
	return style.Reverse(selected)
}

// drawLine writes text on line y using the full width of the screen: longer
// text is cut and shorter text is padded with spaces.
func (a *App) drawLine(y int, style tcell.Style, text string) {
	width, height := a.screen.Size()
	if y < 0 || y >= height {
		return
	}
	runes := []rune(text)
	for x := 0; x < width; x++ {
		r := ' '
		if x < len(runes) {
			r = runes[x]
		}
		a.screen.SetContent(x, y, r, nil, style)
	}
}

// formatCPUTime turns ticks into "minutes:seconds.hundredths", like top.
func formatCPUTime(ticks uint64) string {
	hundredths := ticks * 100 / proc.ClockTicks
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths%100)
}

// formatKB shows an amount of memory given in kB with a readable unit.
func formatKB(kb uint64) string {
	switch {
	case kb < 1024:
		return fmt.Sprintf("%dK", kb)
	case kb < 1024*1024:
		return fmt.Sprintf("%.1fM", float64(kb)/1024)
	default:
		return fmt.Sprintf("%.1fG", float64(kb)/(1024*1024))
	}
}
