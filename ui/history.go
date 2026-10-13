package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	"process-monitor/monitor"
	"process-monitor/proc"
)

// axisWidth is the space on the left of the graph for the scale labels.
const axisWidth = 8

// blocks are the characters that fill a cell from the bottom in eighths:
// blocks[0] is empty and blocks[8] is a full cell.
var blocks = []rune(" ▁▂▃▄▅▆▇█")

// openHistory switches to the CPU history of the selected process.
func (a *App) openHistory() {
	if len(a.rows) == 0 {
		a.setMessage(true, "No process selected")
		return
	}
	a.historyOf = a.rows[a.cursor].Process
	a.mode = modeHistory
	a.checkHistory()
}

// closeHistory goes back to the process table.
func (a *App) closeHistory() {
	a.mode = modeNormal
	a.historyOf = proc.Process{}
}

// checkHistory goes back to the table if the process being watched ended.
func (a *App) checkHistory() {
	if a.history(a.historyOf) == nil {
		gone := describe(a.historyOf)
		a.closeHistory()
		a.setMessage(true, "Cannot show the CPU history of %s: the process no longer exists", gone)
	}
}

// handleHistoryKey handles a key in the history view.
func (a *App) handleHistoryKey(ev *tcell.EventKey, ticker *time.Ticker) (quit bool) {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyBackspace, tcell.KeyBackspace2:
		a.closeHistory()
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return true
		case 'g':
			a.closeHistory()
		case '+':
			a.setInterval(a.interval+intervalStep, ticker)
		case '-':
			a.setInterval(a.interval-intervalStep, ticker)
		}
	}
	return false
}

// drawHistory paints the CPU history of one process in the area of the
// process table: a bar per refresh, the newest on the right.
func (a *App) drawHistory(helpLine int) {
	width, _ := a.screen.Size()
	samples := a.history(a.historyOf)

	a.drawLine(headerLine, styleBar, fmt.Sprintf(
		" CPU history of %s: one bar per refresh, newest on the right", describe(a.historyOf)))

	// Below the graph there is one line for the time axis and one for the
	// numbers; the graph takes what is left.
	plotWidth := width - axisWidth
	plotHeight := helpLine - firstRowLine - 2
	if plotWidth < 1 || plotHeight < 1 {
		a.drawLine(firstRowLine, styleNormal, historyStats(samples))
		return
	}
	// The graph shows the newest samples that fit; the numbers below it
	// are about all the samples kept.
	visible := samples
	if len(visible) > plotWidth {
		visible = visible[len(visible)-plotWidth:]
	}

	// The scale adapts to the highest value on screen.
	scale := niceScale(slicesMax(visible))
	for i, row := range graphRows(visible, plotWidth, plotHeight, scale) {
		label := strings.Repeat(" ", axisWidth-1) + "│"
		switch i {
		case 0:
			label = fmt.Sprintf("%6.1f%%┤", scale)
		case plotHeight - 1:
			label = fmt.Sprintf("%6.1f%%┤", 0.0)
		}
		a.drawLine(firstRowLine+i, styleNormal, label+row)
	}
	a.drawLine(firstRowLine+plotHeight, styleNormal,
		strings.Repeat(" ", axisWidth-1)+"└"+strings.Repeat("─", plotWidth))
	a.drawLine(firstRowLine+plotHeight+1, styleNormal, historyStats(samples))
}

// historyStats summarizes all the samples kept for a process in one line.
func historyStats(samples []float64) string {
	if len(samples) == 0 {
		return "  waiting for the first measurement..."
	}
	var sum float64
	for _, v := range samples {
		sum += v
	}
	return fmt.Sprintf("  now %.1f%%   average %.1f%%   maximum %.1f%%   (last %d samples; %d are kept at most)",
		samples[len(samples)-1], sum/float64(len(samples)), slicesMax(samples),
		len(samples), monitor.HistoryLen)
}

// historyHelp is the bottom line of the history view.
func (a *App) historyHelp() string {
	return fmt.Sprintf(" CPU history of %s   Esc/g:back  +/-:[%.1fs]  q:quit",
		describe(a.historyOf), a.interval.Seconds())
}

// graphRows draws samples as vertical bars in a grid of width x height
// cells and returns its lines, top first. A value equal to scale fills the
// whole height. The last sample is in the last column; if there are fewer
// samples than columns, the left side stays empty.
func graphRows(samples []float64, width, height int, scale float64) []string {
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = []rune(strings.Repeat(" ", width))
	}
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	offset := width - len(samples)

	for i, value := range samples {
		// Each cell has 8 levels, so the bar has height*8 possible sizes.
		eighths := int(math.Round(value / scale * float64(height*8)))
		eighths = min(max(eighths, 0), height*8)
		if value > 0 && eighths == 0 {
			eighths = 1 // any activity is visible, however small
		}
		for fromBottom := 0; fromBottom < height; fromBottom++ {
			fill := min(max(eighths-fromBottom*8, 0), 8)
			grid[height-1-fromBottom][offset+i] = blocks[fill]
		}
	}

	lines := make([]string, height)
	for i, row := range grid {
		lines[i] = string(row)
	}
	return lines
}

// niceScale returns a round number, not lower than highest, to use as the
// top of the graph.
func niceScale(highest float64) float64 {
	for _, step := range []float64{1, 2, 5, 10, 25, 50, 100} {
		if highest <= step {
			return step
		}
	}
	return math.Ceil(highest/100) * 100
}

// slicesMax returns the highest value, or 0 if there are none.
func slicesMax(values []float64) float64 {
	var highest float64
	for _, v := range values {
		highest = max(highest, v)
	}
	return highest
}
