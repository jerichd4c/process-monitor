package proc

import "testing"

func TestPrintable(t *testing.T) {
	cases := map[string]string{
		"sleep 1000":         "sleep 1000",     // normal text is untouched
		"café ñandú 日本語":     "café ñandú 日本語", // so are accents and other alphabets
		"evil\nname":         "evil?name",      // a new line would break the table
		"tab\there":          "tab?here",       // a tab would move the columns
		"\x1b[2J\x1b[31mred": "?[2J?[31mred",   // escape sequences could clear the screen
		"bell\x07 back\x08":  "bell? back?",    // other control characters
		"bad\xff\xfebytes":   "bad??bytes",     // bytes that are not valid UTF-8
		"":                   "",
	}
	for input, want := range cases {
		if got := printable(input); got != want {
			t.Errorf("printable(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseCmdlineHostile(t *testing.T) {
	// The arguments of a process are whatever it wants them to be.
	got := parseCmdline("prog\x00--msg=\x1b[2Jhello\nworld\x00")
	if want := "prog --msg=?[2Jhello?world"; got != want {
		t.Errorf("parseCmdline = %q, want %q", got, want)
	}
}

func TestParseStatHostileName(t *testing.T) {
	line := "77 (a\nb\x1b[1m) S 1 77 77 0 -1 0 0 0 0 0 5 6 0 0 20 0 1 0 100 0 0"
	p, err := parseStat(77, line)
	if err != nil {
		t.Fatal(err)
	}
	if p.Comm != "a?b?[1m" || p.PPID != 1 || p.UTime != 5 {
		t.Errorf("Comm = %q, PPID = %d, UTime = %d", p.Comm, p.PPID, p.UTime)
	}
}
