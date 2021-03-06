package utils

// ByteInSlice checks whether a byte is in a slice of bytes.
func ByteInSlice(b byte, arr []byte) bool {
	for _, item := range arr {
		if item == b {
			return true
		}
	}
	return false
}

// IndexAllByte returns all occurrences of a byte in a slice of bytes.
func IndexAllByte(b []byte, c byte) (indices []int) {
	for i, x := range b {
		if x == c {
			indices = append(indices, i)
		}
	}
	return
}

// StringInSlice checks whether a string is in a slice of strings.
func StringInSlice(s string, sl []string) bool {
	for _, o := range sl {
		if o == s {
			return true
		}
	}
	return false
}
