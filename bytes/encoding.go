package bytes

import (
	"errors"
)

// Encode takes an integer representing Remaining Length and encodes it according to the MQTT spec.
func Encode(x int) (encoded []byte) {
	for x > 0 {
		enc := x % 128
		x = x / 128
		if x > 0 {
			enc = enc | 128
		}
		encoded = append(encoded, byte(enc))
	}
	return
}

// Decode takes an encoded Remaining Length slice of bytes and decodes it into an integer.
// Returns an additional error if the decoding fails.
func Decode(stream []byte) (int, error) {
	mult := 1
	value := 0

	for _, encodedByte := range stream {
		value += int(encodedByte&127) * mult
		mult *= 128

		if mult > 128*128*128 {
			return 0, errors.New("malformed remaining length")
		}

		if (encodedByte & 128) == 0 {
			break
		}
	}
	return value, nil
}
