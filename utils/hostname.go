package utils

import (
	"os"
)

func GetHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return ""
}
