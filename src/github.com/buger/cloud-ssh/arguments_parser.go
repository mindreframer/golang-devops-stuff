package main

import (
	"strings"
)

func splitHostname(str string) (user string, hostname string) {
	if arr := strings.Split(str, "@"); len(arr) > 1 {
		return arr[0], arr[1]
	} else {
		return "", str
	}
}

func joinHostname(user string, hostname string) string {
	if user != "" {
		return user + "@" + hostname
	} else {
		return hostname
	}
}

// Go though arguments, and find one with host
func getTargetHostname(args []string) (user string, hostname string, arg_idx int) {
	for idx, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			if idx == 0 {
				hostname = arg
				arg_idx = idx
				break
			} else {
				if !strings.HasPrefix(args[idx-1], "-") {
					hostname = arg
					arg_idx = idx
					break
				}
			}
		}
	}

	user, hostname = splitHostname(hostname)

	return
}
