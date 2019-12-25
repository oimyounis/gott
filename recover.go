package gott

import (
	"fmt"
	"runtime/debug"
)

func Recover(callback func(err, stack string)) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())

		if callback != nil {
			callback(fmt.Sprintf("%s", r), stack)
		}
	}
}
