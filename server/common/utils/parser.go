package utils

import (
	"strings"
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
	return strings.Split(input, self.separator)
}

func (self *MessageParser) ReadN(input string, n int) []string {
	return strings.SplitN(input, self.separator, n)
}

func (self *MessageParser) Write(input []string) string {
	return strings.Join(input, self.separator)
}
