package gott

import (
	"fmt"
	"runtime/debug"
)

// Recover calls built-in recover function.
// Can call a provided callback function passing in the error reported and the stack of it.
func Recover(callback func(err, stack string)) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())

		if callback != nil {
			callback(fmt.Sprintf("%s", r), stack)
		}
	}
}
