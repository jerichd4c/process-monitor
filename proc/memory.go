package proc

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// MemInfo holds the system memory figures we need, in kB.
type MemInfo struct {
	Total     uint64
	Available uint64
}

// Used is the memory in use: everything that is not available.
func (m MemInfo) Used() uint64 {
	return m.Total - m.Available
}

// ReadMemInfo reads /proc/meminfo.
func ReadMemInfo() (MemInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemInfo{}, err
	}
	return parseMemInfo(string(data))
}

// parseMemInfo parses lines like "MemTotal:        7017052 kB".
func parseMemInfo(data string) (MemInfo, error) {
	var m MemInfo
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			m.Total = value
		case "MemAvailable:":
			m.Available = value
		}
	}
	if m.Total == 0 {
		return MemInfo{}, errors.New("/proc/meminfo: MemTotal not found")
	}
	return m, nil
}
