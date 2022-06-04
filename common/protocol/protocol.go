package protocol

import (
	"distribuidos/tp2/common/socket"
	"encoding/binary"
	"math"
	"time"
)

func encodeF64(number float64) []byte {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, math.Float64bits(number))

	return buffer
}

func encode64(number uint64) []byte {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, number)

	return buffer
}

func encode32(number uint32) []byte {
	buffer := make([]byte, 4)
	binary.BigEndian.PutUint32(buffer, number)

	return buffer
}

func encode16(number uint16) []byte {
	buffer := make([]byte, 2)
	binary.BigEndian.PutUint16(buffer, number)

	return buffer
}

func encode8(number uint8) []byte {
	//With one byte the endianess interpretation are all the same
	buffer := make([]byte, 1)
	buffer[0] = byte(number)
	return buffer
}

func encodeString(str string) []byte {
	encodedSize := encode32(uint32(len(str)))
	//UTF-8 Encoding
	encodedStr := []byte(str)
	return append(encodedSize, encodedStr...)
}

func encodeF64slice(slice []float64) []byte {
	floatSize := 8
	l := len(slice)
	buffer := make([]byte, l*floatSize) //Buffer for encoded slice
	len := encode32(uint32(l))

	for i := 0; i < l; i++ {
		slcIdx := floatSize * i
		binary.BigEndian.PutUint64(buffer[slcIdx:], math.Float64bits(slice[i]))
	}

	return append(len, buffer...)
}

func encodeByteSlice(slice []byte) []byte {
	encodedSize := encode32(uint32(len(slice)))
	return append(encodedSize, slice...)
}

func decodeF64slice(encoded []byte) ([]float64, uint32) {
	floatSize := uint32(8)
	len, start := decode32(encoded)
	slice := make([]float64, len)

	for i := uint32(0); i < len; i++ {
		slcIdx := floatSize*i + start
		decoded := binary.BigEndian.Uint64(encoded[slcIdx:])
		slice[i] = math.Float64frombits(decoded)
	}

	return slice, len*floatSize + start
}

func encodeStringSlice(slice []string) []byte {
	l := uint32(len(slice))
	encodedStrings := make([][]byte, l+1)
	encodedStrings[0] = encode32(l)

	for i, str := range slice {
		encodedStrings[i+1] = encodeString(str)
	}

	return appendSlices(encodedStrings)
}

func decodeStringSlice(encoded []byte) ([]string, uint32) {
	len, start := decode32(encoded)
	slice := make([]string, len)

	byteCount := start
	for i := uint32(0); i < len; i++ {
		str, n := decodeString(encoded[byteCount:])
		slice[i] = str
		byteCount += n
	}

	return slice, byteCount
}

func decodeByteSlice(encoded []byte) ([]byte, uint32) {

	sLen, n := decode32(encoded)
	slice := encoded[n : sLen+n]

	return slice, sLen + n
}

func decodeF64(encoded []byte) (float64, uint32) {
	decoded := binary.BigEndian.Uint64(encoded)

	return math.Float64frombits(decoded), 8
}

func decode64(encoded []byte) (uint64, uint32) {
	return binary.BigEndian.Uint64(encoded), 8
}

func decode32(encoded []byte) (uint32, uint32) {
	return binary.BigEndian.Uint32(encoded), 4
}

func decode16(encoded []byte) (uint16, uint32) {
	return binary.BigEndian.Uint16(encoded), 2
}

func decode8(encoded []byte) (uint8, uint32) {
	return uint8(encoded[0]), 1
}

func decodeString(encoded []byte) (string, uint32) {

	strLen, n := decode32(encoded)
	str := string(encoded[n : strLen+n])

	return str, strLen + n
}

func Send(socket *socket.TCPConnection, message Encodable) error {
	encodedMsg := message.encode()
	size := uint32(len(encodedMsg))
	msgLen := encode32(size)
	toSend := append(msgLen, encodedMsg...)

	return socket.Send(toSend)
}

func ReceiveWithTimeout(socket *socket.TCPConnection, t time.Duration) (Encodable, error) {
	buffer := make([]byte, 4)
	err := socket.ReceiveWithTimeout(buffer, t)
	if err != nil {
		return nil, err
	}
	msgLen, _ := decode32(buffer)
	buffer = make([]byte, msgLen)
	err = socket.ReceiveWithTimeout(buffer, t)
	if err != nil {
		return nil, err
	}
	decoded, err := Decode(buffer)

	return decoded, err
}

func Receive(socket *socket.TCPConnection) (Encodable, error) {
	buffer := make([]byte, 4)
	err := socket.Receive(buffer)
	if err != nil {
		return nil, err
	}
	msgLen, _ := decode32(buffer)
	buffer = make([]byte, msgLen)
	err = socket.Receive(buffer)
	if err != nil {
		return nil, err
	}
	decoded, err := Decode(buffer)

	return decoded, err
}

func appendSlices(slices [][]byte) []byte {
	size := 0
	for _, slice := range slices {
		size += len(slice)
	}

	result := make([]byte, size)

	start := 0
	for _, slice := range slices {
		start += copy(result[start:], slice)
	}

	return result
}
