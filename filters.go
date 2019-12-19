package gott

import (
	"bytes"
)

func FilterInSlice(filter []byte, filters []*TopicLevel) bool {
	for _, item := range filters {
		if bytes.Equal(item.Bytes, filter) {
			return true
		}
	}
	return false
}
