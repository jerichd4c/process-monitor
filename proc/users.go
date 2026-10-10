package proc

import (
	"os"
	"strconv"
	"strings"
)

// Users maps user IDs to user names.
type Users map[int]string

// LoadUsers reads /etc/passwd. If it cannot be read the map is empty and
// Name falls back to numeric IDs.
func LoadUsers() Users {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return Users{}
	}
	return parsePasswd(string(data))
}

// parsePasswd parses lines like "sebas:x:1000:1000::/home/sebas:/bin/bash"
// (name:password:UID:...).
func parsePasswd(data string) Users {
	users := Users{}
	for _, line := range strings.Split(data, "\n") {
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

// Name returns the name for uid, or the number itself if unknown.
func (u Users) Name(uid int) string {
	if name, ok := u[uid]; ok {
		return name
	}
	return strconv.Itoa(uid)
}
