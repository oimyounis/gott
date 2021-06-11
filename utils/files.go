package utils

import (
	"os"
)

// PathExists checks whether a path exists in the filesystem
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
