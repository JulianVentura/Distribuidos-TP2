package message_middleware

import "encoding/binary"

func encode32(number uint32, buffer []byte) uint {
	binary.BigEndian.PutUint32(buffer, number)
	return 4
}

func decode32(encoded []byte) (uint32, uint32) {
	return binary.BigEndian.Uint32(encoded), 4
}

func encode_string(str string, buffer []byte) uint {
	n := encode32(uint32(len(str)), buffer)
	encoded_str := []byte(str)
	copy(buffer[n:], encoded_str)

	return uint(len(str)) + n
}

func decode_string(encoded []byte) (string, uint32) {

	str_len, n := decode32(encoded)
	str := string(encoded[n : str_len+n])

	return str, str_len + n
}

func encode_string_slice(slice []string) []byte {
	l := uint32(len(slice))
	sz := 0
	for _, str := range slice {
		sz += len(str) + 4
	}
	encoded_strings := make([]byte, sz+4)
	n := encode32(l, encoded_strings)
	for _, str := range slice {
		n += encode_string(str, encoded_strings[n:])
	}

	return encoded_strings
}

func decode_string_slice(encoded []byte) []string {
	len, start := decode32(encoded)
	slice := make([]string, len)

	byte_count := start
	for i := uint32(0); i < len; i++ {
		str, n := decode_string(encoded[byte_count:])
		slice[i] = str
		byte_count += n
	}

	return slice
}
