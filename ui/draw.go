package ui

import (
	"fmt"
	"strconv"

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
	tasks := fmt.Sprintf(
		"Tasks: %d total, %d running, %d sleeping, %d stopped, %d zombie",
		sum.Total, sum.Running, sum.Sleeping, sum.Stopped, sum.Zombie)
	if a.snap.Skipped > 0 {
		// Processes that ended while we were reading them, or that we are
		// not allowed to read: they are left out, never a reason to fail.
		tasks += fmt.Sprintf("   (%d could not be read)", a.snap.Skipped)
	}
	a.drawLine(0, styleNormal, tasks)
	a.drawLine(1, styleNormal, fmt.Sprintf(
		"CPU: %5.1f%%   Load average: %.2f %.2f %.2f",
		sum.CPUPercent, sum.Load.One, sum.Load.Five, sum.Load.Fifteen))
	a.drawLine(2, styleNormal, fmt.Sprintf(
		"Mem: %s total, %s used, %s available",
		formatKB(sum.Mem.Total), formatKB(sum.Mem.Used()), formatKB(sum.Mem.Available)))

	// The middle of the screen is the process table or, in the other
	// views, the threads or the CPU history of one process.
	helpLine := height - 1
	switch a.mode {
	case modeThreads:
		a.drawThreads(helpLine)
	case modeHelp:
		a.drawHelp(helpLine)
	case modeHistory:
		a.drawHistory(helpLine)
	default:
		a.drawTable(helpLine)
	}

	text, style := a.bottomLine()
	a.drawLine(helpLine, style, text)
	a.screen.Show()
}

// drawTable paints the process table between the summary and the help line.
func (a *App) drawTable(helpLine int) {
	a.drawLine(headerLine, styleBar, fmt.Sprintf(
		"%7s %7s %-8s %s %5s %5s %7s %4s %3s %3s %9s  %s",
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
			"%7d %7d %-8.8s %s %5.1f %5.1f %7s %4d %3s %3d %9s  %s",
			r.PID, r.PPID, a.users.Name(r.UID), r.State, r.CPUPercent, r.MemPercent,
			formatKB(r.RSSKB), r.NumThreads, formatPriority(r.Priority), r.Nice,
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
	case a.mode == modeHistory:
		return a.historyHelp(), styleBar
	case a.mode == modeHelp:
		return " Help   Arrows:scroll   any other key:back", styleBar
	}

	// Normal mode: the two keys everybody needs and the current settings.
	// The full list of keys is in the help screen.
	order := a.sortBy.String()
	if a.tree {
		order = "tree"
	}
	status := fmt.Sprintf(" ?:help  q:quit   sort[%s]  refresh[%.1fs]", order, a.interval.Seconds())
	if a.filter != "" {
		status += "  filter[" + a.filter + "]"
	}
	return status, styleBar
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
	// Put writes one character and says how many cells it took: most take
	// one, but some (Chinese, Japanese, emoji...) are twice as wide, and
	// counting them as one would misplace everything after them.
	x := 0
	for text != "" && x < width {
		rest, cells := a.screen.Put(x, y, text, style)
		if cells <= 0 || len(rest) >= len(text) {
			break // nothing was written: do not loop forever
		}
		text = rest
		x += cells
	}
	for ; x < width; x++ {
		a.screen.SetContent(x, y, ' ', nil, style)
	}
}

// formatCPUTime turns ticks into "minutes:seconds.hundredths", like top.
// A process that has used 1000 minutes or more is shown as
// "hours:minutes:seconds" instead, so the column never grows too wide.
func formatCPUTime(ticks uint64) string {
	hundredths := ticks * 100 / proc.ClockTicks
	minutes := hundredths / 6000
	seconds := hundredths / 100 % 60
	if minutes >= 1000 {
		return fmt.Sprintf("%d:%02d:%02d", minutes/60, minutes%60, seconds)
	}
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths%100)
}

// formatPriority shows the priority the kernel uses. Normal processes have
// 0 to 39 (20 + nice). Real-time processes have negative values; the
// highest real-time priority, -100, is shown as "rt", like top does.
func formatPriority(priority int) string {
	if priority <= -100 {
		return "rt"
	}
	return strconv.Itoa(priority)
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
