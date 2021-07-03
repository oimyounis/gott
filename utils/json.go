package utils

import json "github.com/json-iterator/go"

func ToJSONBytes(o interface{}, defaultVal string) []byte {
	js, err := json.Marshal(o)
	if err != nil {
		return []byte(defaultVal)
	}
	return js
}
