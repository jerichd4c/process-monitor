package proc

import "testing"

func TestParseCPUTimes(t *testing.T) {
	// Real first line from /proc/stat, followed by a per-CPU line we ignore.
	data := "cpu  46396 0 23260 3969864 1847 0 3607 0 0 0\ncpu0 100 0 100 100 0 0 0 0 0 0\n"
	got, err := parseCPUTimes(data)
	if err != nil {
		t.Fatal(err)
	}
	want := CPUTimes{
		Total: 46396 + 23260 + 3969864 + 1847 + 3607,
		Idle:  3969864 + 1847,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseCPUTimesBad(t *testing.T) {
	for _, data := range []string{"", "intr 1 2 3", "cpu 1 2 x 4 5 6 7 8"} {
		if _, err := parseCPUTimes(data); err == nil {
			t.Errorf("parseCPUTimes(%q) did not fail", data)
		}
	}
}

func TestParseLoadAvg(t *testing.T) {
	got, err := parseLoadAvg("0.03 0.09 0.32 1/459 15125\n")
	if err != nil {
		t.Fatal(err)
	}
	want := LoadAvg{One: 0.03, Five: 0.09, Fifteen: 0.32}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
