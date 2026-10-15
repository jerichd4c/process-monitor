// Package control acts on processes by sending them signals with the kill
// system call. It never runs external programs.
package control

import (
	"errors"
	"os"
	"syscall"

	"process-monitor/proc"
)

// Action is something the user can do to a process.
type Action int

const (
	Terminate Action = iota // ask the process to end (it may clean up)
	Kill                    // end it by force (it cannot refuse)
	Suspend                 // stop it until it is resumed
	Resume                  // let a stopped process continue
)

// Signal returns the signal that carries out the action.
func (a Action) Signal() syscall.Signal {
	switch a {
	case Terminate:
		return syscall.SIGTERM
	case Kill:
		return syscall.SIGKILL
	case Suspend:
		return syscall.SIGSTOP
	default:
		return syscall.SIGCONT
	}
}

// String returns the name of the action for the messages.
func (a Action) String() string {
	switch a {
	case Terminate:
		return "terminate"
	case Kill:
		return "force kill"
	case Suspend:
		return "suspend"
	default:
		return "resume"
	}
}

// SignalName returns the name of the signal of the action.
func (a Action) SignalName() string {
	switch a {
	case Terminate:
		return "SIGTERM"
	case Kill:
		return "SIGKILL"
	case Suspend:
		return "SIGSTOP"
	default:
		return "SIGCONT"
	}
}

// Errors with a clear meaning for the user.
var (
	ErrGone   = errors.New("the process no longer exists")
	ErrDenied = errors.New("permission denied (it belongs to another user)")
	ErrSelf   = errors.New("that is the monitor itself (press q to quit)")
)

// Apply sends the signal of the action to the target process.
func Apply(target proc.Process, action Action) error {
	// Killing or stopping ourselves would leave the terminal unusable.
	if target.PID == os.Getpid() {
		return ErrSelf
	}
	if err := checkTarget(target); err != nil {
		return err
	}

	err := syscall.Kill(target.PID, action.Signal())
	switch {
	case errors.Is(err, syscall.ESRCH):
		return ErrGone
	case errors.Is(err, syscall.EPERM):
		return ErrDenied
	}
	return err
}

// checkTarget makes sure target is still the process the user chose.
func checkTarget(target proc.Process) error {
	// PID 0 and negative PIDs do not name a process but whole groups of
	// them (kill(-1) means "every process I can reach"): never allow them,
	// whatever the caller passes.
	if target.PID <= 0 {
		return ErrGone
	}
	// The PID may have been reused by a new process since the user chose
	// it. The start time tells us whether it is still the same one.
	current, err := proc.Read(target.PID)
	if errors.Is(err, proc.ErrNoAccess) {
		return proc.ErrNoAccess // it exists, but we may not look at it
	}
	if err != nil || current.StartTime != target.StartTime {
		return ErrGone
	}
	return nil
}
