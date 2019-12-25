package utils

func ByteInSlice(b byte, arr []byte) bool {
	for _, item := range arr {
		if item == b {
			return true
		}
	}
	return false
}

func IndexAllByte(b []byte, c byte) (indices []int) {
	for i, x := range b {
		if x == c {
			indices = append(indices, i)
		}
	}
	return
}
