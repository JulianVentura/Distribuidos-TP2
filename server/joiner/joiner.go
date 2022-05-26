package main

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Joiner struct {
	table map[string]string
}

func NewJoiner() Joiner {

	return Joiner{
		table: make(map[string]string, 100), //Initial value
	}
}

func (self *Joiner) Add(input string) {
	log.Debugf("Received: %v", input)

	splits := strings.SplitN(input, ",", 2)
	id := splits[0]
	body := ""
	if len(splits) > 1 {
		body = splits[1]
	}
	self.table[id] = body
}

func (self *Joiner) Join(input string) (string, error) {
	splits := strings.SplitN(input, ",", 2)
	id := splits[0]

	body, exists := self.table[id]
	if !exists {
		return "", fmt.Errorf("entry do not exist")
	}

	result := make([]string, 0, 3)
	result = append(result, id)
	if len(body) > 1 {
		result = append(result, body)
	}
	if len(splits) > 1 {
		result = append(result, splits[1])
	}

	return strings.Join(result, ","), nil
}

func test_function() {
	to_add := []string{
		"id1,0.23,a",
		"id2",
		"id3,0.23",
	}

	to_join := []string{
		"id1",
		"id1,b",
		"id5,0.5,a",
		"id2,0.23,a",
		"id3,-0.5,a,b",
	}

	joiner := NewJoiner()

	for _, line := range to_add {
		joiner.Add(line)
	}

	for _, line := range to_join {
		r, err := joiner.Join(line)
		if err != nil {
			fmt.Printf("%v: X\n", line)
		} else {
			fmt.Printf("%v\n", r)
		}
	}
}
