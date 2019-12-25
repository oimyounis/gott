package bytes

import (
	"fmt"
)

// ByteToBinaryString converts a byte to a bit string
func ByteToBinaryString(b byte) string {
	return fmt.Sprintf("%08b", b)
}

// BitIsSet checks whether a bit is set in a byte.
func BitIsSet(b, bit byte) bool {
	return (1<<bit)&b != 0
}
