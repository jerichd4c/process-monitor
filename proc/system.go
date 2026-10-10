package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CPUTimes holds the CPU time counters of the whole machine, in ticks,
// summed over all CPUs since boot.
type CPUTimes struct {
	Total uint64 // all the time, busy or not
	Idle  uint64 // time doing nothing (idle + iowait)
}

// ReadCPUTimes reads the first line of /proc/stat.
func ReadCPUTimes() (CPUTimes, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return CPUTimes{}, err
	}
	return parseCPUTimes(string(data))
}

// parseCPUTimes parses a line like
// "cpu  46396 0 23260 3969864 1847 0 3607 0 0 0", whose columns are:
// user nice system idle iowait irq softirq steal guest guest_nice.
// guest and guest_nice are already counted inside user and nice, so we only
// add the first eight columns.
func parseCPUTimes(data string) (CPUTimes, error) {
	line, _, _ := strings.Cut(data, "\n")
	fields := strings.Fields(line)
	if len(fields) < 9 || fields[0] != "cpu" {
		return CPUTimes{}, fmt.Errorf("/proc/stat: unexpected first line %q", line)
	}
	var values [8]uint64
	for i := range values {
		v, err := strconv.ParseUint(fields[i+1], 10, 64)
		if err != nil {
			return CPUTimes{}, fmt.Errorf("/proc/stat: column %d: %w", i+1, err)
		}
		values[i] = v
	}

	var t CPUTimes
	for _, v := range values {
		t.Total += v
	}
	t.Idle = values[3] + values[4] // idle + iowait
	return t, nil
}

// LoadAvg is the average number of tasks running or waiting to run over the
// last 1, 5 and 15 minutes. The kernel computes it; we read it as is.
type LoadAvg struct {
	One, Five, Fifteen float64
}

// ReadLoadAvg reads /proc/loadavg.
func ReadLoadAvg() (LoadAvg, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return LoadAvg{}, err
	}
	return parseLoadAvg(string(data))
}

// parseLoadAvg parses a line like "0.03 0.09 0.32 1/459 15125".
func parseLoadAvg(data string) (LoadAvg, error) {
	fields := strings.Fields(data)
	if len(fields) < 3 {
		return LoadAvg{}, fmt.Errorf("/proc/loadavg: unexpected content %q", data)
	}
	var values [3]float64
	for i := range values {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return LoadAvg{}, fmt.Errorf("/proc/loadavg: %w", err)
		}
		values[i] = v
	}
	return LoadAvg{One: values[0], Five: values[1], Fifteen: values[2]}, nil
}
