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

	a.drawLine(headerLine, styleBar, fmt.Sprintf(
		"%7s %7s %-8s %s %5s %5s %7s %3s %3s %9s  %s",
		"PID", "PPID", "USER", "S", "%CPU", "%MEM", "RES", "THR", "NI", "TIME", "COMMAND"))

	// The table uses the lines between the header and the help line; with
	// a small window we simply show fewer processes.
	helpLine := height - 1
	for i, r := range a.snap.Rows {
		y := firstRowLine + i
		if y >= helpLine {
			break
		}
		a.drawLine(y, styleNormal, fmt.Sprintf(
			"%7d %7d %-8.8s %s %5.1f %5.1f %7s %3d %3d %9s  %s",
			r.PID, r.PPID, a.users.Name(r.UID), r.State, r.CPUPercent, r.MemPercent,
			formatKB(r.RSSKB), r.NumThreads, r.Nice, formatCPUTime(r.UTime+r.STime), r.Cmdline))
	}

	help := fmt.Sprintf(" q: quit   +/-: refresh interval (%.1fs)", a.interval.Seconds())
	if a.status != "" {
		help = " " + a.status
	}
	a.drawLine(helpLine, styleBar, help)

	a.screen.Show()
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
