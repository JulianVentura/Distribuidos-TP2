package utils

import (
	"strings"
	"unsafe"
)

type MessageParser struct {
	separator string
}

func NewParser() MessageParser {
	return MessageParser{
		separator: string(rune(0x1f)),
	}
}

func CustomParser(separator rune) MessageParser {
	return MessageParser{
		separator: string(separator),
	}
}

func (self *MessageParser) Read(input string) []string {
	splits := strings.Split(input, self.separator)
	return cloneSplit(splits)
}

func (self *MessageParser) ReadN(input string, n int) []string {
	splits := strings.SplitN(input, self.separator, n)
	return cloneSplit(splits)
}

func (self *MessageParser) Write(input []string) string {
	return strings.Join(input, self.separator)
}

//Used to deep copy strings
func Clone(s string) string {
	b := make([]byte, len(s))
	copy(b, s)
	return *(*string)(unsafe.Pointer(&b))
}

//We do this in order to help Golang's garbage collector to free unnused memmory
func cloneSplit(split []string) []string {
	if len(split) < 2 {
		return split
	}
	r := make([]string, len(split))
	for i, s := range split {
		r[i] = Clone(s)
	}

	return r
}
