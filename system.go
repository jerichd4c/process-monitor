package main

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

// readMemInfo parses /proc/meminfo, whose lines look like
// "MemTotal:        7017052 kB".
func readMemInfo() (MemInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemInfo{}, err
	}
	var m MemInfo
	for _, line := range strings.Split(string(data), "\n") {
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

// loadUsers maps user IDs to names by reading /etc/passwd, whose lines look
// like "sebas:x:1000:1000::/home/sebas:/bin/bash" (name:password:UID:...).
func loadUsers() map[int]string {
	users := make(map[int]string)
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return users // empty: we will show numeric IDs instead
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 3 {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		users[uid] = fields[0]
	}
	return users
}

// userName returns the name for uid, or the number itself if unknown.
func userName(users map[int]string, uid int) string {
	if name, ok := users[uid]; ok {
		return name
	}
	return strconv.Itoa(uid)
}
