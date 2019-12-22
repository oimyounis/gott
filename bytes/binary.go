package bytes

import (
	"fmt"
	"strconv"
)

func ByteToBinaryString(b byte) string {
	return fmt.Sprintf("%08b", b)
}

func BinaryStringToByte(bs string) (byte, error) {
	i, err := strconv.ParseInt(bs, 2, 8)
	if err != nil {
		return 0, err
	}
	return byte(i), nil
}

func BitIsSet(b, bit byte) bool {
	return (1<<bit)&b != 0
}
