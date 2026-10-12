package control

import (
	"errors"
	"fmt"
	"syscall"

	"process-monitor/proc"
)

// Range of the nice value: -20 is the highest priority, 19 the lowest.
const (
	MinNice = -20
	MaxNice = 19
)

// ErrNeedRoot is what a normal user gets when trying to raise a priority.
var ErrNeedRoot = errors.New("only root can raise a priority (lower the nice value)")

// SetNice changes the nice value of the target process with the
// setpriority system call.
//
// On Linux the nice value belongs to each thread, and setpriority changes
// only the thread it is given. To change the whole process we apply it to
// every thread listed in /proc/[pid]/task.
func SetNice(target proc.Process, nice int) error {
	if nice < MinNice || nice > MaxNice {
		return fmt.Errorf("nice must be between %d and %d", MinNice, MaxNice)
	}
	if err := checkTarget(target); err != nil {
		return err
	}

	tids, err := proc.ListThreadIDs(target.PID)
	if err != nil {
		return ErrGone
	}
	for _, tid := range tids {
		err := syscall.Setpriority(syscall.PRIO_PROCESS, tid, nice)
		switch {
		case errors.Is(err, syscall.ESRCH):
			continue // that thread ended in the meantime: nothing to change
		case errors.Is(err, syscall.EACCES):
			return ErrNeedRoot
		case errors.Is(err, syscall.EPERM):
			return ErrDenied
		case err != nil:
			return err
		}
	}
	return nil
}
