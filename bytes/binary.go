package bytes

import (
	"fmt"
)

func ByteToBinaryString(b byte) string {
	return fmt.Sprintf("%08b", b)
}

func BitIsSet(b, bit byte) bool {
	return (1<<bit)&b != 0
}
