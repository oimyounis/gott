package utils

func StringInSlice(str string, arr []string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

func StringsInSliceAny(strs []string, arr []string) bool {
	for _, str := range strs {
		for _, item := range arr {
			if item == str {
				return true
			}
		}
	}
	return false
}

func StringsInSliceAll(strs []string, arr []string) bool {
	matches := 0
	for _, str := range strs {
		for _, item := range arr {
			if item == str {
				matches++
			}
		}
	}
	return len(arr) == matches
}

func StringsInSliceAnyMatch(strs []string, arr []string) (bool, []string) {
	matches := []string{}
	for _, str := range strs {
		for _, item := range arr {
			if item == str {
				matches = append(matches, str)
			}
		}
	}
	return len(matches) != 0, matches
}

func StringInInterfaceSlice(str string, arr []interface{}) bool {
	for _, item := range arr {
		if conv, ok := item.(string); ok {
			if conv == str {
				return true
			}
		}
	}
	return false
}

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
