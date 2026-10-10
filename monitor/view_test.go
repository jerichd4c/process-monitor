package monitor

import (
	"testing"

	"process-monitor/proc"
)

// row builds a fake table row for the tests.
func row(pid int, cpu float64, rssKB uint64, cmd string) Row {
	return Row{
		Process:    proc.Process{PID: pid, Comm: cmd, Cmdline: "/usr/bin/" + cmd, RSSKB: rssKB},
		CPUPercent: cpu,
	}
}

// pids returns the PIDs of rows in order, to compare them easily.
func pids(rows []Row) []int {
	var out []int
	for _, r := range rows {
		out = append(out, r.PID)
	}
	return out
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSort(t *testing.T) {
	rows := []Row{
		row(30, 0.0, 500, "bash"),
		row(10, 6.2, 100, "yes"),
		row(20, 0.0, 900, "firefox"),
		row(40, 1.5, 100, "go"),
	}
	cases := []struct {
		key  SortKey
		want []int
	}{
		{SortByCPU, []int{10, 40, 20, 30}}, // ties (0.0) ordered by PID
		{SortByMem, []int{20, 30, 10, 40}}, // ties (100 kB) ordered by PID
		{SortByPID, []int{10, 20, 30, 40}},
	}
	for _, c := range cases {
		Sort(rows, c.key)
		if got := pids(rows); !equal(got, c.want) {
			t.Errorf("sort by %v = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestFilter(t *testing.T) {
	rows := []Row{row(1, 0, 0, "bash"), row(2, 0, 0, "Firefox"), row(3, 0, 0, "yes")}

	if got := pids(Filter(rows, "")); !equal(got, []int{1, 2, 3}) {
		t.Errorf("empty filter = %v, want all", got)
	}
	if got := pids(Filter(rows, "FIRE")); !equal(got, []int{2}) {
		t.Errorf("filter FIRE = %v, want [2]", got)
	}
	// "/usr/bin" is only in the command line, not in the short name.
	if got := pids(Filter(rows, "usr/bin/y")); !equal(got, []int{3}) {
		t.Errorf("filter by cmdline = %v, want [3]", got)
	}
	if got := Filter(rows, "nothing-matches"); len(got) != 0 {
		t.Errorf("filter with no match = %v, want empty", pids(got))
	}
}

func TestCountState(t *testing.T) {
	var sum Summary
	for _, state := range []string{"R", "S", "S", "D", "I", "T", "t", "Z"} {
		countState(&sum, state)
	}
	want := Summary{Total: 8, Running: 1, Sleeping: 4, Stopped: 2, Zombie: 1}
	if sum != want {
		t.Errorf("got %+v, want %+v", sum, want)
	}
}
