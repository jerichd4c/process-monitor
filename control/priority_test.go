package control

import (
	"errors"
	"os"
	"testing"

	"process-monitor/proc"
)

// self returns the test process itself as read from /proc.
func self(t *testing.T) proc.Process {
	t.Helper()
	p, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSetNiceRange(t *testing.T) {
	for _, nice := range []int{-21, 20, 100} {
		if err := SetNice(self(t), nice); err == nil {
			t.Errorf("SetNice(%d) did not fail", nice)
		}
	}
}

func TestSetNiceSameValue(t *testing.T) {
	// Setting the value it already has is always allowed and changes nothing.
	me := self(t)
	if err := SetNice(me, me.Nice); err != nil {
		t.Fatalf("SetNice(%d) = %v, want nil", me.Nice, err)
	}
	if after := self(t); after.Nice != me.Nice {
		t.Errorf("nice changed from %d to %d", me.Nice, after.Nice)
	}
}

func TestSetNiceLowerPriority(t *testing.T) {
	// Any user may lower the priority (raise the nice) of its own process.
	// Only this short-lived test process is affected.
	me := self(t)
	if me.Nice >= MaxNice {
		t.Skip("already at the lowest priority")
	}
	if err := SetNice(me, me.Nice+1); err != nil {
		t.Fatalf("SetNice(%d) = %v, want nil", me.Nice+1, err)
	}
	if after := self(t); after.Nice != me.Nice+1 {
		t.Errorf("nice = %d, want %d", after.Nice, me.Nice+1)
	}

	// Raising it back needs root.
	if os.Geteuid() != 0 {
		if err := SetNice(self(t), me.Nice); !errors.Is(err, ErrNeedRoot) {
			t.Errorf("raising priority as a normal user = %v, want ErrNeedRoot", err)
		}
	}
}

func TestSetNiceOtherUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root may change any process")
	}
	first, err := proc.Read(1) // PID 1 belongs to root
	if err != nil {
		t.Fatal(err)
	}
	if err := SetNice(first, first.Nice); !errors.Is(err, ErrDenied) {
		t.Errorf("SetNice(PID 1) = %v, want ErrDenied", err)
	}
}

func TestSetNiceGone(t *testing.T) {
	if err := SetNice(proc.Process{PID: 4194000}, 0); !errors.Is(err, ErrGone) {
		t.Errorf("missing process: %v, want ErrGone", err)
	}
	if err := SetNice(proc.Process{PID: -1}, 0); !errors.Is(err, ErrGone) {
		t.Errorf("PID -1: %v, want ErrGone", err)
	}
}
