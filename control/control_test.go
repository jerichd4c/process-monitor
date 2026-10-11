package control

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"process-monitor/proc"
)

func TestSignals(t *testing.T) {
	cases := map[Action]syscall.Signal{
		Terminate: syscall.SIGTERM,
		Kill:      syscall.SIGKILL,
		Suspend:   syscall.SIGSTOP,
		Resume:    syscall.SIGCONT,
	}
	for action, want := range cases {
		if got := action.Signal(); got != want {
			t.Errorf("%v sends %v, want %v", action, got, want)
		}
	}
}

func TestApplyRefusesGroups(t *testing.T) {
	// PID 0 and negative PIDs would signal many processes at once.
	for _, pid := range []int{0, -1, -1234} {
		if err := Apply(proc.Process{PID: pid}, Resume); !errors.Is(err, ErrGone) {
			t.Errorf("Apply(PID %d) = %v, want ErrGone", pid, err)
		}
	}
}

func TestApplyRefusesItself(t *testing.T) {
	self, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(self, Kill); !errors.Is(err, ErrSelf) {
		t.Errorf("Apply(self) = %v, want ErrSelf", err)
	}
}

func TestApplyGone(t *testing.T) {
	// No process has this PID (the kernel maximum is 4194304).
	if err := Apply(proc.Process{PID: 4194000}, Terminate); !errors.Is(err, ErrGone) {
		t.Errorf("missing process: %v, want ErrGone", err)
	}

	// Our parent exists, but with another start time it counts as a
	// different process (a reused PID): the signal must not be sent.
	parent, err := proc.Read(os.Getppid())
	if err != nil {
		t.Fatal(err)
	}
	parent.StartTime++
	if err := Apply(parent, Resume); !errors.Is(err, ErrGone) {
		t.Errorf("reused PID: %v, want ErrGone", err)
	}
}

func TestApplyDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root may signal any process")
	}
	first, err := proc.Read(1) // PID 1 belongs to root
	if err != nil {
		t.Fatal(err)
	}
	// SIGCONT is harmless even if it were delivered.
	if err := Apply(first, Resume); !errors.Is(err, ErrDenied) {
		t.Errorf("Apply(PID 1) = %v, want ErrDenied", err)
	}
}

func TestApplyAllowed(t *testing.T) {
	// SIGCONT to our own parent (the "go test" process) is harmless: it is
	// ignored by a process that is not stopped.
	parent, err := proc.Read(os.Getppid())
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(parent, Resume); err != nil {
		t.Errorf("Apply(parent, Resume) = %v, want nil", err)
	}
}
