package monitor

import (
	"testing"

	"process-monitor/proc"
)

// node builds a fake row with only what the tree needs.
func node(pid, ppid int, name string) Row {
	return Row{Process: proc.Process{PID: pid, PPID: ppid, Comm: name, Cmdline: name}}
}

// render returns the tree as text lines, the way the table shows it.
func render(rows []Row) []string {
	var lines []string
	for _, r := range rows {
		lines = append(lines, r.TreePrefix+r.Cmdline)
	}
	return lines
}

func sameLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTree(t *testing.T) {
	// Given out of order on purpose: the tree must not depend on it.
	rows := []Row{
		node(40, 20, "sleep"),
		node(20, 1, "bash"),
		node(1, 0, "systemd"),
		node(30, 20, "yes"),
		node(10, 1, "cron"),
		node(50, 1, "sshd"),
		node(60, 50, "sshd-session"),
		node(70, 60, "bash"),
	}
	sameLines(t, render(Tree(rows)), []string{
		"systemd",
		"├─ cron",
		"├─ bash",
		"│  ├─ yes",
		"│  └─ sleep",
		"└─ sshd",
		"   └─ sshd-session",
		"      └─ bash",
	})
	if len(rows) != 8 || rows[0].TreePrefix != "" {
		t.Error("Tree must not modify the rows it receives")
	}
}

func TestTreeOrphansBecomeRoots(t *testing.T) {
	// The parents 500 and 600 are not in the list (ended, or hidden by the
	// filter), so their children start their own trees.
	rows := []Row{
		node(1, 0, "systemd"),
		node(700, 600, "orphan-b"),
		node(510, 500, "orphan-a"),
		node(511, 510, "child"),
	}
	sameLines(t, render(Tree(rows)), []string{
		"systemd",
		"orphan-a",
		"└─ child",
		"orphan-b",
	})
}

func TestTreeNeverLosesProcesses(t *testing.T) {
	// Impossible in a real system, but the tree must survive it: 5 and 6
	// are each other's parent, and 7 is its own parent.
	rows := []Row{node(5, 6, "a"), node(6, 5, "b"), node(7, 7, "c"), node(1, 0, "init")}
	got := Tree(rows)
	if len(got) != len(rows) {
		t.Fatalf("got %d rows, want %d: %q", len(got), len(rows), render(got))
	}
	seen := map[int]bool{}
	for _, r := range got {
		seen[r.PID] = true
	}
	if len(seen) != len(rows) {
		t.Errorf("a process is repeated or missing: %q", render(got))
	}
}

func TestTreeEmpty(t *testing.T) {
	if got := Tree(nil); len(got) != 0 {
		t.Errorf("Tree(nil) = %v, want empty", got)
	}
}
