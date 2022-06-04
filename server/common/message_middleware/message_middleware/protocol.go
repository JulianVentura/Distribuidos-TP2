package message_middleware

import "encoding/binary"

func encode32(number uint32, buffer []byte) uint {
	binary.BigEndian.PutUint32(buffer, number)
	return 4
}

func decode32(encoded []byte) (uint32, uint32) {
	return binary.BigEndian.Uint32(encoded), 4
}

func encodeString(str string, buffer []byte) uint {
	n := encode32(uint32(len(str)), buffer)
	encodedStr := []byte(str)
	copy(buffer[n:], encodedStr)

	return uint(len(str)) + n
}

func decodeString(encoded []byte) (string, uint32) {

	strLen, n := decode32(encoded)
	str := string(encoded[n : strLen+n])

	return str, strLen + n
}

func encodeStringSlice(slice []string) []byte {
	l := uint32(len(slice))
	sz := 0
	for _, str := range slice {
		sz += len(str) + 4
	}
	encodedStrings := make([]byte, sz+4)
	n := encode32(l, encodedStrings)
	for _, str := range slice {
		n += encodeString(str, encodedStrings[n:])
	}

	return encodedStrings
}

func decodeStringSlice(encoded []byte) []string {
	len, start := decode32(encoded)
	slice := make([]string, len)

	byteCount := start
	for i := uint32(0); i < len; i++ {
		str, n := decodeString(encoded[byteCount:])
		slice[i] = str
		byteCount += n
	}

	return slice
}
