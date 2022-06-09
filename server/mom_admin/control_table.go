package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type TableEntry struct {
	writers  uint
	readers  uint
	callback func(uint)
}

type ControlTable struct {
	table map[string]*TableEntry
}

func NewControlTable() ControlTable {
	return ControlTable{
		table: make(map[string]*TableEntry),
	}
}

func (self *ControlTable) AddNewWriter(name string, callback func(uint)) {
	//If it already exists, we overwrite it
	entry, exists := self.table[name]
	if exists {
		entry.writers += 1
		if entry.callback == nil {
			entry.callback = callback
		}
		self.table[name] = entry
		return
	}

	self.table[name] = &TableEntry{
		writers:  1,
		readers:  0,
		callback: callback,
	}
}

func (self *ControlTable) AddNewReader(name string) {
	//If it already exists, we overwrite it
	_, exists := self.table[name]
	if exists {
		self.table[name].readers += 1
		return
	}

	self.table[name] = &TableEntry{
		writers:  0,
		readers:  1,
		callback: nil,
	}
}

func (self *ControlTable) DecreaseWriter(name string) error {
	entry, exists := self.table[name]
	if !exists {
		return fmt.Errorf("%v does not exist in control table", name)
	}

	if entry.writers == 0 {
		//We don't have to do anything
		return nil
	}

	entry.writers -= 1

	if entry.writers == 0 {
		if entry.callback != nil {
			entry.callback(entry.readers)
		}
	}

	self.table[name] = entry

	return nil
}

func (self *ControlTable) AnyPendingFinish() bool {
	for k, v := range self.table {
		if v.writers > 0 {
			log.Infof("Queue %v hasn't finished yet", k)
			return true
		}
	}

	return false
}
