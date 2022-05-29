package utils

import (
	"fmt"
	"strings"
)

type MessageParser struct {
	expected_size int
	separator     string
}

func NewParser(expected_size uint) MessageParser {
	return MessageParser{
		expected_size: int(expected_size),
		separator:     string(rune(0x1f)),
	}
}

func CustomParser(separator rune, expected_size uint) MessageParser {
	return MessageParser{
		expected_size: int(expected_size),
		separator:     string(separator),
	}
}

func (self *MessageParser) Read(input string) ([]string, error) {

	splits := strings.Split(input, self.separator)
	if len(splits) != int(self.expected_size) {
		return nil, fmt.Errorf("Size is different than expected")
	}
	return splits, nil
}

func (self *MessageParser) ReadN(input string, n int) []string {
	splits := strings.SplitN(input, self.separator, n)
	return splits
}

func (self *MessageParser) Write(input []string) string {
	return strings.Join(input, self.separator)
}
