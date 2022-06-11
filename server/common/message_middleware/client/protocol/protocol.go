package protocol

import "encoding/binary"

/*
This library contains code which is similar to ./common/protocol/protocol.go (root)
Instead of using that library a new implementation was choosen, because
it has a slightly different API, it is more efficient, and more important, can be changed independently
*/

func encode32(number uint32, buffer []byte) uint {
	binary.BigEndian.PutUint32(buffer, number)
	return 4
}

func decode32(encoded []byte) (uint32, uint32) {
	return binary.BigEndian.Uint32(encoded), 4
}

func encodeSlice(slice []byte, buffer []byte) uint {
	n := encode32(uint32(len(slice)), buffer)
	copy(buffer[n:], slice)

	return uint(len(slice)) + n
}

func decodeSlice(encoded []byte) ([]byte, uint32) {

	l, n := decode32(encoded)
	slice := encoded[n : l+n]

	return slice, l + n
}

func EncodeBytesSlice(slice [][]byte) []byte {
	l := uint32(len(slice))
	sz := 0
	for _, a := range slice {
		sz += len(a) + 4
	}
	encoded := make([]byte, sz+4)
	n := encode32(l, encoded)
	for _, a := range slice {
		n += encodeSlice(a, encoded[n:])
	}

	return encoded
}

func DecodeBytesSlice(encoded []byte) [][]byte {
	len, start := decode32(encoded)
	slice := make([][]byte, len)

	byteCount := start
	for i := uint32(0); i < len; i++ {
		a, n := decodeSlice(encoded[byteCount:])
		slice[i] = a
		byteCount += n
	}

	return slice
}
