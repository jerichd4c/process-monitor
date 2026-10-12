package monitor

import (
	"os"
	"testing"

	"process-monitor/proc"
)

func TestAppendCapped(t *testing.T) {
	var values []float64
	for i := 1; i <= 5; i++ {
		values = appendCapped(values, float64(i), 3)
	}
	want := []float64{3, 4, 5} // the two oldest were dropped
	if len(values) != len(want) {
		t.Fatalf("got %v, want %v", values, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Errorf("got %v, want %v", values, want)
		}
	}
}

func TestSamplerHistory(t *testing.T) {
	self, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	var s Sampler

	if h := s.History(self); h != nil {
		t.Errorf("history before any reading = %v, want nil", h)
	}
	// The first reading has no %CPU yet, so it is not part of the history.
	if _, err := s.Sample(); err != nil {
		t.Fatal(err)
	}
	// It is empty, but not nil: the process is known to be alive.
	if h := s.History(self); h == nil || len(h) != 0 {
		t.Errorf("history after one reading = %#v, want empty and not nil", h)
	}
	for i := 0; i < 3; i++ {
		if _, err := s.Sample(); err != nil {
			t.Fatal(err)
		}
	}
	h := s.History(self)
	if len(h) != 3 {
		t.Fatalf("history after four readings has %d values, want 3", len(h))
	}

	// What we get is a copy: changing it must not change the sampler's data.
	h[0] = -1
	if again := s.History(self); again[0] == -1 {
		t.Error("History returned its internal slice instead of a copy")
	}

	// A process that does not exist (or ended) has no history.
	if h := s.History(proc.Process{PID: 4194000}); h != nil {
		t.Errorf("history of a missing process = %v, want nil", h)
	}
	// Same PID with another start time is another process.
	other := self
	other.StartTime++
	if h := s.History(other); h != nil {
		t.Errorf("history of a reused PID = %v, want nil", h)
	}
}
