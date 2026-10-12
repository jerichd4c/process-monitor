package monitor

import (
	"errors"
	"os"
	"testing"

	"process-monitor/proc"
)

func TestThreadSampler(t *testing.T) {
	self, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	var s ThreadSampler

	first, err := s.Sample(self)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) < 2 {
		t.Fatalf("got %d threads, want several (Go programs are multi-threaded)", len(first))
	}
	for i, row := range first {
		if row.CPUPercent != 0 {
			t.Errorf("thread %d: %%CPU = %v on the first reading, want 0", row.TID, row.CPUPercent)
		}
		if i > 0 && first[i-1].TID >= row.TID {
			t.Errorf("threads are not sorted by TID: %d before %d", first[i-1].TID, row.TID)
		}
	}

	// Burn some CPU so that at least one thread has something to show.
	x := 0
	for i := 0; i < 300_000_000; i++ {
		x += i
	}
	_ = x

	second, err := s.Sample(self)
	if err != nil {
		t.Fatal(err)
	}
	var total float64
	for _, row := range second {
		if row.CPUPercent < 0 || row.CPUPercent > 100 {
			t.Errorf("thread %d: %%CPU = %v, out of range", row.TID, row.CPUPercent)
		}
		total += row.CPUPercent
	}
	if total <= 0 {
		t.Errorf("total %%CPU of the threads = %v, want more than 0 after working", total)
	}
}

func TestThreadSamplerGone(t *testing.T) {
	var s ThreadSampler
	if _, err := s.Sample(proc.Process{PID: 4194000}); !errors.Is(err, ErrProcessGone) {
		t.Errorf("missing process: %v, want ErrProcessGone", err)
	}

	// Same PID but another start time: a different process.
	self, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	self.StartTime++
	if _, err := s.Sample(self); !errors.Is(err, ErrProcessGone) {
		t.Errorf("reused PID: %v, want ErrProcessGone", err)
	}
}
