package proc

import "testing"

// A real line of /proc/[pid]/stat of a program named "x) y": the name has
// a space and a ")" inside, which is the tricky case for the parser.
const weirdStat = "1094 (x) y) S 495 1094 495 34816 1094 4194304 300 0 0 0 " +
	"56 44 0 0 20 0 1 0 242977 16502784 1342 18446744073709551615"

func TestParseStatWeirdName(t *testing.T) {
	p, err := parseStat(1094, weirdStat)
	if err != nil {
		t.Fatal(err)
	}
	if p.Comm != "x) y" {
		t.Errorf("Comm = %q, want %q", p.Comm, "x) y")
	}
	if p.State != "S" || p.PPID != 495 {
		t.Errorf("State, PPID = %s, %d; want S, 495", p.State, p.PPID)
	}
	if p.UTime != 56 || p.STime != 44 {
		t.Errorf("UTime, STime = %d, %d; want 56, 44", p.UTime, p.STime)
	}
	if p.Priority != 20 || p.Nice != 0 || p.NumThreads != 1 {
		t.Errorf("Priority, Nice, NumThreads = %d, %d, %d; want 20, 0, 1",
			p.Priority, p.Nice, p.NumThreads)
	}
	if p.StartTime != 242977 {
		t.Errorf("StartTime = %d, want 242977", p.StartTime)
	}
}

func TestParseStatMalformed(t *testing.T) {
	for _, line := range []string{"", "123 no parentheses", "1 (a) S 2"} {
		if _, err := parseStat(1, line); err == nil {
			t.Errorf("parseStat(%q) did not fail", line)
		}
	}
}

func TestParseStatus(t *testing.T) {
	status := "Name:\tsleep\nState:\tS (sleeping)\n" +
		"Uid:\t1000\t0\t0\t0\nVmRSS:\t    7876 kB\nThreads:\t1\n"
	uid, rss, err := parseStatus(status)
	if err != nil {
		t.Fatal(err)
	}
	// The effective UID (second number) is the one we keep.
	if uid != 0 || rss != 7876 {
		t.Errorf("uid, rss = %d, %d; want 0, 7876", uid, rss)
	}
}

func TestParseStatusZombie(t *testing.T) {
	// Zombies have no VmRSS line: memory must be 0, not an error.
	uid, rss, err := parseStatus("Name:\tperl\nState:\tZ (zombie)\nUid:\t1000\t1000\t1000\t1000\n")
	if err != nil {
		t.Fatal(err)
	}
	if uid != 1000 || rss != 0 {
		t.Errorf("uid, rss = %d, %d; want 1000, 0", uid, rss)
	}
}

func TestParseCmdline(t *testing.T) {
	got := parseCmdline("sleep\x001000\x00")
	if got != "sleep 1000" {
		t.Errorf("parseCmdline = %q, want %q", got, "sleep 1000")
	}
	if got := parseCmdline(""); got != "" {
		t.Errorf("empty cmdline = %q, want empty", got)
	}
}

func TestParseMemInfo(t *testing.T) {
	m, err := parseMemInfo("MemTotal:        7017052 kB\nMemFree:         5338800 kB\nMemAvailable:    5547648 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if m.Total != 7017052 || m.Available != 5547648 || m.Used() != 1469404 {
		t.Errorf("got %+v (used %d)", m, m.Used())
	}
}

func TestUsersName(t *testing.T) {
	u := parsePasswd("root:x:0:0:root:/root:/bin/bash\nsebas:x:1000:1000::/home/sebas:/bin/bash\n")
	if u.Name(1000) != "sebas" || u.Name(0) != "root" {
		t.Errorf("names = %q, %q", u.Name(1000), u.Name(0))
	}
	if u.Name(4242) != "4242" {
		t.Errorf("unknown uid = %q, want %q", u.Name(4242), "4242")
	}
}
